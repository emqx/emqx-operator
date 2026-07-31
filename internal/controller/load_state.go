package controller

import (
	"context"
	"reflect"
	"strings"

	emperror "emperror.dev/errors"
	crd "github.com/emqx/emqx-operator/api/v3beta1"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

type reconcileState struct {
	coreSets      []*appsv1.StatefulSet
	replicantSets []*appsv1.ReplicaSet
	pods          []*corev1.Pod
}

type reconcileStatePodFilter interface {
	passes(pod *corev1.Pod) bool
}

type podsNot struct {
	inner reconcileStatePodFilter
}

func (filter podsNot) passes(pod *corev1.Pod) bool {
	return !filter.inner.passes(pod)
}

type podsManagedBy struct {
	manager metav1.Object
}

func (filter podsManagedBy) passes(pod *corev1.Pod) bool {
	if filter.manager == nil && reflect.ValueOf(filter.manager).IsNil() {
		return false
	}
	return util.IsPodManagedBy(pod, filter.manager)
}

type podsWithRole struct {
	string
}

func (filter podsWithRole) passes(pod *corev1.Pod) bool {
	return pod.Labels[crd.LabelMriaRole] == filter.string
}

type podsAlive struct{}

func (filter podsAlive) passes(pod *corev1.Pod) bool {
	return pod.DeletionTimestamp == nil
}

type podsOnRevision struct {
	revision string
}

func (filter podsOnRevision) passes(pod *corev1.Pod) bool {
	podRevision := pod.Labels[appsv1.ControllerRevisionHashLabelKey]
	return podRevision == filter.revision
}

type podsWithCondition struct {
	cond corev1.PodConditionType
}

func (filter podsWithCondition) passes(pod *corev1.Pod) bool {
	return util.IsPodConditionTrue(pod, filter.cond)
}

type podsWithEMQXVersion struct {
	instance *crd.EMQX
	prefix   string
}

func (filter podsWithEMQXVersion) passes(pod *corev1.Pod) bool {
	node := filter.instance.Status.FindNodeByPodName(pod.Name)
	return node != nil && strings.HasPrefix(node.Version, filter.prefix)
}

func (r *reconcileState) podWithName(name string) *corev1.Pod {
	for _, pod := range r.pods {
		if pod.Name == name {
			return pod
		}
	}
	return nil
}

func (r *reconcileState) podsManagedBy(object metav1.Object) []*corev1.Pod {
	return r.listPods(podsManagedBy{object})
}

func (r *reconcileState) listPods(filters ...reconcileStatePodFilter) []*corev1.Pod {
	list := []*corev1.Pod{}
	for _, pod := range r.pods {
		passes := true
		for _, f := range filters {
			passes = passes && f.passes(pod)
		}
		if passes {
			list = append(list, pod)
		}
	}
	return list
}

// listOutdatedPods returns core StatefulSet pods whose pod template is not yet the
// desired one: anything not labeled with Status.UpdateRevision.
func (r *reconcileState) listOutdatedPods() []*corev1.Pod {
	var outdated []*corev1.Pod
	coreSet := r.coreSet()
	if coreSet == nil ||
		coreSet.Status.UpdateRevision == "" ||
		coreSet.Status.UpdateRevision == coreSet.Status.CurrentRevision {
		return outdated
	}
	return r.listPods(
		podsManagedBy{coreSet},
		podsNot{podsOnRevision{coreSet.Status.UpdateRevision}},
	)
}

// coreSet returns the single core StatefulSet, or nil if none exists.
// With the rolling update model, there is always at most one core StatefulSet.
func (r *reconcileState) coreSet() *appsv1.StatefulSet {
	if len(r.coreSets) > 0 {
		return r.coreSets[0]
	}
	return nil
}

func (r *reconcileState) partOfCoreSet(pod *corev1.Pod) bool {
	coreSet := r.coreSet()
	if coreSet == nil {
		return false
	}
	return util.IsPodManagedBy(pod, coreSet)
}

// partOfCoreSetRevision checks if a core pod's StatefulSet-assigned revision matches the specified revision.
func (r *reconcileState) partOfCoreSetRevision(pod *corev1.Pod, revision string) bool {
	coreSet := r.coreSet()
	if coreSet == nil {
		return false
	}
	if !util.IsPodManagedBy(pod, coreSet) {
		return false
	}
	podRevision := pod.Labels[appsv1.ControllerRevisionHashLabelKey]
	return podRevision == revision
}

// numCoresRevision counts number of core pods running specified StatefulSet revision.
func (r *reconcileState) numCoresRevision(revision string) int {
	coreSet := r.coreSet()
	if coreSet == nil {
		return 0
	}
	num := 0
	for _, pod := range r.podsManagedBy(coreSet) {
		podRevision := pod.Labels[appsv1.ControllerRevisionHashLabelKey]
		if podRevision == revision {
			num++
		}
	}
	return num
}

// Returns ReplicaSet representing current set of replicant nodes.
// Current set is considered outdated if CurrentRevision != UpdateRevision.
func (r *reconcileState) currentReplicantSet(instance *crd.EMQX) *appsv1.ReplicaSet {
	for _, rs := range r.replicantSets {
		hash := rs.Labels[crd.LabelPodTemplateHash]
		if hash == instance.Status.ReplicantNodesStatus.CurrentRevision {
			return rs
		}
	}
	return nil
}

// Returns ReplicaSet representing newest set of replicant nodes.
// Same as current if CurrentRevision == UpdateRevision.
func (r *reconcileState) updateReplicantSet(instance *crd.EMQX) *appsv1.ReplicaSet {
	for _, rs := range r.replicantSets {
		hash := rs.Labels[crd.LabelPodTemplateHash]
		if hash == instance.Status.ReplicantNodesStatus.UpdateRevision {
			return rs
		}
	}
	return nil
}

func (r *reconcileState) hasReplicants() bool {
	for _, rs := range r.replicantSets {
		if rs.Status.Replicas > 0 || ptr.Deref(rs.Spec.Replicas, 1) > 0 {
			return true
		}
	}
	for _, pod := range r.pods {
		if pod.Labels[crd.LabelMriaRole] == crd.RoleReplicant {
			return true
		}
	}
	return false
}

// outdatedReplicantReplicaSets returns all replicant ReplicaSets except the update revision set,
// sorted by creation timestamp (oldest first).
func (r *reconcileState) outdatedReplicantSets(instance *crd.EMQX) []*appsv1.ReplicaSet {
	updateRs := r.updateReplicantSet(instance)
	if updateRs == nil {
		return nil
	}
	out := []*appsv1.ReplicaSet{}
	for _, rs := range r.replicantSets {
		if rs.UID == updateRs.UID {
			continue
		}
		out = append(out, rs)
	}
	sortByCreationTimestamp(out)
	return out
}

// outdatedReplicantPodsSorted lists pods owned by any outdated replicant ReplicaSet (all except the
// update revision), de-duplicated by name and sorted by name for deterministic drain order.
func (r *reconcileState) outdatedReplicantPods(instance *crd.EMQX) []*corev1.Pod {
	out := []*corev1.Pod{}
	outdatedRs := r.outdatedReplicantSets(instance)
	if outdatedRs == nil {
		return nil
	}
	for _, rs := range outdatedRs {
		outdatedPods := r.podsManagedBy(rs)
		sortByName(outdatedPods)
		out = append(out, outdatedPods...)
	}
	return out
}

func (r *reconcileState) numReplicants() int32 {
	out := int32(0)
	for _, rs := range r.replicantSets {
		out += rs.Status.Replicas
	}
	return out
}

func (r *reconcileState) numReadyReplicants() int32 {
	out := int32(0)
	for _, rs := range r.replicantSets {
		out += rs.Status.ReadyReplicas
	}
	return out
}

func (r *reconcileState) numAvailableReplicants() int32 {
	out := int32(0)
	for _, rs := range r.replicantSets {
		out += rs.Status.AvailableReplicas
	}
	return out
}

type loadState struct {
	*EMQXReconciler
}

func (l *loadState) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	state, err := loadReconcileState(r.ctx, l.Client, instance)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to load reconcile round state")}
	}
	r.state = state
	return subResult{}
}

