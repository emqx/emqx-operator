package v2beta1

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	emperror "emperror.dev/errors"
	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	"github.com/go-logr/logr"
	"github.com/rory-z/go-hocon"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

type syncConfig struct {
	*EMQXReconciler
}

func (s *syncConfig) reconcile(ctx context.Context, logger logr.Logger, instance *appsv2beta1.EMQX, r innerReq.RequesterInterface) subResult {
	hoconConfig := mergeDefaultConfig(instance.Spec.Config.Data)
	confStr := hoconConfig.String()

	// Make sure the config map exists
	configMap := &corev1.ConfigMap{}
	if err := s.Client.Get(ctx, types.NamespacedName{
		Name:      instance.ConfigsNamespacedName().Name,
		Namespace: instance.Namespace,
	}, configMap); err != nil {
		if k8sErrors.IsNotFound(err) {
			configMap = generateConfigMap(instance, confStr)
			if err := ctrl.SetControllerReference(instance, configMap, s.Scheme); err != nil {
				return subResult{err: emperror.Wrap(err, "failed to set controller reference for configMap")}
			}
			if err := s.Client.Create(ctx, configMap); err != nil {
				return subResult{err: emperror.Wrap(err, "failed to create configMap")}
			}
			return subResult{}
		}
		return subResult{err: emperror.Wrap(err, "failed to get configMap")}
	}

	lastConfigStr, ok := instance.Annotations[appsv2beta1.AnnotationsLastEMQXConfigKey]
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

	lastHoconConfig, _ := hocon.ParseString(lastConfigStr)
	if !deepEqualHoconValue(hoconConfig.GetRoot(), lastHoconConfig.GetRoot()) {
		_, coreReady := instance.Status.GetCondition(appsv2beta1.CoreNodesReady)
		if coreReady == nil || !instance.Status.IsConditionTrue(appsv2beta1.CoreNodesReady) {
			return subResult{}
		}

		// Delete readonly configs
		hoconConfigObj := hoconConfig.GetRoot().(hocon.Object)
		if _, ok := hoconConfigObj["node"]; ok {
			s.EventRecorder.Event(instance, corev1.EventTypeNormal, "WontUpdateReadOnlyConfig", "Won't update `node` config, because it's readonly config")
			delete(hoconConfigObj, "node")
		}
		if _, ok := hoconConfigObj["cluster"]; ok {
			s.EventRecorder.Event(instance, corev1.EventTypeNormal, "WontUpdateReadOnlyConfig", "Won't update `cluster` config, because it's readonly config")
			delete(hoconConfigObj, "cluster")
		}
		if _, ok := hoconConfigObj["dashboard"]; ok {
			s.EventRecorder.Event(instance, corev1.EventTypeNormal, "WontUpdateReadOnlyConfig", "Won't update `dashboard` config, because it's readonly config")
			delete(hoconConfigObj, "dashboard")
		}
		if _, ok := hoconConfigObj["rpc"]; ok {
			s.EventRecorder.Event(instance, corev1.EventTypeNormal, "WontUpdateReadOnlyConfig", "Won't update `rpc` config, because it's readonly config")
			delete(hoconConfigObj, "rpc")
		}

		if err := putEMQXConfigsByAPI(r, instance.Spec.Config.Mode, hoconConfigObj.String()); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to put emqx config")}
		}

		if err := s.Client.Update(ctx, generateConfigMap(instance, confStr)); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to update configMap")}
		}

		instance.Annotations[appsv2beta1.AnnotationsLastEMQXConfigKey] = instance.Spec.Config.Data
		if err := s.Client.Update(ctx, instance); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to update emqx instance annotation")}
		}

		return subResult{}
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

func mergeDefaultConfig(config string) *hocon.Config {
	defaultListenerConfig := ""
	defaultListenerConfig += fmt.Sprintln("listeners.tcp.default.bind = 1883")
	defaultListenerConfig += fmt.Sprintln("listeners.ssl.default.bind = 8883")
	defaultListenerConfig += fmt.Sprintln("listeners.ws.default.bind  = 8083")
	defaultListenerConfig += fmt.Sprintln("listeners.wss.default.bind = 8084")

	hoconConfig, _ := hocon.ParseString(defaultListenerConfig + config)
	return hoconConfig
}

func deepEqualHoconValue(val1, val2 hocon.Value) bool {
	switch val1.Type() {
	case hocon.ObjectType:
		if len(val1.(hocon.Object)) != len(val2.(hocon.Object)) {
			return false
		}
		for key := range val1.(hocon.Object) {
			if _, ok := val2.(hocon.Object)[key]; !ok {
				return false
			}
			if !deepEqualHoconValue(val1.(hocon.Object)[key], val2.(hocon.Object)[key]) {
				return false
			}
		}
	case hocon.ArrayType:
		if len(val1.(hocon.Array)) != len(val2.(hocon.Array)) {
			return false
		}
		for i := range val1.(hocon.Array) {
			if !deepEqualHoconValue(val1.(hocon.Array)[i], val2.(hocon.Array)[i]) {
				return false
			}
		}
	case hocon.StringType, hocon.NumberType, hocon.BooleanType, hocon.NullType:
		return val1.String() == val2.String()
	default:
		return false
	}
	return true
}
