package controller

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	emperror "emperror.dev/errors"
	appsv2beta1 "github.com/emqx/emqx-operator/api/v2beta1"
	config "github.com/emqx/emqx-operator/internal/controller/config"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type syncConfig struct {
	*EMQXReconciler
}

func (s *syncConfig) reconcile(ctx context.Context, logger logr.Logger, instance *appsv2beta1.EMQX, r innerReq.RequesterInterface) subResult {
	// Merge default config.
	// Assuming the config is valid, otherwise master controller would bail out.
	confStr := config.MergeDefaults(instance.Spec.Config.Data)

	// Make sure the config map exists
	configMap := &corev1.ConfigMap{}
	err := s.Client.Get(ctx, instance.ConfigsNamespacedName(), configMap)
	if err != nil && k8sErrors.IsNotFound(err) {
		configMap = generateConfigMap(instance, confStr)
		if err := ctrl.SetControllerReference(instance, configMap, s.Scheme); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to set controller reference for configMap")}
		}
		if err := s.Client.Create(ctx, configMap); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to create configMap")}
		}
		return subResult{}
	}
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to get configMap")}
	}

	// If the config is different, update the config right away.
	if configMap.Data["emqx.conf"] != confStr {
		if err := s.Client.Update(ctx, generateConfigMap(instance, confStr)); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to update configMap")}
		}
	}

	lastConfStr, ok := instance.Annotations[appsv2beta1.AnnotationsLastEMQXConfigKey]

	// If the annotation is not set, set it to the current config and return.
	if !ok {
		if instance.Annotations == nil {
			instance.Annotations = map[string]string{}
		}
		instance.Annotations[appsv2beta1.AnnotationsLastEMQXConfigKey] = instance.Spec.Config.Data
		if err := s.Client.Update(ctx, instance); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to update emqx instance annotation")}
		}
		return subResult{}
	}

	// If the annotation is set, and the config is different, update the config.
	if lastConfStr != instance.Spec.Config.Data {
		conf := s.conf.Copy()

		if !instance.Status.IsConditionTrue(appsv2beta1.CoreNodesReady) {
			return subResult{}
		}

		// Delete readonly configs
		stripped := conf.StripReadOnlyConfig()
		if len(stripped) > 0 {
			s.EventRecorder.Event(
				instance,
				corev1.EventTypeNormal, "WontUpdateReadOnlyConfig",
				fmt.Sprintf("Stripped readonly config entries, will not be updated: %v", stripped),
			)
		}

		if err := putEMQXConfigsByAPI(r, instance.Spec.Config.Mode, conf.Print()); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to update emqx config through API")}
		}

		instance.Annotations[appsv2beta1.AnnotationsLastEMQXConfigKey] = instance.Spec.Config.Data
		if err := s.Client.Update(ctx, instance); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to update emqx instance annotation")}
		}
	}

	return subResult{}
}

func generateConfigMap(instance *appsv2beta1.EMQX, data string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.ConfigsNamespacedName().Name,
			Namespace: instance.Namespace,
			Labels:    appsv2beta1.CloneAndMergeMap(appsv2beta1.DefaultLabels(instance), instance.Labels),
		},
		Data: map[string]string{
			"emqx.conf": data,
		},
	}
}

func putEMQXConfigsByAPI(r innerReq.RequesterInterface, mode, config string) error {
	url := r.GetURL("api/v5/configs", "mode="+strings.ToLower(mode), "ignore_readonly=true")

	resp, body, err := r.Request("PUT", url, []byte(config), http.Header{
		"Content-Type": []string{"text/plain"},
	})
	if err != nil {
		return emperror.Wrapf(err, "failed to put API %s", url.String())
	}
	if resp.StatusCode != 200 {
		return emperror.Errorf("failed to put API %s, status : %s, body: %s", url.String(), resp.Status, body)
	}
	return nil
}
