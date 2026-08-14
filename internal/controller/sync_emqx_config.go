package controller

import (
	"crypto/sha256"
	"fmt"

	emperror "emperror.dev/errors"
	crd "github.com/emqx/emqx-operator/api/v3beta1"
	config "github.com/emqx/emqx-operator/internal/controller/config"
	resources "github.com/emqx/emqx-operator/internal/controller/resources"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	"github.com/emqx/emqx-operator/internal/emqx/api"
	"github.com/emqx/emqx-operator/internal/errors"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

type syncConfig struct {
	*EMQXReconciler
}

func (s *syncConfig) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	userConfig := config.RenderRoots(instance.Spec.Config.Roots)
	baseConfig := config.WithDefaults(userConfig)
	runtimeRoots, restartRequired := config.SplitRuntimeRoots(instance.Spec.Config.Roots)
	runtimeConfig := config.RenderRoots(runtimeRoots)

	runtimeHash := fmt.Sprintf("%x", sha256.Sum256([]byte(runtimeConfig)))
	runtimeHashApplied := instance.Annotations[crd.AnnotationLastRuntimeConfigHash]
	metadataChanged := false

	// No coreSet yet:
	// Create or update ConfigMap resource.
	if r.state.coreSet() == nil {
		_, err := s.syncConfigMap(r, instance, baseConfig)
		if err != nil {
			return reconcileError(err)
		}
		instance.Status.SetCondition(
			crd.ConfigApplied,
			metav1.ConditionFalse,
			"PendingInitialStartup",
			"Configuration is staged for the initial EMQX cluster startup",
			instance.Generation,
		)
		err = s.persistConfigState(r, instance, false)
		if err != nil {
			return reconcileError(err)
		}
		return subResult{}
	}

	req := r.preferredCoreRequester()

	// Runtime-applicable configuration changed / EMQX API unavailable:
	// Update ConfigMap, EMQX API might be unavailable because of the broken config.
	if runtimeHashApplied != runtimeHash && req == nil {
		_, err := s.syncConfigMap(r, instance, baseConfig)
		if err != nil {
			return reconcileError(err)
		}
		instance.Status.SetCondition(
			crd.ConfigApplied,
			metav1.ConditionFalse,
			"PendingAvailability",
			"Configuration is staged for the EMQX cluster availability",
			instance.Generation,
		)
		err = s.persistConfigState(r, instance, false)
		if err != nil {
			return reconcileError(err)
		}
		return subResult{}
	}

	// Runtime-applicable configuration changed:
	// Update runtime-applicable configuration through EMQX API if there were any changes.
	// If there were not (no previous runtime update was performed), just record the hash.
	if runtimeHashApplied != runtimeHash {
		var err error
		if runtimeHashApplied != "" {
			err = api.UpdateConfigs(req, runtimeConfig)
		}
		if err != nil && errors.IsTransientAPIError(err) {
			return reconcileError(emperror.Wrap(err, "failed to update EMQX runtime config"))
		}
		if err != nil {
			instance.Status.SetCondition(
				crd.ConfigApplied,
				metav1.ConditionFalse,
				"ConfigRejected",
				"Runtime configuration rejected by EMQX API",
				instance.Generation,
			)
			// Ignoring persistence errors, will reconcile anyway soon:
			_ = s.persistConfigState(r, instance, false)
			return reconcileError(emperror.Wrap(err, "failed to update EMQX runtime config"))
		}
		metadataChanged = s.reflectConfigHash(instance, crd.AnnotationLastRuntimeConfigHash, runtimeHash)
	}

	// Only accepted update is persisted to ConfigMap for future Pod starts.
	configMapChanged, err := s.syncConfigMap(r, instance, baseConfig)
	if err != nil {
		return reconcileError(err)
	}

	if configMapChanged || metadataChanged {
		if len(restartRequired) == 0 {
			instance.Status.SetCondition(
				crd.ConfigApplied,
				metav1.ConditionTrue,
				"Applied",
				"Desired configuration is active",
				instance.Generation,
			)
		} else {
			instance.Status.SetCondition(
				crd.ConfigApplied,
				metav1.ConditionFalse,
				"RestartRequired",
				fmt.Sprintf("Configuration paths require a Pod restart: %v", restartRequired),
				instance.Generation,
			)
			s.EventRecorder.Event(
				instance,
				corev1.EventTypeWarning,
				"ConfigRestartRequired",
				fmt.Sprintf("Configuration paths require a Pod restart: %v", restartRequired),
			)
		}
		err = s.persistConfigState(r, instance, metadataChanged)
		if err != nil {
			return reconcileError(err)
		}
	}

	return subResult{}
}

func (s *syncConfig) syncConfigMap(r *reconcileRound, instance *crd.EMQX, baseConfig string) (bool, error) {
	resource := resources.EMQXConfig(instance)
	configMap := &corev1.ConfigMap{}
	err := s.Client.Get(r.ctx, instance.ConfigsNamespacedName(), configMap)
	if err != nil && k8sErrors.IsNotFound(err) {
		configMap = resource.ConfigMap(baseConfig)
		r.log.V(1).Info("creating config resource", "configMap", klog.KObj(configMap))
		if err := ctrl.SetControllerReference(instance, configMap, s.Scheme); err != nil {
			return false, emperror.Wrap(err, "failed to set controller reference for configMap")
		}
		if err := s.Client.Create(r.ctx, configMap); err != nil {
			return false, emperror.Wrap(err, "failed to create configMap")
		}
		return true, nil
	}
	if err != nil {
		return false, emperror.Wrap(err, "failed to get configMap")
	}
	if configMap.Data[resources.BaseConfigFile] != baseConfig {
		r.log.V(1).Info("updating config resource", "configMap", klog.KObj(configMap))
		configMap.Data[resources.BaseConfigFile] = baseConfig
		if err := s.Client.Update(r.ctx, configMap); err != nil {
			return false, emperror.Wrap(err, "failed to update configMap")
		}
		return true, nil
	}
	return false, nil
}

func (s *syncConfig) reflectConfigHash(instance *crd.EMQX, annotation, hash string) bool {
	return util.AttachAnnotation(instance, annotation, hash)
}

func (s *syncConfig) persistConfigState(r *reconcileRound, instance *crd.EMQX, metadataChanged bool) error {
	if err := s.Client.Status().Update(r.ctx, instance); err != nil {
		return emperror.Wrap(err, "failed to update EMQX configuration status")
	}
	if metadataChanged {
		if err := s.Client.Update(r.ctx, instance); err != nil {
			return emperror.Wrap(err, "failed to record EMQX annotations")
		}
	}
	return nil
}
