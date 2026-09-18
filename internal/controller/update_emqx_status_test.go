package controller

import (
	"testing"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestManagedWorkloadReadiness(t *testing.T) {

	emqx := &crd.EMQX{}
	emqx.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(2))
	emqx.Spec.ReplicantTemplate.Spec.Replicas = ptr.To(int32(2))
	emqx.Status.ReplicantNodesStatus.UpdateRevision = "update"
	emqx.Status.ClusterNodes = []crd.EMQXNode{
		{Name: "emqx@core-0", PodName: "core-0", Role: "core", Status: "running"},
		{Name: "emqx@core-1", PodName: "core-1", Role: "core", Status: "running"},
		{Name: "emqx@replicant-0", PodName: "replicant-0", Role: "replicant", Status: "running"},
		{Name: "emqx@replicant-1", PodName: "replicant-1", Role: "replicant", Status: "running"},
	}

	mkState := func() *reconcileState {
		coreSet := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{UID: "core"},
			Status:     appsv1.StatefulSetStatus{Replicas: 2, ReadyReplicas: 2, UpdatedReplicas: 2},
		}
		replicantSet := &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{UID: "replicant", Labels: map[string]string{crd.LabelPodTemplateHash: "update"}},
			Status:     appsv1.ReplicaSetStatus{Replicas: 2, ReadyReplicas: 2},
		}
		return &reconcileState{
			coreSets:      []*appsv1.StatefulSet{coreSet},
			replicantSets: []*appsv1.ReplicaSet{replicantSet},
			pods: []*corev1.Pod{
				{ObjectMeta: metav1.ObjectMeta{Name: "core-0", OwnerReferences: ownerReferences(coreSet)}},
				{ObjectMeta: metav1.ObjectMeta{Name: "core-1", OwnerReferences: ownerReferences(coreSet)}},
				{ObjectMeta: metav1.ObjectMeta{Name: "replicant-0", OwnerReferences: ownerReferences(replicantSet)}},
				{ObjectMeta: metav1.ObjectMeta{Name: "replicant-1", OwnerReferences: ownerReferences(replicantSet)}},
			},
		}
	}

	t.Run("foreign nodes do not block readiness", func(t *testing.T) {
		instance := emqx.DeepCopy()
		state := mkState()
		instance.Status.ClusterNodes = append(instance.Status.ClusterNodes,
			crd.EMQXNode{Name: "emqx@foreign", Role: "core", Status: "running"},
			crd.EMQXNode{Name: "emqx@silent", Role: "replicant", Status: "unreachable"})
		require.True(t, evaluateCoresReady(state, instance))
		require.True(t, evaluateReplicantsReady(state, instance))
	})

	t.Run("foreign node cannot replace a missing core report", func(t *testing.T) {
		instance := emqx.DeepCopy()
		state := mkState()
		state.pods = append(state.pods, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foreign", Labels: crd.CoreLabels()}})
		instance.Status.ClusterNodes[0].PodName = "foreign"
		instance.Status.CoreNodesStatus.ReadyReplicas = 2
		require.False(t, evaluateCoresReady(state, instance))
	})

	t.Run("duplicate report cannot replace a missing core report", func(t *testing.T) {
		instance := emqx.DeepCopy()
		state := mkState()
		instance.Status.ClusterNodes[0] = instance.Status.ClusterNodes[1]
		require.False(t, evaluateCoresReady(state, instance))
	})

	t.Run("missing replicant report despite ready summary", func(t *testing.T) {
		instance := emqx.DeepCopy()
		state := mkState()
		instance.Status.ClusterNodes = instance.Status.ClusterNodes[:3]
		instance.Status.ReplicantNodesStatus.ReadyReplicas = 2
		require.False(t, evaluateReplicantsReady(state, instance))
	})

	t.Run("unreachable core with unknown role", func(t *testing.T) {
		instance := emqx.DeepCopy()
		state := mkState()
		instance.Status.ClusterNodes[0].Status = "unreachable"
		instance.Status.ClusterNodes[0].Role = ""
		require.False(t, evaluateCoresReady(state, instance))
	})

	t.Run("core Kubernetes readiness incomplete", func(t *testing.T) {
		instance := emqx.DeepCopy()
		state := mkState()
		state.coreSets[0].Status.ReadyReplicas = 1
		require.False(t, evaluateCoresReady(state, instance))
	})

	t.Run("core Kubernetes revision incomplete", func(t *testing.T) {
		instance := emqx.DeepCopy()
		state := mkState()
		state.coreSets[0].Status.UpdatedReplicas = 1
		require.False(t, evaluateCoresReady(state, instance))
	})

	t.Run("replicant Kubernetes readiness incomplete", func(t *testing.T) {
		instance := emqx.DeepCopy()
		state := mkState()
		state.replicantSets[0].Status.ReadyReplicas = 1
		require.False(t, evaluateReplicantsReady(state, instance))
	})

	t.Run("old ReplicaSet still has replicas", func(t *testing.T) {
		instance := emqx.DeepCopy()
		state := mkState()
		state.replicantSets = append(state.replicantSets, &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{UID: "old"}, Status: appsv1.ReplicaSetStatus{Replicas: 1},
		})
		require.False(t, evaluateReplicantsReady(state, instance))
	})

	t.Run("old Pod remains after leaving EMQX", func(t *testing.T) {
		instance := emqx.DeepCopy()
		state := mkState()
		old := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{UID: "old"}}
		state.replicantSets = append(state.replicantSets, old)
		state.pods = append(state.pods, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
			Name: "old-replicant", OwnerReferences: ownerReferences(old),
		}})
		require.False(t, evaluateReplicantsReady(state, instance))
	})

	t.Run("deleting core still reports running", func(t *testing.T) {
		instance := emqx.DeepCopy()
		state := mkState()
		state.pods[0].DeletionTimestamp = ptr.To(metav1.Now())
		require.False(t, evaluateCoresReady(state, instance))
	})

	t.Run("replicants scaled to zero", func(t *testing.T) {
		instance := emqx.DeepCopy()
		state := mkState()
		instance.Spec.ReplicantTemplate.Spec.Replicas = ptr.To(int32(0))
		state.replicantSets[0].Status = appsv1.ReplicaSetStatus{}
		state.pods = state.pods[:2]
		instance.Status.ClusterNodes = instance.Status.ClusterNodes[:2]
		require.True(t, evaluateReplicantsReady(state, instance))
	})
}
