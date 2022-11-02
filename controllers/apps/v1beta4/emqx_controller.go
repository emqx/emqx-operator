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
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	emperror "emperror.dev/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	"github.com/emqx/emqx-operator/pkg/handler"
	"github.com/tidwall/gjson"
)

var _ reconcile.Reconciler = &EmqxBrokerReconciler{}

type EmqxReconciler struct {
	*handler.Handler
	Scheme *runtime.Scheme
	record.EventRecorder
}

func (r *EmqxReconciler) Do(ctx context.Context, instance appsv1beta4.Emqx) (ctrl.Result, error) {
	if !instance.IsPluginInitialized() {
		condition, err := r.initializedPluginList(instance)
		if condition != nil {
			instance.SetCondition(*condition)
			_ = r.Client.Status().Update(ctx, instance)
		}
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	condition, err := r.createOrUpdateResourceList(instance)
	if condition != nil {
		instance.SetCondition(*condition)
		_ = r.Client.Status().Update(ctx, instance)
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	status, err := r.updateEmqxStatus(instance)
	if err != nil {
		return ctrl.Result{}, err
	}
	instance.SetStatus(status)
	_ = r.Client.Status().Update(ctx, instance)

	return ctrl.Result{RequeueAfter: time.Duration(20) * time.Second}, nil
}

func (r *EmqxReconciler) initializedPluginList(instance appsv1beta4.Emqx) (*appsv1beta4.Condition, error) {
	plugins, err := r.createInitPluginList(instance)
	if err != nil {
		return nil, err
	}

	if err := r.CreateOrUpdateList(instance, r.Scheme, plugins); err != nil {
		if err != nil {
			r.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedCreateOrUpdate", err.Error())
			condition := appsv1beta4.NewCondition(
				appsv1beta4.ConditionPluginInitialized,
				corev1.ConditionFalse,
				"PluginInitializeFailed",
				err.Error(),
			)
			return condition, err
		}
	}
	condition := appsv1beta4.NewCondition(
		appsv1beta4.ConditionPluginInitialized,
		corev1.ConditionTrue,
		"PluginInitializeSuccessfully",
		"All default plugins initialized",
	)
	return condition, nil
}

func (r *EmqxReconciler) createOrUpdateResourceList(instance appsv1beta4.Emqx) (*appsv1beta4.Condition, error) {
	resources, err := r.createResourceList(instance)
	if err != nil {
		return nil, err
	}
	if err := r.CreateOrUpdateList(instance, r.Scheme, resources); err != nil {
		r.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedCreateOrUpdate", err.Error())
		condition := appsv1beta4.NewCondition(
			appsv1beta4.ConditionRunning,
			corev1.ConditionFalse,
			"FailedCreateOrUpdate",
			err.Error(),
		)
		return condition, err
	}
	return nil, nil
}

func (r *EmqxReconciler) updateEmqxStatus(instance appsv1beta4.Emqx) (appsv1beta4.Status, error) {
	var condition *appsv1beta4.Condition

	status := instance.GetStatus()
	status.Replicas = *instance.GetReplicas()

	emqxNodes, err := r.getNodeStatusesByAPI(instance)
	if err != nil {
		r.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedToGetNodeStatues", err.Error())
		condition = appsv1beta4.NewCondition(
			appsv1beta4.ConditionRunning,
			corev1.ConditionFalse,
			"FailedToGetNodeStatues",
			err.Error(),
		)
		status.SetCondition(*condition)
		return status, err
	}

	if emqxNodes != nil {
		readyReplicas := int32(0)
		for _, node := range emqxNodes {
			if node.NodeStatus == "Running" {
				readyReplicas++
			}
		}
		status.ReadyReplicas = readyReplicas
		status.EmqxNodes = emqxNodes
	}

	if status.Replicas == status.ReadyReplicas {
		condition = appsv1beta4.NewCondition(
			appsv1beta4.ConditionRunning,
			corev1.ConditionTrue,
			"ClusterReady",
			"All resources are ready",
		)
	} else {
		condition = appsv1beta4.NewCondition(
			appsv1beta4.ConditionRunning,
			corev1.ConditionFalse,
			"ClusterNotReady",
			"Some nodes are not ready",
		)
	}
	status.SetCondition(*condition)
	return status, nil
}

func (r *EmqxReconciler) createInitPluginList(instance appsv1beta4.Emqx) ([]client.Object, error) {
	pluginsList := &appsv1beta4.EmqxPluginList{}
	err := r.Client.List(context.Background(), pluginsList, client.InNamespace(instance.GetNamespace()))
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}
	initPluginsList := generateInitPluginList(instance, pluginsList)
	defaultPluginsConfig := generateDefaultPluginsConfig(instance)
	return append([]client.Object{defaultPluginsConfig}, initPluginsList...), nil
}

func (r *EmqxReconciler) createResourceList(instance appsv1beta4.Emqx) ([]client.Object, error) {
	var resources []client.Object

	if instance.IsRunning() {
		serviceTemplate := instance.GetServiceTemplate()
		ports, _ := r.getListenerPortsByAPI(instance)
		serviceTemplate.MergePorts(ports)
		instance.SetServiceTemplate(serviceTemplate)
	}

	headlessSvc, svc := generateService(instance)
	acl := generateEmqxACL(instance)
	sts := generateStatefulSet(instance)
	sts = updateStatefulSetForACL(sts, acl)
	sts = updateStatefulSetForPluginsConfig(sts, generateDefaultPluginsConfig(instance))

	if emqxEnterprise, ok := instance.(*appsv1beta4.EmqxEnterprise); ok {
		var license *corev1.Secret
		if instance.GetTemplate().Spec.EmqxContainer.EmqxLicense.SecretName != "" {
			license = &corev1.Secret{}
			if err := r.Client.Get(context.Background(), types.NamespacedName{Name: instance.GetTemplate().Spec.EmqxContainer.EmqxLicense.SecretName, Namespace: emqxEnterprise.GetNamespace()}, license); err != nil {
				return nil, err
			}
		} else {
			license = generateLicense(emqxEnterprise)
		}

		if license != nil {
			resources = append(resources, license)
			sts = updateStatefulSetForLicense(sts, license)
		}
	}

	resources = append(resources, acl, headlessSvc, svc, sts)
	return resources, nil
}

func (r *EmqxReconciler) getNodeStatusesByAPI(instance appsv1beta4.Emqx) ([]appsv1beta4.EmqxNode, error) {
	resp, body, err := r.Handler.RequestAPI(instance, instance.GetTemplate().Spec.EmqxContainer.Name, "GET", "admin", "public", "8081", "api/v4/nodes")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, emperror.Errorf("failed to get node statuses from API: %s", resp.Status)
	}

	emqxNodes := []appsv1beta4.EmqxNode{}
	data := gjson.GetBytes(body, "data")
	if err := json.Unmarshal([]byte(data.Raw), &emqxNodes); err != nil {
		return nil, emperror.Wrap(err, "failed to unmarshal node statuses")
	}
	return emqxNodes, nil
}

