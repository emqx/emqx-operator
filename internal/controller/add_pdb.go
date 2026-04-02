package controller

import (
	emperror "emperror.dev/errors"
	crd "github.com/emqx/emqx-operator/api/v3alpha1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addPdb struct {
	*EMQXReconciler
}

func (a *addPdb) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	pdbList := []client.Object{}
	corePdb, replPdb := generatePodDisruptionBudget(instance)
	pdbList = append(pdbList, corePdb)
	if replPdb != nil {
		pdbList = append(pdbList, replPdb)
	}

	err := a.CreateOrUpdateList(r.ctx, a.Scheme, r.log, instance, pdbList)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to create or update PDBs")}
	}
	return subResult{}
}

func generatePodDisruptionBudget(instance *crd.EMQX) (*policyv1.PodDisruptionBudget, *policyv1.PodDisruptionBudget) {
	corePdb := &policyv1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy/v1",
			Kind:       "PodDisruptionBudget",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: instance.Namespace,
			Name:      instance.CoreNamespacedName().Name,
			Labels:    instance.DefaultLabelsWith(instance.Labels),
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: instance.DefaultLabelsWith(crd.CoreLabels()),
			},
			MinAvailable:   instance.Spec.CoreTemplate.Spec.MinAvailable,
			MaxUnavailable: instance.Spec.CoreTemplate.Spec.MaxUnavailable,
		},
	}

	if instance.Spec.HasReplicants() {
		replPdb := corePdb.DeepCopy()
		replPdb.Name = instance.ReplicantNamespacedName().Name
		replPdb.Spec.Selector.MatchLabels = instance.DefaultLabelsWith(crd.ReplicantLabels())
		replPdb.Spec.MinAvailable = instance.Spec.ReplicantTemplate.Spec.MinAvailable
		replPdb.Spec.MaxUnavailable = instance.Spec.ReplicantTemplate.Spec.MaxUnavailable
		return corePdb, replPdb
	}
	return corePdb, nil
}
