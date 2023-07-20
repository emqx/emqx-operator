package v2alpha2

import (
	"context"
	"net/http"

	emperror "emperror.dev/errors"
	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	"github.com/rory-z/go-hocon"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type syncConfig struct {
	*EMQXReconciler
}

func (s *syncConfig) reconcile(ctx context.Context, instance *appsv2alpha2.EMQX, r innerReq.RequesterInterface) subResult {
	// If core nodes is nil, the EMQX is in the process of being created
	if len(instance.Status.CoreNodes) == 0 {
		configMap := generateConfigMap(instance, instance.Spec.Config.Data)
		if err := s.Handler.CreateOrUpdateList(instance, s.Scheme, []client.Object{configMap}); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to create or update configMap")}
		}
		return subResult{}
	}

	if _, ok := instance.Annotations[appsv2alpha2.NeedUpdateConfigsAnnotationKey]; ok && instance.Status.IsConditionTrue(appsv2alpha2.CoreNodesReady) {
		// Delete readonly configs
		config, _ := hocon.ParseString(instance.Spec.Config.Data)
		configObj := config.GetRoot().(hocon.Object)
		delete(configObj, "node")
		delete(configObj, "cluster")
		delete(configObj, "dashboard")

		if err := putEMQXConfigsByAPI(r, instance.Spec.Config.Mode, configObj.String()); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to put emqx config")}
		}
		delete(instance.Annotations, appsv2alpha2.NeedUpdateConfigsAnnotationKey)
		if err := s.Client.Update(ctx, instance); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to update emqx instance")}
		}
	}

	config, err := getEMQXConfigsByAPI(r)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to get emqx config")}
	}

	configMap := generateConfigMap(instance, config)
	if err := s.Handler.CreateOrUpdateList(instance, s.Scheme, []client.Object{configMap}); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to create or update configMap")}
	}

	return subResult{}
}

func generateConfigMap(instance *appsv2alpha2.EMQX, data string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.ConfigsNamespacedName().Name,
			Namespace: instance.Namespace,
			Labels:    instance.Labels,
		},
		Data: map[string]string{
			"emqx.conf": data,
		},
	}
}

func getEMQXConfigsByAPI(r innerReq.RequesterInterface) (string, error) {
	url := r.GetURL("api/v5/configs")

	resp, body, err := r.Request("GET", url, nil, http.Header{
		"Accept": []string{"text/plain"},
	})
	if err != nil {
		return "", emperror.Wrapf(err, "failed to get API %s", url.String())
	}
	if resp.StatusCode != 200 {
		return "", emperror.Errorf("failed to get API %s, status : %s, body: %s", url.String(), resp.Status, body)
	}
	return string(body), nil
}

func putEMQXConfigsByAPI(r innerReq.RequesterInterface, mode, config string) error {
	url := r.GetURL("api/v5/configs", "mode="+mode)

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
