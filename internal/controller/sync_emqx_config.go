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
	runtimeRoots, startupRoots := config.SplitRoots(instance.Spec.Config.Roots)
	runtimeConfig := config.RenderRoots(runtimeRoots)
	baseConfig := config.RenderBaseConfig(runtimeRoots)
	startupConfig := config.RenderRoots(startupRoots)

	runtimeRevisionTarget := configStringHash(runtimeConfig)
	startupRevisionTarget := startupConfigRevision(startupRoots)

	statusChanged := false
	runtimeRevision := instance.Status.Config.RuntimeRevision

	// No coreSet yet:
	// Create or update ConfigMap resource.
	if r.state.coreSet() == nil {
		err := s.syncConfigMap(r, instance, baseConfig, startupConfig)
		if err != nil {
			return reconcileError(err)
		}
		// Record the runtime-hash to avoid sending the same configuration through the API after
		// initial startup.
		s.recordRuntimeConfigRevision(instance, runtimeRevisionTarget)
		instance.Status.SetCondition(
			crd.ConfigApplied,
			metav1.ConditionFalse,
			"PendingInitialStartup",
			"Configuration is staged for the initial EMQX cluster startup",
			instance.Generation,
		)
		err = s.persistConfigState(r, instance)
		if err != nil {
			return reconcileError(err)
		}
		return subResult{}
	}

	req := r.preferredCoreRequester()

	// Runtime-applicable configuration changed / EMQX API unavailable:
	// Update ConfigMap, EMQX API might be unavailable because of the broken config.
	if runtimeRevision != runtimeRevisionTarget && req == nil {
		err := s.syncConfigMap(r, instance, baseConfig, startupConfig)
		if err != nil {
			return reconcileError(err)
		}
		instance.Status.SetCondition(
			crd.ConfigApplied,
			metav1.ConditionFalse,
			"PendingAvailability",
			"Configuration is staged until EMQX cluster becomes available",
			instance.Generation,
		)
		err = s.persistConfigState(r, instance)
		if err != nil {
			return reconcileError(err)
		}
		return subResult{}
	}

	// Runtime-applicable configuration changed:
	// Update runtime-applicable configuration through EMQX API if there were any changes.
	if runtimeRevision != runtimeRevisionTarget {
		var err error
		if len(runtimeRoots) > 0 {
			r.log.V(1).Info("updating runtime config", "config", runtimeConfig)
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
			_ = s.persistConfigState(r, instance)
			return reconcileError(emperror.Wrap(err, "failed to update EMQX runtime config"))
		}
		statusChanged = s.recordRuntimeConfigRevision(instance, runtimeRevisionTarget) || statusChanged
	}

	// Only accepted update is persisted to ConfigMap for future Pod starts.
	err := s.syncConfigMap(r, instance, baseConfig, startupConfig)
	if err != nil {
		return reconcileError(err)
	}

	// Reaching this point means that the runtime revision is current.
	// Ready Pods are the authoritative source for whether the startup configuration is active.
	activeStartupRevisions := r.state.activeStartupConfigRevisions()
	outdatedStartupRevisions := len(activeStartupRevisions)
	if _, targetActive := activeStartupRevisions[startupRevisionTarget]; targetActive {
		outdatedStartupRevisions -= 1
	}

	condition := metav1.Condition{
		Type:               crd.ConfigApplied,
		ObservedGeneration: instance.Generation,
	}
	switch {
	case outdatedStartupRevisions > 0:
		condition.Status = metav1.ConditionFalse
		condition.Reason = "StartupConfigPending"
		paths := config.StartupConfigPaths(startupRoots)
		if len(paths) > 0 {
			condition.Message = fmt.Sprintf("Configuration roots require rolling restart: %v", paths)
		} else {
			condition.Message = "Removal of startup configuration roots requires rolling restart"
		}
	case len(activeStartupRevisions) == 0:
		condition.Status = metav1.ConditionFalse
		condition.Reason = "PendingAvailability"
		condition.Message = "Desired startup configuration needs ready EMQX pod for verification"
	default:
		condition.Status = metav1.ConditionTrue
		condition.Reason = "Applied"
		condition.Message = "Desired configuration is active"
	}

	conditionChanged := instance.Status.AttachCondition(condition)

	if statusChanged || conditionChanged {
		if conditionChanged && outdatedStartupRevisions > 0 {
			s.EventRecorder.Event(
				instance,
				corev1.EventTypeNormal,
				"StartupConfigRollout",
				condition.Message,
			)
		}
		err = s.persistConfigState(r, instance)
		if err != nil {
			return reconcileError(err)
		}
	}

	return subResult{}
}

func attachTemplateStartupConfigRevision(instance *crd.EMQX, template *corev1.PodTemplateSpec) {
	_, startupRoots := config.SplitRoots(instance.Spec.Config.Roots)
	if len(startupRoots) > 0 {
		util.AttachAnnotation(
			template,
			crd.AnnotationStartupConfigRevision,
			startupConfigRevision(startupRoots),
		)
	}
}

func startupConfigRevision(roots crd.ConfigRoots) string {
	return configRevision(roots)
}

func configRevision(roots crd.ConfigRoots) string {
	return configStringHash(config.RenderRoots(roots))
}

func configStringHash(config string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(config)))
}

func (s *syncConfig) syncConfigMap(
	r *reconcileRound,
	instance *crd.EMQX,
	baseConfig, startupConfig string,
) error {
	resource := resources.EMQXConfig(instance)
	configMap := &corev1.ConfigMap{}
	err := s.Client.Get(r.ctx, instance.ConfigsNamespacedName(), configMap)
	if err != nil && k8sErrors.IsNotFound(err) {
		configMap = resource.ConfigMap(baseConfig, startupConfig)
		r.log.V(1).Info("creating config resource", "configMap", klog.KObj(configMap))
		if err := ctrl.SetControllerReference(instance, configMap, s.Scheme); err != nil {
			return emperror.Wrap(err, "failed to set controller reference for configMap")
		}
		if err := s.Client.Create(r.ctx, configMap); err != nil {
			return emperror.Wrap(err, "failed to create configMap")
		}
		return nil
	}
	if err != nil {
		return emperror.Wrap(err, "failed to get configMap")
	}
	if configMap.Data[resources.BaseConfigFile] != baseConfig ||
		configMap.Data[resources.StartupConfigFile] != startupConfig {
		r.log.V(1).Info("updating config resource", "configMap", klog.KObj(configMap))
		if configMap.Data == nil {
			configMap.Data = map[string]string{}
		}
		configMap.Data[resources.BaseConfigFile] = baseConfig
		configMap.Data[resources.StartupConfigFile] = startupConfig
		if err := s.Client.Update(r.ctx, configMap); err != nil {
			return emperror.Wrap(err, "failed to update configMap")
		}
	}
	return nil
}

func (*syncConfig) recordRuntimeConfigRevision(instance *crd.EMQX, revision string) bool {
	if instance.Status.Config.RuntimeRevision == revision {
		return false
	}
	instance.Status.Config.RuntimeRevision = revision
	return true
}

func (s *syncConfig) persistConfigState(r *reconcileRound, instance *crd.EMQX) error {
	if err := s.Client.Status().Update(r.ctx, instance); err != nil {
		return emperror.Wrap(err, "failed to update EMQX configuration status")
	}
	return nil
}
