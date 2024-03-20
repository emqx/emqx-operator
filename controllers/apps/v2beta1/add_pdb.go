package v2beta1

import (
	"context"

	emperror "emperror.dev/errors"
	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	"github.com/go-logr/logr"
	policyv1 "k8s.io/api/policy/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

type addPdb struct {
	*EMQXReconciler
}

func (a *addPdb) reconcile(ctx context.Context, logger logr.Logger, instance *appsv2beta1.EMQX, _ innerReq.RequesterInterface) subResult {
	pdbList := generatePodDisruptionBudget(instance)
	for _, pdb := range pdbList {
		if err := ctrl.SetControllerReference(instance, pdb, a.Scheme); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to set controller reference")}
		}
		if err := a.Client.Create(ctx, pdb); err != nil {
			if !k8sErrors.IsAlreadyExists(err) {
				return subResult{err: emperror.Wrap(err, "failed to create PDB")}
			}
		}
	}
	return subResult{}
}

func generatePodDisruptionBudget(instance *appsv2beta1.EMQX) []*policyv1.PodDisruptionBudget {
	pdbList := []*policyv1.PodDisruptionBudget{}
	corePdb := &policyv1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy/v1",
			Kind:       "PodDisruptionBudget",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: instance.Namespace,
			Name:      instance.CoreNamespacedName().Name,
			Labels:    appsv2beta1.CloneAndMergeMap(appsv2beta1.DefaultLabels(instance), instance.Labels),
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: appsv2beta1.CloneAndMergeMap(
					appsv2beta1.DefaultCoreLabels(instance),
					instance.Spec.CoreTemplate.Labels,
				),
			},
			MinAvailable: &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 1,
			},
		},
	}
	pdbList = append(pdbList, corePdb)
	if appsv2beta1.IsExistReplicant(instance) {
		replPdb := corePdb.DeepCopy()
		replPdb.Name = instance.ReplicantNamespacedName().Name
		replPdb.Spec.Selector.MatchLabels = appsv2beta1.CloneAndMergeMap(
			appsv2beta1.DefaultReplicantLabels(instance),
			instance.Spec.ReplicantTemplate.Labels,
		)
		pdbList = append(pdbList, replPdb)
	}
	return pdbList
}
