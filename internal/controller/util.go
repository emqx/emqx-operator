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

	emperror "emperror.dev/errors"
	"github.com/cisco-open/k8s-objectmatcher/patch"
	"github.com/davecgh/go-spew/spew"
	crd "github.com/emqx/emqx-operator/api/v3alpha1"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	"github.com/tidwall/gjson"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// JustCheckPodTemplate will check only the differences between the podTemplate of the two statefulSets
func justCheckPodTemplate() patch.CalculateOption {
	getPodTemplate := func(obj []byte) ([]byte, error) {
		podTemplateSpecJson := gjson.GetBytes(obj, "spec.template")
		podTemplateSpec := &corev1.PodTemplateSpec{}
		_ = json.Unmarshal([]byte(podTemplateSpecJson.String()), podTemplateSpec)

		// Remove the podTemplateHashLabelKey from the podTemplateSpec
		delete(podTemplateSpec.Labels, crd.LabelPodTemplateHash)

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

func ignoreField(path []string) patch.CalculateOption {
	return func(current, modified []byte) ([]byte, []byte, error) {
		current, err := deleteFieldPath(current, path)
		if err != nil {
			return []byte{}, []byte{}, emperror.Wrap(err, "could not delete the field from current byte sequence")
		}

		modified, err = deleteFieldPath(modified, path)
		if err != nil {
			return []byte{}, []byte{}, emperror.Wrap(err, "could not delete the field from modified byte sequence")
		}

		return current, modified, nil
	}
}

func deleteFieldPath(obj []byte, path []string) ([]byte, error) {
	var objectMap map[string]interface{}
	err := json.Unmarshal(obj, &objectMap)
	if err != nil {
		return []byte{}, emperror.Wrap(err, "could not unmarshal byte sequence")
	}
	pathLen := len(path)
	if pathLen == 0 {
		return obj, nil
	}
	innerObject := objectMap
	for _, k := range path[:pathLen-1] {
		innerNext, ok := innerObject[k]
		if !ok {
			return obj, nil
		}
		switch innerNext := innerNext.(type) {
		case map[string]interface{}:
			innerObject = innerNext
		default:
			return obj, nil
		}
	}
	delete(innerObject, path[pathLen-1])
	obj, err = json.Marshal(objectMap)
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

// sortByOrdinal sorts pods by their StatefulSet ordinal (numeric suffix) ascending.
// Pods whose names do not end in a number get ordinal -1 and sort first.
func sortByOrdinal(list []*corev1.Pod) {
	slices.SortFunc(list, func(a, b *corev1.Pod) int {
		return cmp.Compare(util.PodOrdinal(a.Name), util.PodOrdinal(b.Name))
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

func parseNodeName(s string, instance *crd.EMQX) *nodeName {
	// Example: emqx@emqx-core-557c8b7684-0.emqx-headless.default.svc.cluster.local
	// Example: emqx@10.244.0.23
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