func (r *EmqxReconciler) getListenerPortsByAPI(instance appsv1beta4.Emqx) ([]corev1.ServicePort, error) {
	type emqxListener struct {
		Protocol string `json:"protocol"`
		ListenOn string `json:"listen_on"`
	}

	type emqxListeners struct {
		Node      string         `json:"node"`
		Listeners []emqxListener `json:"listeners"`
	}

	intersection := func(listeners1 []emqxListener, listeners2 []emqxListener) []emqxListener {
		hSection := map[string]struct{}{}
		ans := make([]emqxListener, 0)
		for _, listener := range listeners1 {
			hSection[listener.ListenOn] = struct{}{}
		}
		for _, listener := range listeners2 {
			_, ok := hSection[listener.ListenOn]
			if ok {
				ans = append(ans, listener)
				delete(hSection, listener.ListenOn)
			}
		}
		return ans
	}

	resp, body, err := r.Handler.RequestAPI(instance, instance.GetTemplate().Spec.EmqxContainer.Name, "GET", "admin", "public", "8081", "api/v4/listeners")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, err
	}

	listenerList := []emqxListeners{}
	data := gjson.GetBytes(body, "data")
	if err := json.Unmarshal([]byte(data.Raw), &listenerList); err != nil {
		return nil, emperror.Wrap(err, "failed to unmarshal node statuses")
	}

	var listeners []emqxListener
	for i := 0; i < len(listenerList)-1; i++ {
		listeners = intersection(listenerList[i].Listeners, listenerList[i+1].Listeners)
	}

	ports := []corev1.ServicePort{}
	for _, l := range listeners {
		var name string
		var protocol corev1.Protocol
		var strPort string
		var intPort int

		compile := regexp.MustCompile(".*(udp|dtls|sn).*")
		if compile.MatchString(l.Protocol) {
			protocol = corev1.ProtocolUDP
		} else {
			protocol = corev1.ProtocolTCP
		}

		if strings.Contains(l.ListenOn, ":") {
			_, strPort, err = net.SplitHostPort(l.ListenOn)
			if err != nil {
				strPort = l.ListenOn
			}
		} else {
			strPort = l.ListenOn
		}
		intPort, _ = strconv.Atoi(strPort)

		// Get name by protocol and port from API
		// protocol maybe like mqtt:wss:8084
		// protocol maybe like mqtt:tcp
		// We had to do something with the "protocol" to make it conform to the kubernetes service port name specification
		name = regexp.MustCompile(`:[\d]+`).ReplaceAllString(l.Protocol, "")
		name = strings.ReplaceAll(name, ":", "-")
		name = fmt.Sprintf("%s-%s", name, strPort)

		ports = append(ports, corev1.ServicePort{
			Name:       name,
			Protocol:   protocol,
			Port:       int32(intPort),
			TargetPort: intstr.FromInt(intPort),
		})
	}
	return ports, nil
}
