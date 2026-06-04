package controller

import (
	"fmt"

	emperror "emperror.dev/errors"
	crdv2 "github.com/emqx/emqx-operator/api/v2"
	config "github.com/emqx/emqx-operator/internal/controller/config"
	resources "github.com/emqx/emqx-operator/internal/controller/resources"
	"github.com/emqx/emqx-operator/internal/emqx/api"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

type syncConfig struct {
	*EMQXReconciler
}

func (s *syncConfig) reconcile(r *reconcileRound, instance *crdv2.EMQX) subResult {
	// Fetch desired / applied configuration.
	confSpec := instance.Spec.Config.Data
	confLast := lastAppliedConfig(instance)

	// Do not try to update not-really-changeable configuration options.
	conf := confSpec
	stripped := []string{}
	if confLast != nil {
		conf, stripped = stripNonChangeableConfig(confSpec, config.WithDefaults(*confLast))
	}

	// Make sure the config map exists
	resource := resources.EMQXConfig(instance)
	confWithDefaults := config.WithDefaults(conf)
	configMap := &corev1.ConfigMap{}
	err := s.Client.Get(r.ctx, instance.ConfigsNamespacedName(), configMap)
	if err != nil && k8sErrors.IsNotFound(err) {
		configMap = resource.ConfigMap(confWithDefaults)
		if err := ctrl.SetControllerReference(instance, configMap, s.Scheme); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to set controller reference for configMap")}
		}
		r.log.V(1).Info("creating config resource", "configMap", klog.KObj(configMap))
		if err := s.Client.Create(r.ctx, configMap); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to create configMap")}
		}
		return subResult{}
	}
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to get configMap")}
	}

	// If the config is different, update the config right away.
	// Assuming the config is valid, otherwise master controller would bail out.
	if configMap.Data[resources.BaseConfigFile] != confWithDefaults {
		configMap = resource.ConfigMap(confWithDefaults)
		if err := ctrl.SetControllerReference(instance, configMap, s.Scheme); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to set controller reference for configMap")}
		}
		r.log.V(1).Info("updating config resource", "configMap", klog.KObj(configMap))
		if err := s.Client.Update(r.ctx, configMap); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to update configMap")}
		}
		if len(stripped) > 0 {
			s.EventRecorder.Event(
				instance,
				corev1.EventTypeNormal, "PartialConfigUpdate",
				fmt.Sprintf("Skipped unchangeable entries: %v", stripped),
			)
		}
	}

	// If the annotation is not set, set it to the current config and return.
	// This reconciler is apparently running for the first time.
	if confLast == nil {
		reflectLastAppliedConfig(instance, conf)
		if err := s.Client.Update(r.ctx, instance); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to update emqx instance annotation")}
		}
		return subResult{}
	}

	// Postpone runtime config updates until ready.
	if !instance.Status.IsConditionTrue(crdv2.CoreNodesReady) {
		return subResult{}
	}

	// If the annotation is set, and the config is different, update the config.
	if *confLast != conf {
		// Delete readonly configs
		c, err := config.EMQXConfig(confWithDefaults)
		if err != nil || c == nil {
			return subResult{err: emperror.Wrap(err, "failed to parse .spec.config.data")}
		}
		strippedReadonly := c.StripReadOnlyConfig()
		confRuntime := c.JSON()

		// Update the config through API
		r.log.V(1).Info("applying runtime config", "config", confRuntime)
		if err := api.UpdateConfigs(r.oldestCoreRequester(), instance.Spec.Config.Mode, confRuntime); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to update emqx config through API")}
		}
		if len(strippedReadonly) > 0 {
			s.EventRecorder.Event(
				instance,
				corev1.EventTypeNormal, "PartialRuntimeConfigUpdate",
				fmt.Sprintf("Skipped readonly entries: %v", strippedReadonly),
			)
		}

		reflectLastAppliedConfig(instance, conf)
		if err := s.Client.Update(r.ctx, instance); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to update emqx instance annotation")}
		}

		// Restart reconciliation loop with consistent reconcile state.
		return subResult{result: ctrl.Result{Requeue: true}}
	}

	return subResult{}
}

func stripNonChangeableConfig(confDesired string, confLast string) (string, []string) {
	// Operator relies on Dashboard listener to access EMQX API.
	// Changing the dashboard listener port is too finicky to allow it, so we strip
	// any changes to previously configured dashboard listener port, either `http` or
	// `https`, whichever is currently configured.
	var dashboardConfigPaths = []string{
		"dashboard.listeners.http.bind",
		"dashboard.listeners.https.bind",
	}
	var stripped = []string{}
	cd, _ := config.EMQXConfig(confDesired)
	cl, _ := config.EMQXConfig(confLast)
	if cd != nil && cl != nil {
		for _, path := range dashboardConfigPaths {
			vd, vdOk := cd.GetString(path)
			vl, vlOk := cl.GetString(path)
			// Disallow changing if following conditions are met:
			// * Desired is configured and was configured previously (but not "0", i.e. disabled)
			// * Desired is different from previous value
			// Otherwise, listener is being enabled, which should be allowed.
			if vdOk && vlOk && vd != "" && vl != "" && vl != "0" && vd != vl {
				_ = cd.Strip(path)
				stripped = append(stripped, path)
				return cd.JSON(), stripped
			}
		}
	}
	return confDesired, stripped
}

func reflectLastAppliedConfig(instance *crdv2.EMQX, confStr string) {
	if instance.Annotations == nil {
		instance.Annotations = map[string]string{}
	}
	instance.Annotations[crdv2.AnnotationLastEMQXConfig] = confStr
}

func lastAppliedConfig(instance *crdv2.EMQX) *string {
	if instance.Annotations != nil {
		if confStr := instance.Annotations[crdv2.AnnotationLastEMQXConfig]; confStr != "" {
			return &confStr
		}
	}
	return nil
}

func applicableConfig(instance *crdv2.EMQX) string {
	// If the annotation is set, use it: most of the time it's the config currently in use.
	if confStr := lastAppliedConfig(instance); confStr != nil {
		return config.WithDefaults(*confStr)
	}
	// If not, running for the first time, use spec's config.
	return config.WithDefaults(instance.Spec.Config.Data)
}
