package v2alpha2

import (
	"context"
	"encoding/binary"
	"fmt"
	"hash"
	"hash/fnv"
	"sort"

	"github.com/davecgh/go-spew/spew"
	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func isExistReplicant(instance *appsv2alpha2.EMQX) bool {
	return instance.Spec.ReplicantTemplate != nil && instance.Spec.ReplicantTemplate.Spec.Replicas != nil && *instance.Spec.ReplicantTemplate.Spec.Replicas > 0
}

func getPodMap(ctx context.Context, client client.Client, opts ...client.ListOption) map[types.UID][]*corev1.Pod {
	podList := &corev1.PodList{}
	_ = client.List(ctx, podList, opts...)

	replicaSetList := &appsv1.ReplicaSetList{}
	_ = client.List(ctx, replicaSetList, opts...)
	// Create a map from ReplicaSet UID to ReplicaSet.
	rsMap := make(map[types.UID][]*corev1.Pod, len(replicaSetList.Items))
	for _, rs := range replicaSetList.Items {
		rsMap[rs.UID] = []*corev1.Pod{}
	}
	for _, p := range podList.Items {
		// Do not ignore inactive Pods because Recreate replicaSets need to verify that no
		// Pods from older versions are running before spinning up new Pods.
		pod := p.DeepCopy()
		controllerRef := metav1.GetControllerOf(pod)
		if controllerRef == nil {
			continue
		}
		// Only append if we care about this UID.
		if _, ok := rsMap[controllerRef.UID]; ok {
			rsMap[controllerRef.UID] = append(rsMap[controllerRef.UID], pod)
		}
	}
	return rsMap
}

func getReplicaSetList(ctx context.Context, client client.Client, opts ...client.ListOption) []*appsv1.ReplicaSet {
	list := &appsv1.ReplicaSetList{}
	_ = client.List(ctx, list, opts...)
	return handlerReplicaSetList(list)
}

func getEventList(ctx context.Context, clientSet *kubernetes.Clientset, obj client.Object) []*corev1.Event {
	// https://github.com/kubernetes-sigs/kubebuilder/issues/547#issuecomment-450772300
	eventList, _ := clientSet.CoreV1().Events(obj.GetNamespace()).List(context.Background(), metav1.ListOptions{
		FieldSelector: "involvedObject.name=" + obj.GetName(),
	})
	return handlerEventList(eventList)
}

func handlerReplicaSetList(list *appsv1.ReplicaSetList) []*appsv1.ReplicaSet {
	rsList := []*appsv1.ReplicaSet{}
	for _, rs := range list.Items {
		if rs.Status.Replicas != 0 && rs.Status.ReadyReplicas == rs.Status.Replicas {
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
