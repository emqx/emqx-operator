package v2beta1

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	emperror "emperror.dev/errors"
	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	"github.com/go-logr/logr"
	"github.com/rory-z/go-hocon"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type syncConfig struct {
	*EMQXReconciler
}

func (s *syncConfig) reconcile(ctx context.Context, logger logr.Logger, instance *appsv2beta1.EMQX, r innerReq.RequesterInterface) subResult {
	hoconConfig := mergeDefaultConfig(instance.Spec.Config.Data)
	confStr := hoconConfig.String()

	lastConfigStr, ok := instance.Annotations[appsv2beta1.AnnotationsLastEMQXConfigKey]
	if !ok {
		if err := s.update(ctx, logger, instance, confStr); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to update emqx config")}
		}
		return subResult{}
	}

	lastHoconConfig, _ := hocon.ParseString(lastConfigStr)
	if !reflect.DeepEqual(hoconConfig, lastHoconConfig) {
		_, coreReady := instance.Status.GetCondition(appsv2beta1.CoreNodesReady)
		if coreReady == nil || !instance.Status.IsConditionTrue(appsv2beta1.CoreNodesReady) {
			return subResult{}
		}

		// Delete readonly configs
		hoconConfigObj := hoconConfig.GetRoot().(hocon.Object)
		delete(hoconConfigObj, "node")
		delete(hoconConfigObj, "cluster")
		delete(hoconConfigObj, "dashboard")

		if err := putEMQXConfigsByAPI(r, instance.Spec.Config.Mode, hoconConfigObj.String()); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to put emqx config")}
		}

		if err := s.update(ctx, logger, instance, confStr); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to update emqx config")}
		}

		return subResult{}
	}

	return subResult{}
}

func (s *syncConfig) update(ctx context.Context, logger logr.Logger, instance *appsv2beta1.EMQX, confStr string) error {
	configMap := generateConfigMap(instance, confStr)
	if err := s.Handler.CreateOrUpdate(ctx, s.Scheme, logger, instance, configMap); err != nil {
		return emperror.Wrap(err, "failed to create or update configMap")
	}

	if instance.Annotations == nil {
		instance.Annotations = map[string]string{}
	}
	instance.Annotations[appsv2beta1.AnnotationsLastEMQXConfigKey] = confStr
	if err := s.Client.Update(ctx, instance); err != nil {
		return emperror.Wrap(err, "failed to update emqx instance annotation")
	}

	return nil
}

func mergeDefaultConfig(config string) *hocon.Config {
	defaultListenerConfig := ""
	defaultListenerConfig += fmt.Sprintln("listeners.tcp.default.bind = 1883")
	defaultListenerConfig += fmt.Sprintln("listeners.ssl.default.bind = 8883")
	defaultListenerConfig += fmt.Sprintln("listeners.ws.default.bind  = 8083")
	defaultListenerConfig += fmt.Sprintln("listeners.wss.default.bind = 8084")

	hoconConfig, _ := hocon.ParseString(defaultListenerConfig + config)
	return hoconConfig
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
	url := r.GetURL("api/v5/configs", "mode="+strings.ToLower(mode))

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
