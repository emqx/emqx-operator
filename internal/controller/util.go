package controller

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
	appsv2beta1 "github.com/emqx/emqx-operator/api/v2beta1"
	"github.com/tidwall/gjson"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func listPodsManagedBy(ctx context.Context, k8sClient client.Client, instance *appsv2beta1.EMQX, uid types.UID) []*corev1.Pod {
	result := []*corev1.Pod{}
	podList := &corev1.PodList{}
	labels := appsv2beta1.DefaultLabels(instance)
	_ = k8sClient.List(ctx, podList,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(labels),
	)
	for _, pod := range podList.Items {
		controllerRef := metav1.GetControllerOf(&pod)
		if controllerRef != nil && controllerRef.UID == uid {
			result = append(result, pod.DeepCopy())
		}
	}
	return result
}

func getStateFulSetList(ctx context.Context, k8sClient client.Client, instance *appsv2beta1.EMQX) (updateSts, currentSts *appsv1.StatefulSet, oldStsList []*appsv1.StatefulSet) {
	list := &appsv1.StatefulSetList{}
	_ = k8sClient.List(ctx, list,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(appsv2beta1.DefaultCoreLabels(instance)),
	)
	for _, sts := range list.Items {
		if hash, ok := sts.Labels[appsv2beta1.LabelsPodTemplateHashKey]; ok {
			if hash == instance.Status.CoreNodesStatus.UpdateRevision {
				updateSts = sts.DeepCopy()
			}
			if hash == instance.Status.CoreNodesStatus.CurrentRevision {
				currentSts = sts.DeepCopy()
			}
			if hash != instance.Status.CoreNodesStatus.UpdateRevision && hash != instance.Status.CoreNodesStatus.CurrentRevision {
				oldStsList = append(oldStsList, sts.DeepCopy())
			}
		}
	}

	sort.Sort(StatefulSetsByCreationTimestamp(oldStsList))
	return
}

func getReplicaSetList(ctx context.Context, k8sClient client.Client, instance *appsv2beta1.EMQX) (updateRs, currentRs *appsv1.ReplicaSet, oldRsList []*appsv1.ReplicaSet) {
	labels := appsv2beta1.DefaultReplicantLabels(instance)

	list := &appsv1.ReplicaSetList{}
	_ = k8sClient.List(ctx, list,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(labels),
	)

	for _, rs := range list.Items {
		if hash, ok := rs.Labels[appsv2beta1.LabelsPodTemplateHashKey]; ok {
			if hash == instance.Status.ReplicantNodesStatus.UpdateRevision {
				updateRs = rs.DeepCopy()
			}
			if hash == instance.Status.ReplicantNodesStatus.CurrentRevision {
				currentRs = rs.DeepCopy()
			}
			if hash != instance.Status.ReplicantNodesStatus.UpdateRevision && hash != instance.Status.ReplicantNodesStatus.CurrentRevision {
				oldRsList = append(oldRsList, rs.DeepCopy())
			}
		}
	}
	sort.Sort(ReplicaSetsByCreationTimestamp(oldRsList))
	return
}

func getEventList(ctx context.Context, clientSet *kubernetes.Clientset, obj client.Object) []*corev1.Event {
	// https://github.com/kubernetes-sigs/kubebuilder/issues/547#issuecomment-450772300
	eventList, _ := clientSet.CoreV1().Events(obj.GetNamespace()).List(ctx, metav1.ListOptions{
		FieldSelector: "involvedObject.name=" + obj.GetName(),
	})
	return handlerEventList(eventList)
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

func updatePodCondition(
	ctx context.Context,
	k8sClient client.Client,
	pod *corev1.Pod,
	condition corev1.PodCondition,
) error {
	patchBytes, _ := json.Marshal(corev1.Pod{
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{condition},
		},
	})
	patch := client.RawPatch(types.StrategicMergePatchType, patchBytes)
	return k8sClient.Status().Patch(ctx, pod, patch)
}

func checkInitialDelaySecondsReady(instance *appsv2beta1.EMQX) bool {
	_, condition := instance.Status.GetCondition(appsv2beta1.Available)
	if condition == nil || condition.Type != appsv2beta1.Available {
		return false
	}
	delay := time.Since(condition.LastTransitionTime.Time).Seconds()
	return int32(delay) > instance.Spec.UpdateStrategy.InitialDelaySeconds
}

func checkWaitTakeoverReady(instance *appsv2beta1.EMQX, eList []*corev1.Event) bool {
	if len(eList) == 0 {
		return true
	}

	lastEvent := eList[len(eList)-1]
	delay := time.Since(lastEvent.LastTimestamp.Time).Seconds()
	return int32(delay) > instance.Spec.UpdateStrategy.EvacuationStrategy.WaitTakeover
}

// JustCheckPodTemplate will check only the differences between the podTemplate of the two statefulSets
func justCheckPodTemplate() patch.CalculateOption {
	getPodTemplate := func(obj []byte) ([]byte, error) {
		podTemplateSpecJson := gjson.GetBytes(obj, "spec.template")
		podTemplateSpec := &corev1.PodTemplateSpec{}
		_ = json.Unmarshal([]byte(podTemplateSpecJson.String()), podTemplateSpec)

		// Remove the podTemplateHashLabelKey from the podTemplateSpec
		if _, ok := podTemplateSpec.Labels[appsv2beta1.LabelsPodTemplateHashKey]; ok {
			podTemplateSpec.Labels = appsv2beta1.CloneAndRemoveLabel(podTemplateSpec.Labels, appsv2beta1.LabelsPodTemplateHashKey)
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

// PodsByCreationTimestamp sorts a list of Pod by creation timestamp, using their names as a tie breaker.
type PodsByCreationTimestamp []*corev1.Pod

func (o PodsByCreationTimestamp) Len() int      { return len(o) }
func (o PodsByCreationTimestamp) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o PodsByCreationTimestamp) Less(i, j int) bool {
	if o[i].CreationTimestamp.Equal(&o[j].CreationTimestamp) {
		return o[i].Name < o[j].Name
	}
	return o[i].CreationTimestamp.Before(&o[j].CreationTimestamp)
}

// PodsByNameOlder sorts a list of Pod by size in descending order, using their creation timestamp or name as a tie breaker.
// By using the creation timestamp, this sorts from old to new replica sets.
type PodsByNameOlder []*corev1.Pod

func (o PodsByNameOlder) Len() int      { return len(o) }
func (o PodsByNameOlder) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o PodsByNameOlder) Less(i, j int) bool {
	if o[i].Name == o[j].Name {
		return PodsByCreationTimestamp(o).Less(i, j)
	}
	return o[i].Name > o[j].Name
}

// PodsByNameNewer sorts a list of Pod by size in descending order, using their creation timestamp or name as a tie breaker.
// By using the creation timestamp, this sorts from new to old replica sets.
type PodsByNameNewer []*corev1.Pod

func (o PodsByNameNewer) Len() int      { return len(o) }
func (o PodsByNameNewer) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o PodsByNameNewer) Less(i, j int) bool {
	if o[i].Name == o[j].Name {
		return PodsByCreationTimestamp(o).Less(j, i)
	}
	return o[i].Name > o[j].Name
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
