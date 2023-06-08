package v2alpha2

import (
	"context"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
		// Do not ignore inactive Pods because Recreate Deployments need to verify that no
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

	dList := getDeploymentList(ctx, client, opts...)
	dMap := make(map[types.UID][]*corev1.Pod, len(dList))
	for _, d := range dList {
		dMap[d.UID] = []*corev1.Pod{}
	}
	for _, rs := range replicaSetList.Items {
		controllerRef := metav1.GetControllerOf(rs.DeepCopy())
		if controllerRef == nil {
			continue
		}
		// Only append if we care about this UID.
		if _, ok := dMap[controllerRef.UID]; ok {
			dMap[controllerRef.UID] = rsMap[rs.UID]
		}
	}

	return dMap
}

func getDeploymentList(ctx context.Context, client client.Client, opts ...client.ListOption) []*appsv1.Deployment {
	deploymentList := &appsv1.DeploymentList{}
	_ = client.List(ctx, deploymentList, opts...)
	return handlerDeploymentList(deploymentList)
}

func getEventList(ctx context.Context, clientSet *kubernetes.Clientset, obj client.Object) []*corev1.Event {
	// https://github.com/kubernetes-sigs/kubebuilder/issues/547#issuecomment-450772300
	eventList, _ := clientSet.CoreV1().Events(obj.GetNamespace()).List(context.Background(), metav1.ListOptions{
		FieldSelector: "involvedObject.name=" + obj.GetName(),
	})
	return handlerEventList(eventList)
}

func handlerDeploymentList(list *appsv1.DeploymentList) []*appsv1.Deployment {
	dList := []*appsv1.Deployment{}
	for _, d := range list.Items {
		if d.Status.Replicas != 0 && d.Status.ReadyReplicas == d.Status.Replicas {
			dList = append(dList, d.DeepCopy())
		}
	}
	sort.Sort(DeploymentsByCreationTimestamp(dList))
	return dList
}

func handlerEventList(list *corev1.EventList) []*corev1.Event {
	eList := []*corev1.Event{}
	for _, e := range list.Items {
		if e.Reason == "ScalingReplicaSet" {
			eList = append(eList, e.DeepCopy())
		}
	}
	sort.Sort(EventsByLastTimestamp(eList))
	return eList
}

// DeploymentsByCreationTimestamp sorts a list of Deployment by creation timestamp, using their names as a tie breaker.
type DeploymentsByCreationTimestamp []*appsv1.Deployment

func (o DeploymentsByCreationTimestamp) Len() int      { return len(o) }
func (o DeploymentsByCreationTimestamp) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o DeploymentsByCreationTimestamp) Less(i, j int) bool {
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
