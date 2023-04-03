/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta4

import (
	"context"
	"fmt"
	"sort"
	"time"

	emperror "emperror.dev/errors"
	json "github.com/json-iterator/go"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	innerErr "github.com/emqx/emqx-operator/internal/errors"

	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	"github.com/emqx/emqx-operator/internal/handler"
)

type pluginByAPIReturn struct {
	Name        string
	Version     string
	Description string
	Active      bool
	Type        string
}

type pluginListByAPIReturn struct {
	Node    string
	Plugins []pluginByAPIReturn
}

// EmqxPluginReconciler reconciles a EmqxPlugin object
type EmqxPluginReconciler struct {
	*handler.Handler
	Clientset *kubernetes.Clientset
	Config    *rest.Config
}

func NewEmqxPluginReconciler(mgr manager.Manager) *EmqxPluginReconciler {
	return &EmqxPluginReconciler{
		Handler:   handler.NewHandler(mgr),
		Clientset: kubernetes.NewForConfigOrDie(mgr.GetConfig()),
		Config:    mgr.GetConfig(),
	}
}

//+kubebuilder:rbac:groups=apps.emqx.io,resources=emqxplugins,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.emqx.io,resources=emqxplugins/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.emqx.io,resources=emqxplugins/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EmqxPlugin object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *EmqxPluginReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)
	logger.V(1).Info("Reconcile EmqxPlugin")

	instance := &appsv1beta4.EmqxPlugin{}
	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	emqxList, err := r.getEmqxList(instance.Namespace, instance.Spec.Selector)
	if err != nil {
		return ctrl.Result{}, err
	}

	finalizer := "apps.emqx.io/finalizer"
	if instance.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(instance, finalizer) {
			for _, emqx := range emqxList {
				p, _ := newPortForwardAPI(ctx, r.Client, r.Clientset, r.Config, emqx)
				if p == nil {
					// The EMQX is not ready, requeue
					return ctrl.Result{RequeueAfter: time.Second}, nil
				}
				defer close(p.Options.StopChannel)
				if err := p.Options.ForwardPorts(); err != nil {
					return ctrl.Result{}, emperror.Wrap(err, "failed to forward ports")
				}
				if err := r.unloadPluginByAPI(p, instance.Spec.PluginName); err != nil {
					if innerErr.IsCommonError(err) {
						return ctrl.Result{RequeueAfter: time.Second}, nil
					}
					return ctrl.Result{}, err
				}
			}

			// Remove Finalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(instance, finalizer)
			err := r.Client.Update(ctx, instance)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(instance, finalizer) {
		controllerutil.AddFinalizer(instance, finalizer)
		err := r.Client.Update(ctx, instance)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	for _, emqx := range emqxList {
		if len(emqx.GetStatus().GetEmqxNodes()) == 0 {
			// The EMQX is not ready, requeue
			continue
		}

		equalPluginConfig, err := r.checkPluginConfig(instance, emqx)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !equalPluginConfig {
			if err = r.loadPluginConfig(instance, emqx); err != nil {
				if !k8sErrors.IsConflict(err) {
					return ctrl.Result{}, err
				}
			}
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}

		// The EMQX is not ready, requeue
		p, _ := newPortForwardAPI(ctx, r.Client, r.Clientset, r.Config, emqx)
		if p == nil {
			continue
		}
		defer close(p.Options.StopChannel)
		if err := p.Options.ForwardPorts(); err != nil {
			return ctrl.Result{}, emperror.Wrap(err, "failed to forward ports")
		}
		if err := r.checkPluginStatusByAPI(p, instance.Spec.PluginName); err != nil {
			if innerErr.IsCommonError(err) {
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{RequeueAfter: time.Duration(20) * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EmqxPluginReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1beta4.EmqxPlugin{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				// Ignore updates to CR status in which case metadata.Generation does not change
				return e.ObjectNew.GetGeneration() != e.ObjectOld.GetGeneration()
			},
		}).
		Complete(r)
}

func (r *EmqxPluginReconciler) checkPluginStatusByAPI(p *portForwardAPI, pluginName string) error {
	list, err := r.getPluginsByAPI(p)
	if err != nil {
		return err
	}
	for _, node := range list {
		for _, plugin := range node.Plugins {
			if plugin.Name == pluginName {
				if !plugin.Active {
					err := r.doLoadPluginByAPI(p, node.Node, plugin.Name, "reload")
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (r *EmqxPluginReconciler) unloadPluginByAPI(p *portForwardAPI, pluginName string) error {
	list, err := r.getPluginsByAPI(p)
	if err != nil {
		return err
	}
	for _, node := range list {
		for _, plugin := range node.Plugins {
			if plugin.Name == pluginName {
				err := r.doLoadPluginByAPI(p, node.Node, plugin.Name, "unload")
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (r *EmqxPluginReconciler) doLoadPluginByAPI(p *portForwardAPI, nodeName, pluginName, reloadOrUnload string) error {
	resp, _, err := p.requestAPI("PUT", fmt.Sprintf("api/v4/nodes/%s/plugins/%s/%s", nodeName, pluginName, reloadOrUnload), nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return emperror.Errorf("request api failed: %s", resp.Status)
	}
	return nil
}

func (r *EmqxPluginReconciler) getPluginsByAPI(p *portForwardAPI) ([]pluginListByAPIReturn, error) {
	var data []pluginListByAPIReturn
	resp, body, err := p.requestAPI("GET", "api/v4/plugins", nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, emperror.Errorf("request api failed: %s", resp.Status)
	}

	err = json.Unmarshal([]byte(gjson.GetBytes(body, "data").String()), &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (r *EmqxPluginReconciler) checkPluginConfig(plugin *appsv1beta4.EmqxPlugin, emqx appsv1beta4.Emqx) (bool, error) {
	pluginConfigStr := generateConfigStr(plugin)

	pluginsConfig, err := r.getPluginsConfig(emqx)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	configMapStr, err := json.ConfigCompatibleWithStandardLibrary.Marshal(pluginsConfig.Data)
	if err != nil {
		return false, err
	}

	path := plugin.Spec.PluginName + "\\.conf"
	storePluginConfig := gjson.GetBytes(configMapStr, path)
	if storePluginConfig.Exists() {
		if storePluginConfig.String() == pluginConfigStr {
			return true, nil
		}
	}
	return false, nil
}

func (r *EmqxPluginReconciler) loadPluginConfig(plugin *appsv1beta4.EmqxPlugin, emqx appsv1beta4.Emqx) error {
	pluginConfigStr := generateConfigStr(plugin)

	pluginsConfig, err := r.getPluginsConfig(emqx)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	configMapStr, err := json.ConfigCompatibleWithStandardLibrary.Marshal(pluginsConfig.Data)
	if err != nil {
		return err
	}

	path := plugin.Spec.PluginName + "\\.conf"

	newConfigMapStr, err := sjson.SetBytes(configMapStr, path, pluginConfigStr)
	if err != nil {
		return err
	}

	configData := map[string]string{}
	if err := json.Unmarshal(newConfigMapStr, &configData); err != nil {
		return err
	}
	pluginsConfig.Data = configData

	// Update plugin config
	if err := r.Handler.Update(pluginsConfig); err != nil {
		return err
	}
	return nil
}

func (r *EmqxPluginReconciler) getEmqxList(namespace string, labels map[string]string) ([]appsv1beta4.Emqx, error) {
	var emqxList []appsv1beta4.Emqx

	emqxBrokerList := &appsv1beta4.EmqxBrokerList{}
	if err := r.Client.List(context.Background(), emqxBrokerList, client.InNamespace(namespace), client.MatchingLabels(labels)); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return nil, err
		}
	}
	for _, emqxBroker := range emqxBrokerList.Items {
		emqxList = append(emqxList, &emqxBroker)
	}

	emqxEnterpriseList := &appsv1beta4.EmqxEnterpriseList{}
	if err := r.Client.List(context.Background(), emqxEnterpriseList, client.InNamespace(namespace), client.MatchingLabels(labels)); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return nil, err
		}
	}
	for _, emqxEnterprise := range emqxEnterpriseList.Items {
		emqxList = append(emqxList, &emqxEnterprise)
	}

	return emqxList, nil
}

func (r *EmqxPluginReconciler) getPluginsConfig(emqx appsv1beta4.Emqx) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	if err := r.Client.Get(
		context.TODO(),
		client.ObjectKey{
			Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "plugins-config"),
			Namespace: emqx.GetNamespace(),
		},
		configMap,
	); err != nil {
		return nil, err
	}
	return configMap, nil
}

func generateConfigStr(plugin *appsv1beta4.EmqxPlugin) string {
	keys := make([]string, 0)
	for k := range plugin.Spec.Config {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var config string
	for _, k := range keys {
		config += fmt.Sprintln(k, " = ", plugin.Spec.Config[k])
	}
	return config
}
