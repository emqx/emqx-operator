package controller

import (
	"cmp"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash"
	"hash/fnv"
	"slices"
	"strings"
	"time"

	emperror "emperror.dev/errors"
	"github.com/cisco-open/k8s-objectmatcher/patch"
	"github.com/davecgh/go-spew/spew"
	crdv2 "github.com/emqx/emqx-operator/api/v2"
	"github.com/tidwall/gjson"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func checkInitialDelaySecondsReady(instance *crdv2.EMQX) bool {
	_, condition := instance.Status.GetCondition(crdv2.Available)
	if condition == nil || condition.Status != metav1.ConditionTrue {
		return false
	}
	delay := time.Since(condition.LastTransitionTime.Time).Seconds()
	return delay > float64(instance.Spec.UpdateStrategy.InitialDelaySeconds)
}

// JustCheckPodTemplate will check only the differences between the podTemplate of the two statefulSets
func justCheckPodTemplate() patch.CalculateOption {
	getPodTemplate := func(obj []byte) ([]byte, error) {
		podTemplateSpecJson := gjson.GetBytes(obj, "spec.template")
		podTemplateSpec := &corev1.PodTemplateSpec{}
		_ = json.Unmarshal([]byte(podTemplateSpecJson.String()), podTemplateSpec)

		// Remove the podTemplateHashLabelKey from the podTemplateSpec
		delete(podTemplateSpec.Labels, crdv2.LabelPodTemplateHash)

		emptyRs := &appsv1.ReplicaSet{}
		emptyRs.Spec.Template = *podTemplateSpec
		return json.Marshal(emptyRs)
	}

	return func(current, modified []byte) ([]byte, []byte, error) {
		current, err := getPodTemplate(current)
		if err != nil {
			return []byte{}, []byte{}, emperror.Wrap(err, "could not get pod template field from current byte sequence")
		}

		modified, err = getPodTemplate(modified)
		if err != nil {
			return []byte{}, []byte{}, emperror.Wrap(err, "could not get pod template field from modified byte sequence")
		}

		return current, modified, nil
	}
}

// IgnoreStatefulSetReplicas will ignore the `Replicas` field of the statefulSet
func ignoreStatefulSetReplicas() patch.CalculateOption {
	return func(current, modified []byte) ([]byte, []byte, error) {
		current, err := filterStatefulSetReplicasField(current)
		if err != nil {
			return []byte{}, []byte{}, emperror.Wrap(err, "could not filter replicas field from current byte sequence")
		}

		modified, err = filterStatefulSetReplicasField(modified)
		if err != nil {
			return []byte{}, []byte{}, emperror.Wrap(err, "could not filter replicas field from modified byte sequence")
		}

		return current, modified, nil
	}
}

func filterStatefulSetReplicasField(obj []byte) ([]byte, error) {
	sts := appsv1.StatefulSet{}
	err := json.Unmarshal(obj, &sts)
	if err != nil {
		return []byte{}, emperror.Wrap(err, "could not unmarshal byte sequence")
	}
	sts.Spec.Replicas = ptr.To(int32(1))
	obj, err = json.Marshal(sts)
	if err != nil {
		return []byte{}, emperror.Wrap(err, "could not marshal byte sequence")
	}

	return obj, nil
}

func compareCreationTimestamp(a, b client.Object) int {
	atime := a.GetCreationTimestamp()
	btime := b.GetCreationTimestamp()
	cmpTime := atime.Time.Compare(btime.Time)
	// Use name as a tie breaker:
	if cmpTime == 0 {
		return cmp.Compare(a.GetName(), b.GetName())
	}
	return cmpTime
}

func compareName(a, b client.Object) int {
	cmpName := cmp.Compare(a.GetName(), b.GetName())
	// Use creation timestamp as a tie breaker:
	if cmpName == 0 {
		atime := a.GetCreationTimestamp()
		btime := b.GetCreationTimestamp()
		return atime.Time.Compare(btime.Time)
	}
	return cmpName
}

func sortByCreationTimestamp[T client.Object](list []T) {
	slices.SortFunc(list, func(a, b T) int {
		return compareCreationTimestamp(a, b)
	})
}

func sortByName[T client.Object](list []T) {
	slices.SortFunc(list, func(a, b T) int {
		return compareName(a, b)
	})
}

// ComputeHash returns a hash value calculated from pod template and
// a collisionCount to avoid hash collision. The hash will be safe encoded to
// avoid bad words.
func computeHash(template *corev1.PodTemplateSpec, collisionCount *int32) string {
	templateSpecHasher := fnv.New32a()
	deepHashObject(templateSpecHasher, *template)

	// Add collisionCount in the hash if it exists.
	if collisionCount != nil {
		collisionCountBytes := make([]byte, 8)
		binary.LittleEndian.PutUint32(collisionCountBytes, uint32(*collisionCount))
		templateSpecHasher.Write(collisionCountBytes)
	}

	return rand.SafeEncodeString(fmt.Sprint(templateSpecHasher.Sum32()))
}

// DeepHashObject writes specified object to hash using the spew library
// which follows pointers and prints actual values of the nested objects
// ensuring the hash does not change when a pointer changes.
func deepHashObject(hasher hash.Hash, objectToWrite interface{}) {
	hasher.Reset()
	printer := spew.ConfigState{
		Indent:         " ",
		SortKeys:       true,
		DisableMethods: true,
		SpewKeys:       true,
	}
	_, _ = printer.Fprintf(hasher, "%#v", objectToWrite)
}

type nodeName struct {
	name     string
	hostName string
	podName  string
}

func parseNodeName(s string, instance *crdv2.EMQX) *nodeName {
	var parsed nodeName
	nameParts := strings.Split(s, "@")
	if len(nameParts) != 2 {
		return nil
	}
	parsed.name = nameParts[0]
	parsed.hostName = nameParts[1]
	hostParts := strings.Split(nameParts[1], instance.HeadlessServiceNamespacedName().Name)
	if len(hostParts) > 1 {
		parsed.podName = strings.TrimRight(hostParts[0], ".")
	}
	return &parsed
}

func constructNodeName(podName string, instance *crdv2.EMQX) string {
	return fmt.Sprintf("emqx@%s.%s", podName, clusterDNSName(instance))
}

func clusterDNSName(instance *crdv2.EMQX) string {
	return fmt.Sprintf("%s.%s.svc.%s",
		instance.HeadlessServiceNamespacedName().Name,
		instance.Namespace,
		instance.Spec.ClusterDomain,
	)
}