func reloadReconcileState(r *reconcileRound, client k8s.Client, instance *crd.EMQX) error {
	state, err := loadReconcileState(r.ctx, client, instance)
	if err != nil {
		return err
	}
	r.state = state
	return nil
}

func loadReconcileState(
	ctx context.Context,
	client k8s.Client,
	instance *crd.EMQX,
) (*reconcileState, error) {
	var err error
	state := &reconcileState{}

	stsList := &appsv1.StatefulSetList{}
	err = client.List(ctx, stsList,
		k8s.InNamespace(instance.Namespace),
		k8s.MatchingLabels(instance.DefaultLabelsWith(crd.CoreLabels())),
	)
	if err != nil {
		return nil, err
	}

	for _, sts := range stsList.Items {
		state.coreSets = append(state.coreSets, sts.DeepCopy())
	}

	sortByCreationTimestamp(state.coreSets)

	rsList := &appsv1.ReplicaSetList{}
	err = client.List(ctx, rsList,
		k8s.InNamespace(instance.Namespace),
		k8s.MatchingLabels(instance.DefaultLabelsWith(crd.ReplicantLabels())),
	)
	if err != nil {
		return nil, err
	}

	for _, rs := range rsList.Items {
		state.replicantSets = append(state.replicantSets, rs.DeepCopy())
	}

	sortByCreationTimestamp(state.replicantSets)

	podList := &corev1.PodList{}
	err = client.List(ctx, podList,
		k8s.InNamespace(instance.Namespace),
		k8s.MatchingLabels(instance.DefaultLabels()),
	)
	if err != nil {
		return nil, err
	}

	for _, pod := range podList.Items {
		pod := pod.DeepCopy()
		state.pods = append(state.pods, pod)
	}

	return state, nil
}
