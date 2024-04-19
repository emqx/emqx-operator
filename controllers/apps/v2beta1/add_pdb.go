package v2beta1

import (
	"context"

	emperror "emperror.dev/errors"
	semver "github.com/Masterminds/semver/v3"
	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	"github.com/go-logr/logr"
	policyv1 "k8s.io/api/policy/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/discovery"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kubeConfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

type addPdb struct {
	*EMQXReconciler
}

func (a *addPdb) reconcile(ctx context.Context, logger logr.Logger, instance *appsv2beta1.EMQX, _ innerReq.RequesterInterface) subResult {
	discoveryClient, _ := discovery.NewDiscoveryClientForConfig(kubeConfig.GetConfigOrDie())
	kubeVersion, _ := discoveryClient.ServerVersion()
	v, _ := semver.NewVersion(kubeVersion.String())

	pdbList := []client.Object{}
	if v.LessThan(semver.MustParse("1.21")) {
		corePdb, replPdb := generatePodDisruptionBudgetV1beta1(instance)
		pdbList = append(pdbList, corePdb)
		if replPdb != nil {
			pdbList = append(pdbList, replPdb)
		}
	} else {
		corePdb, replPdb := generatePodDisruptionBudget(instance)
		pdbList = append(pdbList, corePdb)
		if replPdb != nil {
			pdbList = append(pdbList, replPdb)
		}
	}

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

func generatePodDisruptionBudget(instance *appsv2beta1.EMQX) (*policyv1.PodDisruptionBudget, *policyv1.PodDisruptionBudget) {
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
	if appsv2beta1.IsExistReplicant(instance) {
		replPdb := corePdb.DeepCopy()
		replPdb.Name = instance.ReplicantNamespacedName().Name
		replPdb.Spec.Selector.MatchLabels = appsv2beta1.CloneAndMergeMap(
			appsv2beta1.DefaultReplicantLabels(instance),
			instance.Spec.ReplicantTemplate.Labels,
		)
		return corePdb, replPdb
	}
	return corePdb, nil
}

func generatePodDisruptionBudgetV1beta1(instance *appsv2beta1.EMQX) (*policyv1beta1.PodDisruptionBudget, *policyv1beta1.PodDisruptionBudget) {
	corePdb := &policyv1beta1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy/v1",
			Kind:       "PodDisruptionBudget",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: instance.Namespace,
			Name:      instance.CoreNamespacedName().Name,
			Labels:    appsv2beta1.CloneAndMergeMap(appsv2beta1.DefaultLabels(instance), instance.Labels),
		},
		Spec: policyv1beta1.PodDisruptionBudgetSpec{
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
	if appsv2beta1.IsExistReplicant(instance) {
		replPdb := corePdb.DeepCopy()
		replPdb.Name = instance.ReplicantNamespacedName().Name
		replPdb.Spec.Selector.MatchLabels = appsv2beta1.CloneAndMergeMap(
			appsv2beta1.DefaultReplicantLabels(instance),
			instance.Spec.ReplicantTemplate.Labels,
		)
		return corePdb, replPdb
	}
	return corePdb, nil
}
