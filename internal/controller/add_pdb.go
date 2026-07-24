package controller

import (
	emperror "emperror.dev/errors"
	crdv2 "github.com/emqx/emqx-operator/api/v2"
	policyv1 "k8s.io/api/policy/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

type addPdb struct {
	*EMQXReconciler
}

func (a *addPdb) reconcile(r *reconcileRound, instance *crdv2.EMQX) subResult {
	if err := a.reconcilePodDisruptionBudget(
		r,
		instance,
		instance.Spec.CoreTemplate.Spec.PodDisruptionBudgetEnabled(),
		generateCorePodDisruptionBudget(instance),
	); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to reconcile core PDB")}
	}

	if err := a.reconcilePodDisruptionBudget(
		r,
		instance,
		instance.Spec.HasReplicants() && instance.Spec.ReplicantTemplate.Spec.PodDisruptionBudgetEnabled(),
		generateReplicantPodDisruptionBudget(instance),
	); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to reconcile replicant PDB")}
	}

	return subResult{}
}

func (a *addPdb) reconcilePodDisruptionBudget(
	r *reconcileRound,
	instance *crdv2.EMQX,
	enabled bool,
	pdb *policyv1.PodDisruptionBudget,
) error {
	if enabled {
		return a.CreateOrUpdate(r.ctx, a.Scheme, r.log, instance, pdb)
	}
	return a.removeStalePodDisruptionBudget(r, pdb)
}

func (a *addPdb) removeStalePodDisruptionBudget(r *reconcileRound, pdb *policyv1.PodDisruptionBudget) error {
	err := a.Client.Delete(r.ctx, pdb)
	if err != nil && !k8sErrors.IsNotFound(err) {
		return err
	}
	return nil
}

func generateCorePodDisruptionBudget(instance *crdv2.EMQX) *policyv1.PodDisruptionBudget {
	template := instance.Spec.CoreTemplate
	return newPodDisruptionBudget(
		instance,
		instance.CoreNamespacedName().Name,
		instance.DefaultLabelsWith(crdv2.CoreLabels(), template.Labels),
		template.Spec.MinAvailable,
		template.Spec.MaxUnavailable,
	)
}

func generateReplicantPodDisruptionBudget(instance *crdv2.EMQX) *policyv1.PodDisruptionBudget {
	template := ptr.Deref(instance.Spec.ReplicantTemplate, crdv2.EMQXReplicantTemplate{})
	return newPodDisruptionBudget(
		instance,
		instance.ReplicantNamespacedName().Name,
		instance.DefaultLabelsWith(crdv2.ReplicantLabels(), template.Labels),
		template.Spec.MinAvailable,
		template.Spec.MaxUnavailable,
	)
}

func newPodDisruptionBudget(
	instance *crdv2.EMQX,
	name string,
	selector map[string]string,
	minAvailable,
	maxUnavailable *intstr.IntOrString,
) *policyv1.PodDisruptionBudget {
	return &policyv1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy/v1",
			Kind:       "PodDisruptionBudget",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: instance.Namespace,
			Name:      name,
			Labels:    instance.DefaultLabelsWith(instance.Labels),
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: selector,
			},
			MinAvailable:   minAvailable,
			MaxUnavailable: maxUnavailable,
		},
	}
}
