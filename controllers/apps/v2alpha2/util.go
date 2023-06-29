package v2alpha2

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash"
	"hash/fnv"
	"sort"
	"time"

	emperror "emperror.dev/errors"
	"github.com/cisco-open/k8s-objectmatcher/patch"
	"github.com/davecgh/go-spew/spew"
	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	"github.com/tidwall/gjson"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func isExistReplicant(instance *appsv2alpha2.EMQX) bool {
	return instance.Spec.ReplicantTemplate != nil && instance.Spec.ReplicantTemplate.Spec.Replicas != nil && *instance.Spec.ReplicantTemplate.Spec.Replicas > 0
}

// func getPodMap(ctx context.Context, client client.Client, opts ...client.ListOption) map[types.UID][]*corev1.Pod {
// 	podList := &corev1.PodList{}
// 	_ = client.List(ctx, podList, opts...)

// 	replicaSetList := &appsv1.ReplicaSetList{}
// 	_ = client.List(ctx, replicaSetList, opts...)
// 	// Create a map from ReplicaSet UID to ReplicaSet.
// 	rsMap := make(map[types.UID][]*corev1.Pod, len(replicaSetList.Items))
// 	for _, rs := range replicaSetList.Items {
// 		rsMap[rs.UID] = []*corev1.Pod{}
// 	}
// 	for _, p := range podList.Items {
// 		// Do not ignore inactive Pods because Recreate replicaSets need to verify that no
// 		// Pods from older versions are running before spinning up new Pods.
// 		pod := p.DeepCopy()
// 		controllerRef := metav1.GetControllerOf(pod)
// 		if controllerRef == nil {
// 			continue
// 		}
// 		// Only append if we care about this UID.
// 		if _, ok := rsMap[controllerRef.UID]; ok {
// 			rsMap[controllerRef.UID] = append(rsMap[controllerRef.UID], pod)
// 		}
// 	}
// 	return rsMap
// }

func canBeScaledDown(instance *appsv2alpha2.EMQX, conditionType string, eList []*corev1.Event) bool {
	var initialDelaySecondsReady bool
	var waitTakeover bool

	_, condition := instance.Status.GetCondition(conditionType)
	if condition != nil && condition.Status == metav1.ConditionTrue {
		delay := time.Since(condition.LastTransitionTime.Time).Seconds()
		if int32(delay) > instance.Spec.UpdateStrategy.InitialDelaySeconds {
			initialDelaySecondsReady = true
		}
	}

	if len(eList) == 0 {
		waitTakeover = true
		return initialDelaySecondsReady && waitTakeover
	}

	lastEvent := eList[len(eList)-1]
	delay := time.Since(lastEvent.LastTimestamp.Time).Seconds()
	if int32(delay) > instance.Spec.UpdateStrategy.EvacuationStrategy.WaitTakeover {
		waitTakeover = true
	}

	return initialDelaySecondsReady && waitTakeover
}

func getStateFulSetList(ctx context.Context, k8sClient client.Client, instance *appsv2alpha2.EMQX) (currentSts *appsv1.StatefulSet, oldStsList []*appsv1.StatefulSet) {
	list := &appsv1.StatefulSetList{}
	_ = k8sClient.List(ctx, list,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(instance.Spec.CoreTemplate.Labels),
	)
	for _, sts := range list.Items {
		if hash, ok := sts.Labels[appsv2alpha2.PodTemplateHashLabelKey]; ok && hash == instance.Status.CoreNodesStatus.CurrentRevision {
			currentSts = sts.DeepCopy()
		} else {
			if *sts.Spec.Replicas != 0 && sts.Status.ReadyReplicas == sts.Status.Replicas {
				oldStsList = append(oldStsList, sts.DeepCopy())
			}
		}
	}

	sort.Sort(StatefulSetsByCreationTimestamp(oldStsList))
	return
}

func getReplicaSetList(ctx context.Context, k8sClient client.Client, instance *appsv2alpha2.EMQX) (currentRs *appsv1.ReplicaSet, oldRsList []*appsv1.ReplicaSet) {
	list := &appsv1.ReplicaSetList{}
	_ = k8sClient.List(ctx, list,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
	)
	for _, rs := range list.Items {
		if hash, ok := rs.Labels[appsv2alpha2.PodTemplateHashLabelKey]; ok && hash == instance.Status.ReplicantNodesStatus.CurrentRevision {
			currentRs = rs.DeepCopy()
		} else {
			if *rs.Spec.Replicas != 0 && rs.Status.ReadyReplicas == rs.Status.Replicas {
				oldRsList = append(oldRsList, rs.DeepCopy())
			}
		}
	}
	sort.Sort(ReplicaSetsByCreationTimestamp(oldRsList))
	return
}

func getEventList(ctx context.Context, clientSet *kubernetes.Clientset, obj client.Object) []*corev1.Event {
	// https://github.com/kubernetes-sigs/kubebuilder/issues/547#issuecomment-450772300
	eventList, _ := clientSet.CoreV1().Events(obj.GetNamespace()).List(context.Background(), metav1.ListOptions{
		FieldSelector: "involvedObject.name=" + obj.GetName(),
	})
	return handlerEventList(eventList)
}

func handlerStatefulSetList(list *appsv1.StatefulSetList) []*appsv1.StatefulSet {
	stsList := []*appsv1.StatefulSet{}

	for _, sts := range list.Items {
		if *sts.Spec.Replicas > 0 && sts.Status.ReadyReplicas == sts.Status.Replicas {
			stsList = append(stsList, sts.DeepCopy())
		}
	}

	sort.Sort(StatefulSetsByCreationTimestamp(stsList))
	return stsList
}

func handlerReplicaSetList(list *appsv1.ReplicaSetList) []*appsv1.ReplicaSet {
	rsList := []*appsv1.ReplicaSet{}
	for _, rs := range list.Items {
		if *rs.Spec.Replicas != 0 && rs.Status.ReadyReplicas == rs.Status.Replicas {
			rsList = append(rsList, rs.DeepCopy())
		}
	}
	sort.Sort(ReplicaSetsByCreationTimestamp(rsList))
	return rsList
}

func handlerEventList(list *corev1.EventList) []*corev1.Event {
	eList := []*corev1.Event{}
	for _, e := range list.Items {
		if e.Reason == "SuccessfulDelete" {
			eList = append(eList, e.DeepCopy())
		}
	}
	sort.Sort(EventsByLastTimestamp(eList))
	return eList
}

// JustCheckPodTemplate will check only the differences between the podTemplate of the two statefulSets
func justCheckPodTemplate() patch.CalculateOption {
	getPodTemplate := func(obj []byte) ([]byte, error) {
		podTemplateSpecJson := gjson.GetBytes(obj, "spec.template")
		podTemplateSpec := &corev1.PodTemplateSpec{}
		_ = json.Unmarshal([]byte(podTemplateSpecJson.String()), podTemplateSpec)

		// Remove the podTemplateHashLabelKey from the podTemplateSpec
		if _, ok := podTemplateSpec.Labels[appsv2alpha2.PodTemplateHashLabelKey]; ok {
			podTemplateSpec.Labels = appsv2alpha2.CloneAndRemoveLabel(podTemplateSpec.Labels, appsv2alpha2.PodTemplateHashLabelKey)
		}

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

// StatefulSetsByCreationTimestamp sorts a list of StatefulSet by creation timestamp, using their names as a tie breaker.
type StatefulSetsByCreationTimestamp []*appsv1.StatefulSet

func (o StatefulSetsByCreationTimestamp) Len() int      { return len(o) }
func (o StatefulSetsByCreationTimestamp) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o StatefulSetsByCreationTimestamp) Less(i, j int) bool {
	if o[i].CreationTimestamp.Equal(&o[j].CreationTimestamp) {
		return o[i].Name < o[j].Name
	}
	return o[i].CreationTimestamp.Before(&o[j].CreationTimestamp)
}

// ReplicaSetsByCreationTimestamp sorts a list of ReplicaSet by creation timestamp, using their names as a tie breaker.
type ReplicaSetsByCreationTimestamp []*appsv1.ReplicaSet

func (o ReplicaSetsByCreationTimestamp) Len() int      { return len(o) }
func (o ReplicaSetsByCreationTimestamp) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o ReplicaSetsByCreationTimestamp) Less(i, j int) bool {
	if o[i].CreationTimestamp.Equal(&o[j].CreationTimestamp) {
		return o[i].Name < o[j].Name
	}
	return o[i].CreationTimestamp.Before(&o[j].CreationTimestamp)
}

// EventsByLastTimestamp sorts a list of Event by last timestamp, using their creation timestamp as a tie breaker.
type EventsByLastTimestamp []*corev1.Event

func (o EventsByLastTimestamp) Len() int      { return len(o) }
func (o EventsByLastTimestamp) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o EventsByLastTimestamp) Less(i, j int) bool {
	if o[i].LastTimestamp.Equal(&o[j].LastTimestamp) {
		return o[i].CreationTimestamp.Second() < o[j].CreationTimestamp.Second()
	}
	return o[i].LastTimestamp.Before(&o[j].LastTimestamp)
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
	printer.Fprintf(hasher, "%#v", objectToWrite)
}
