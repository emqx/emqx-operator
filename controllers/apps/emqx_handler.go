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

package apps

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appsv1beta3 "github.com/emqx/emqx-operator/apis/apps/v1beta3"
)

var _ reconcile.Reconciler = &EmqxBrokerReconciler{}

type EmqxReconciler struct {
	Handler
}

func (r *EmqxReconciler) Do(ctx context.Context, instance appsv1beta3.Emqx) error {
	var resources []client.Object

	var sts *appsv1.StatefulSet
	sts = generateStatefulSetDef(instance)

	headlessSvc, svc, sts := generateSvc(instance, sts)
	resources = append(resources, headlessSvc, svc)

	acl, sts := generateAcl(instance, sts)
	resources = append(resources, acl)

	module, sts := generateLoadedModules(instance, sts)
	resources = append(resources, module)

	mqttsCerts, sts := generateMQTTSCertificate(instance, sts)
	if mqttsCerts != nil {
		resources = append(resources, mqttsCerts)
	}

	wssCerts, sts := generateWSSCertificate(instance, sts)
	if wssCerts != nil {
		resources = append(resources, wssCerts)
	}

	license, sts := generateLicense(instance, sts)
	if license != nil {
		resources = append(resources, license)
	}

	loadedPlugins, sts := generateLoadedPlugins(instance, sts)
	emptyPluginsConfig, sts := generateEmptyPlugins(instance, sts)
	resources = append(resources, emptyPluginsConfig, loadedPlugins)

	// First reconcile
	if len(instance.GetStatus().Conditions) == 0 {
		pluginsList := &appsv1beta3.EmqxPluginList{}
		err := r.Client.List(ctx, pluginsList, client.InNamespace(instance.GetNamespace()))
		if err != nil && !k8sErrors.IsNotFound(err) {
			return err
		}

		pluginResourceList := generateInitPluginList(instance, pluginsList)
		resources = append(resources, pluginResourceList...)

		// StateFulSet should be created last
		resources = append(resources, sts)
	} else {
		// StateFulSet should be created last
		resources = append(resources, sts)
	}

	ownerRef := metav1.NewControllerRef(instance, instance.GetObjectKind().GroupVersionKind())

	for _, resource := range resources {
		addOwnerRefToObject(resource, *ownerRef)

		var err error
		names := appsv1beta3.Names{Object: instance}
		switch resource.GetName() {
		case names.PluginsConfig(), names.LoadedPlugins():
			// Only create plugins config and loaded plugins, do not update
			configMap := &corev1.ConfigMap{}
			err = r.Get(context.TODO(), client.ObjectKeyFromObject(resource), configMap)
			if k8sErrors.IsNotFound(err) {
				nothing := func(client.Object) error { return nil }
				err = r.Handler.doCreate(resource, nothing)
			} else {
				err = nil
			}
		case names.MQTTSCertificate():
			postFun := func(instance client.Object) error {
				return r.Handler.ExecToPods(instance, "emqx", "emqx_ctl listeners restart mqtt:wss:external")
			}
			err = r.Handler.CreateOrUpdate(resource, postFun)
		case names.WSSCertificate():
			postFun := func(instance client.Object) error {
				return r.Handler.ExecToPods(instance, "emqx", "emqx_ctl listeners restart mqtt:wss:external")
			}
			err = r.Handler.CreateOrUpdate(resource, postFun)

		default:
			nothing := func(client.Object) error { return nil }
			err = r.Handler.CreateOrUpdate(resource, nothing)
		}

		if err != nil {
			r.EventRecorder.Event(instance, corev1.EventTypeWarning, "Reconciled", err.Error())
			instance.SetFailedCondition(err.Error())
			instance.DescConditionsByTime()
			_ = r.Status().Update(ctx, instance)
			return err
		}
	}

	instance.SetRunningCondition("Reconciled")
	instance.DescConditionsByTime()
	_ = r.Status().Update(ctx, instance)
	return nil
}

func generateStatefulSetDef(instance appsv1beta3.Emqx) *appsv1.StatefulSet {
	sts := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.GetName(),
			Namespace: instance.GetNamespace(),
			Labels:    instance.GetLabels(),
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: instance.GetReplicas(),
			Selector: &metav1.LabelSelector{
				MatchLabels: instance.GetLabels(),
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      instance.GetLabels(),
					Annotations: instance.GetAnnotations(),
				},
				Spec: corev1.PodSpec{
					Affinity:         instance.GetAffinity(),
					Tolerations:      instance.GetToleRations(),
					NodeName:         instance.GetNodeName(),
					NodeSelector:     instance.GetNodeSelector(),
					ImagePullSecrets: instance.GetImagePullSecrets(),
					SecurityContext:  instance.GetSecurityContext(),
					InitContainers:   instance.GetInitContainers(),
					Containers: []corev1.Container{
						{
							Name:            "emqx",
							Image:           instance.GetImage(),
							ImagePullPolicy: instance.GetImagePullPolicy(),
							Resources:       instance.GetResource(),
							Env:             instance.GetEnv(),
							Args:            instance.GetArgs(),
							ReadinessProbe:  instance.GetReadinessProbe(),
							LivenessProbe:   instance.GetLivenessProbe(),
							StartupProbe:    instance.GetStartupProbe(),
							Lifecycle: &corev1.Lifecycle{
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"/opt/emqx/bin/emqx_ctl",
											"cluster",
											"leave",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	terminationGracePeriodSeconds := int64(60)
	sts.Spec.Template.Spec.TerminationGracePeriodSeconds = &terminationGracePeriodSeconds

	return generateVolume(instance, sts)
}

func generateInitPluginList(instance appsv1beta3.Emqx, exitsPluginList *appsv1beta3.EmqxPluginList) []client.Object {
	matchedPluginList := []appsv1beta3.EmqxPlugin{}
	for _, exitsPlugin := range exitsPluginList.Items {
		selector, _ := labels.ValidatedSelectorFromSet(exitsPlugin.Spec.Selector)
		if selector.Empty() || !selector.Matches(labels.Set(instance.GetLabels())) {
			continue
		}
		matchedPluginList = append(matchedPluginList, exitsPlugin)
	}

	isExitsPlugin := func(pluginName string, pluginList []appsv1beta3.EmqxPlugin) bool {
		for _, plugin := range pluginList {
			if plugin.Spec.PluginName == pluginName {
				return true
			}
		}
		return false
	}

	pluginList := []client.Object{}
	// Default plugins
	if !isExitsPlugin("emqx_management", matchedPluginList) {
		emqxManagement := &appsv1beta3.EmqxPlugin{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps.emqx.io/v1beta3",
				Kind:       "EmqxPlugin",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-management", instance.GetName()),
				Namespace: instance.GetNamespace(),
				Labels:    instance.GetLabels(),
			},
			Spec: appsv1beta3.EmqxPluginSpec{
				PluginName: "emqx_management",
				Selector:   instance.GetLabels(),
				Config: map[string]string{
					"management.listener.http":              "8081",
					"management.default_application.id":     "admin",
					"management.default_application.secret": "public",
				},
			},
		}
		pluginList = append(pluginList, emqxManagement)
	}

	if !isExitsPlugin("emqx_dashboard", matchedPluginList) {
		emqxDashboard := &appsv1beta3.EmqxPlugin{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps.emqx.io/v1beta3",
				Kind:       "EmqxPlugin",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-dashboard", instance.GetName()),
				Namespace: instance.GetNamespace(),
				Labels:    instance.GetLabels(),
			},
			Spec: appsv1beta3.EmqxPluginSpec{
				PluginName: "emqx_dashboard",
				Selector:   instance.GetLabels(),
				Config: map[string]string{
					"dashboard.listener.http":         "18083",
					"dashboard.default_user.login":    "admin",
					"dashboard.default_user.password": "public",
				},
			},
		}
		pluginList = append(pluginList, emqxDashboard)
	}

	if !isExitsPlugin("emqx_rule_engine", matchedPluginList) {
		emqxRuleEngine := &appsv1beta3.EmqxPlugin{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps.emqx.io/v1beta3",
				Kind:       "EmqxPlugin",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-rule-engine", instance.GetName()),
				Namespace: instance.GetNamespace(),
				Labels:    instance.GetLabels(),
			},
			Spec: appsv1beta3.EmqxPluginSpec{
				PluginName: "emqx_rule_engine",
				Selector:   instance.GetLabels(),
				Config:     map[string]string{},
			},
		}
		pluginList = append(pluginList, emqxRuleEngine)
	}

	if !isExitsPlugin("emqx_retainer", matchedPluginList) {
		emqxRetainer := &appsv1beta3.EmqxPlugin{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps.emqx.io/v1beta3",
				Kind:       "EmqxPlugin",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-retainer", instance.GetName()),
				Namespace: instance.GetNamespace(),
				Labels:    instance.GetLabels(),
			},
			Spec: appsv1beta3.EmqxPluginSpec{
				PluginName: "emqx_retainer",
				Selector:   instance.GetLabels(),
				Config:     map[string]string{},
			},
		}
		pluginList = append(pluginList, emqxRetainer)
	}

	return pluginList
}

func generateEmptyPlugins(instance appsv1beta3.Emqx, sts *appsv1.StatefulSet) (*corev1.ConfigMap, *appsv1.StatefulSet) {
	names := appsv1beta3.Names{Object: instance}

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    instance.GetLabels(),
			Namespace: instance.GetNamespace(),
			Name:      names.PluginsConfig(),
		},
	}

	container := sts.Spec.Template.Spec.Containers[0]
	container.Env = append(
		container.Env,
		corev1.EnvVar{
			Name:  "EMQX_PLUGINS__ETC_DIR",
			Value: "/mounted/plugins/etc",
		},
	)
	container.VolumeMounts = append(
		container.VolumeMounts,
		corev1.VolumeMount{
			Name:      cm.Name,
			MountPath: "/mounted/plugins/etc",
		},
	)
	sts.Spec.Template.Spec.Containers = []corev1.Container{container}

	sts.Spec.Template.Spec.Volumes = append(
		sts.Spec.Template.Spec.Volumes,
		corev1.Volume{
			Name: cm.Name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cm.Name,
					},
				},
			},
		},
	)
	return cm, sts
}

func generateLoadedPlugins(instance appsv1beta3.Emqx, sts *appsv1.StatefulSet) (*corev1.ConfigMap, *appsv1.StatefulSet) {
	names := appsv1beta3.Names{Object: instance}
	loadedPlugins := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    instance.GetLabels(),
			Namespace: instance.GetNamespace(),
			Name:      names.LoadedPlugins(),
		},
		Data: map[string]string{
			"loaded_plugins": "{emqx_management, true}.\n{emqx_dashboard, true}.\n{emqx_retainer, true}.\n{emqx_rule_engine, true}.\n",
		},
	}

	container := sts.Spec.Template.Spec.Containers[0]
	container.VolumeMounts = append(
		container.VolumeMounts,
		corev1.VolumeMount{
			Name:      loadedPlugins.Name,
			MountPath: "/mounted/plugins/data",
		},
	)
	container.Env = append(
		container.Env,
		corev1.EnvVar{
			Name:  "EMQX_PLUGINS__LOADED_FILE",
			Value: "/mounted/plugins/data/loaded_plugins",
		},
	)
	sts.Spec.Template.Spec.Containers = []corev1.Container{container}

	sts.Spec.Template.Spec.Volumes = append(
		sts.Spec.Template.Spec.Volumes,
		corev1.Volume{
			Name: loadedPlugins.Name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: loadedPlugins.Name,
					},
				},
			},
		},
	)
	return loadedPlugins, sts
}

func generateSvc(instance appsv1beta3.Emqx, sts *appsv1.StatefulSet) (*corev1.Service, *corev1.Service, *appsv1.StatefulSet) {
	names := appsv1beta3.Names{Object: instance}
	listener := instance.GetListener()

	headlessSvcIPFamilyPolicy := corev1.IPFamilyPolicySingleStack
	headlessSvc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    instance.GetLabels(),
			Name:      names.HeadlessSvc(),
			Namespace: instance.GetNamespace(),
		},
		Spec: corev1.ServiceSpec{
			Selector:       instance.GetLabels(),
			ClusterIP:      corev1.ClusterIPNone,
			IPFamilyPolicy: &headlessSvcIPFamilyPolicy,
		},
	}
	sts.Spec.ServiceName = headlessSvc.Name

	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        instance.GetName(),
			Namespace:   instance.GetNamespace(),
			Annotations: listener.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Type:                     listener.Type,
			LoadBalancerIP:           listener.LoadBalancerIP,
			LoadBalancerSourceRanges: listener.LoadBalancerSourceRanges,
			ExternalIPs:              listener.ExternalIPs,
			Selector:                 instance.GetLabels(),
		},
	}
	labels := listener.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	for k, v := range instance.GetLabels() {
		labels[k] = v
	}
	svc.Labels = labels

	container := sts.Spec.Template.Spec.Containers[0]

	for _, portName := range [6]string{"API", "Dashboard", "MQTT", "MQTTS", "WS", "WSS"} {
		port := int32(reflect.ValueOf(listener).FieldByName(portName).FieldByName("Port").Int())
		if port != 0 {
			nodePort := int32(reflect.ValueOf(listener).FieldByName(portName).FieldByName("NodePort").Int())
			name := strings.ToLower(portName)

			var envName string
			switch name {
			default:
				envName = fmt.Sprintf("EMQX_LISTENER__%s__EXTERNAL", strings.ToUpper(portName))
			case "mqtt":
				envName = "EMQX_LISTENER__TCP__EXTERNAL"
			case "mqtts":
				envName = "EMQX_LISTENER__SSL__EXTERNAL"
			case "dashboard":
				envName = "EMQX_DASHBOARD__LISTENER__HTTP"
			case "api":
				envName = "EMQX_MANAGEMENT__LISTENER__HTTP"
			}

			container.Env = append(container.Env, corev1.EnvVar{
				Name:  envName,
				Value: fmt.Sprint(port),
			})

			container.Ports = append(container.Ports, corev1.ContainerPort{
				Name:          name,
				ContainerPort: port,
				Protocol:      corev1.ProtocolTCP,
			})

			svc.Spec.Ports = append(svc.Spec.Ports, corev1.ServicePort{
				Name:     name,
				Port:     port,
				NodePort: nodePort,
				Protocol: "TCP",
				TargetPort: intstr.IntOrString{
					Type:   0,
					IntVal: port,
				},
			})

			if name == "api" {
				headlessSvc.Spec.Ports = append(headlessSvc.Spec.Ports, corev1.ServicePort{
					Name:     name,
					Port:     port,
					Protocol: "TCP",
					TargetPort: intstr.IntOrString{
						Type:   0,
						IntVal: port,
					},
				})
			}
		}
	}
	sts.Spec.Template.Spec.Containers = []corev1.Container{container}
	return headlessSvc, svc, sts
}

func generateAcl(instance appsv1beta3.Emqx, sts *appsv1.StatefulSet) (*corev1.ConfigMap, *appsv1.StatefulSet) {
	names := appsv1beta3.Names{Object: instance}
	acls := &appsv1beta3.ACLList{
		Items: instance.GetACL(),
	}
	aclString := acls.String()

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    instance.GetLabels(),
			Namespace: instance.GetNamespace(),
			Name:      names.ACL(),
		},
		Data: map[string]string{"acl.conf": aclString},
	}

	annotations := sts.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["ACL/Base64EncodeConfig"] = base64.StdEncoding.EncodeToString([]byte(aclString))
	sts.Annotations = annotations

	container := sts.Spec.Template.Spec.Containers[0]
	container.VolumeMounts = append(
		container.VolumeMounts,
		corev1.VolumeMount{
			Name:      cm.Name,
			MountPath: "/mounted/acl",
		},
	)
	container.Env = append(
		container.Env,
		corev1.EnvVar{
			Name:  "EMQX_ACL_FILE",
			Value: "/mounted/acl/acl.conf",
		},
	)
	sts.Spec.Template.Spec.Containers = []corev1.Container{container}

	sts.Spec.Template.Spec.Volumes = append(
		sts.Spec.Template.Spec.Volumes,
		corev1.Volume{
			Name: cm.Name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cm.Name,
					},
				},
			},
		},
	)

	return cm, sts
}

func generateLoadedModules(instance appsv1beta3.Emqx, sts *appsv1.StatefulSet) (*corev1.ConfigMap, *appsv1.StatefulSet) {
	names := appsv1beta3.Names{Object: instance}
	var loadedModulesString string
	switch obj := instance.(type) {
	case *appsv1beta3.EmqxBroker:
		modules := &appsv1beta3.EmqxBrokerModuleList{
			Items: obj.Spec.EmqxTemplate.Modules,
		}
		loadedModulesString = modules.String()
	case *appsv1beta3.EmqxEnterprise:
		data, _ := json.Marshal(obj.Spec.EmqxTemplate.Modules)
		loadedModulesString = string(data)
	}
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    instance.GetLabels(),
			Namespace: instance.GetNamespace(),
			Name:      names.LoadedModules(),
		},
		Data: map[string]string{"loaded_modules": loadedModulesString},
	}

	annotations := sts.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["LoadedModules/Base64EncodeConfig"] = base64.StdEncoding.EncodeToString([]byte(loadedModulesString))
	sts.Annotations = annotations

	container := sts.Spec.Template.Spec.Containers[0]
	container.VolumeMounts = append(
		container.VolumeMounts,
		corev1.VolumeMount{
			Name:      cm.Name,
			MountPath: "/mounted/modules",
		},
	)
	container.Env = append(
		container.Env,
		corev1.EnvVar{
			Name:  "EMQX_MODULES__LOADED_FILE",
			Value: "/mounted/modules/loaded_modules",
		},
	)
	sts.Spec.Template.Spec.Containers = []corev1.Container{container}

	sts.Spec.Template.Spec.Volumes = append(
		sts.Spec.Template.Spec.Volumes,
		corev1.Volume{
			Name: cm.Name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cm.Name,
					},
				},
			},
		},
	)
	return cm, sts
}

func generateMQTTSCertificate(instance appsv1beta3.Emqx, sts *appsv1.StatefulSet) (*corev1.Secret, *appsv1.StatefulSet) {
	names := appsv1beta3.Names{Object: instance}
	cert := instance.GetListener().MQTTS.Cert
	if reflect.ValueOf(cert).IsZero() {
		return nil, sts
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    instance.GetLabels(),
			Namespace: instance.GetNamespace(),
			Name:      names.MQTTSCertificate(),
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"ca.crt":  cert.Data.CaCert,
			"tls.crt": cert.Data.TLSCert,
			"tls.key": cert.Data.TLSKey,
		},
	}

	if cert.StringData.CaCert != "" || cert.StringData.TLSCert != "" || cert.StringData.TLSKey != "" {
		secret.StringData = map[string]string{
			"ca.crt":  cert.StringData.CaCert,
			"tls.crt": cert.StringData.TLSCert,
			"tls.key": cert.StringData.TLSKey,
		}
	}

	container := sts.Spec.Template.Spec.Containers[0]
	container.VolumeMounts = append(
		container.VolumeMounts,
		corev1.VolumeMount{
			Name:      secret.Name,
			MountPath: "/mounted/certs/mqtts",
			ReadOnly:  true,
		},
	)
	container.Env = append(
		container.Env,
		corev1.EnvVar{
			Name:  "EMQX_LISTENER__SSL__EXTERNAL__CERTFILE",
			Value: "/mounted/certs/mqtts/tls.crt",
		},
		corev1.EnvVar{
			Name:  "EMQX_LISTENER__SSL__EXTERNAL__KEYFILE",
			Value: "/mounted/certs/mqtts/tls.key",
		},
	)
	if len(cert.Data.CaCert) != 0 || cert.StringData.CaCert != "" {
		container.Env = append(
			container.Env,
			corev1.EnvVar{
				Name:  "EMQX_LISTENER__SSL__EXTERNAL__CACERTFILE",
				Value: "/mounted/certs/mqtts/ca.crt",
			},
		)
	}
	sts.Spec.Template.Spec.Containers = []corev1.Container{container}
	sts.Spec.Template.Spec.Volumes = append(
		sts.Spec.Template.Spec.Volumes,
		corev1.Volume{
			Name: secret.Name,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secret.Name,
				},
			},
		},
	)

	return secret, sts
}

func generateWSSCertificate(instance appsv1beta3.Emqx, sts *appsv1.StatefulSet) (*corev1.Secret, *appsv1.StatefulSet) {
	names := appsv1beta3.Names{Object: instance}
	cert := instance.GetListener().WSS.Cert
	if reflect.ValueOf(cert).IsZero() {
		return nil, sts
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    instance.GetLabels(),
			Namespace: instance.GetNamespace(),
			Name:      names.WSSCertificate(),
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"ca.crt":  cert.Data.CaCert,
			"tls.crt": cert.Data.TLSCert,
			"tls.key": cert.Data.TLSKey,
		},
	}

	if cert.StringData.CaCert != "" || cert.StringData.TLSCert != "" || cert.StringData.TLSKey != "" {
		secret.StringData = map[string]string{
			"ca.crt":  cert.StringData.CaCert,
			"tls.crt": cert.StringData.TLSCert,
			"tls.key": cert.StringData.TLSKey,
		}
	}

	container := sts.Spec.Template.Spec.Containers[0]
	container.VolumeMounts = append(
		container.VolumeMounts,
		corev1.VolumeMount{
			Name:      secret.Name,
			MountPath: "/mounted/certs/wss",
			ReadOnly:  true,
		},
	)
	container.Env = append(
		container.Env,
		corev1.EnvVar{
			Name:  "EMQX_LISTENER__WSS__EXTERNAL__CERTFILE",
			Value: "/mounted/certs/wss/tls.crt",
		},
		corev1.EnvVar{
			Name:  "EMQX_LISTENER__WSS__EXTERNAL__KEYFILE",
			Value: "/mounted/certs/wss/tls.key",
		},
	)
	if len(cert.Data.CaCert) != 0 || cert.StringData.CaCert != "" {
		container.Env = append(
			container.Env,
			corev1.EnvVar{
				Name:  "EMQX_LISTENER__WSS__EXTERNAL__CACERTFILE",
				Value: "/mounted/certs/wss/ca.crt",
			},
		)
	}
	sts.Spec.Template.Spec.Containers = []corev1.Container{container}
	sts.Spec.Template.Spec.Volumes = append(
		sts.Spec.Template.Spec.Volumes,
		corev1.Volume{
			Name: secret.Name,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secret.Name,
				},
			},
		},
	)

	return secret, sts
}

func generateLicense(instance appsv1beta3.Emqx, sts *appsv1.StatefulSet) (*corev1.Secret, *appsv1.StatefulSet) {
	names := appsv1beta3.Names{Object: instance}
	emqxEnterprise, ok := instance.(*appsv1beta3.EmqxEnterprise)
	if !ok {
		return nil, sts
	}
	if reflect.ValueOf(emqxEnterprise.GetLicense()).IsZero() {
		return nil, sts
	}
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    instance.GetLabels(),
			Namespace: instance.GetNamespace(),
			Name:      names.License(),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{"emqx.lic": emqxEnterprise.GetLicense().Data},
	}
	if emqxEnterprise.GetLicense().StringData != "" {
		secret.StringData = map[string]string{"emqx.lic": emqxEnterprise.GetLicense().StringData}
	}

	container := sts.Spec.Template.Spec.Containers[0]
	container.VolumeMounts = append(
		container.VolumeMounts,
		corev1.VolumeMount{
			Name:      secret.Name,
			MountPath: "/mounted/license",
			ReadOnly:  true,
		},
	)
	container.Env = append(
		container.Env,
		corev1.EnvVar{
			Name:  "EMQX_LICENSE__FILE",
			Value: "/mounted/license/emqx.lic",
		},
	)
	sts.Spec.Template.Spec.Containers = []corev1.Container{container}
	sts.Spec.Template.Spec.Volumes = append(
		sts.Spec.Template.Spec.Volumes,
		corev1.Volume{
			Name: secret.Name,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secret.Name,
				},
			},
		},
	)

	return secret, sts
}

func generateVolume(instance appsv1beta3.Emqx, sts *appsv1.StatefulSet) *appsv1.StatefulSet {
	names := appsv1beta3.Names{Object: instance}

	dataName := names.Data()
	logName := names.Log()

	container := sts.Spec.Template.Spec.Containers[0]
	container.VolumeMounts = append(
		container.VolumeMounts,
		corev1.VolumeMount{
			Name:      dataName,
			MountPath: "/opt/emqx/data",
		},
		corev1.VolumeMount{
			Name:      logName,
			MountPath: "/opt/emqx/log",
		},
	)

	if reflect.ValueOf(instance.GetPersistent()).IsZero() {
		sts.Spec.Template.Spec.Volumes = append(
			sts.Spec.Template.Spec.Volumes,
			generateEmptyDirVolume(dataName),
			generateEmptyDirVolume(logName),
		)
	} else {
		sts.Spec.VolumeClaimTemplates = append(
			sts.Spec.VolumeClaimTemplates,
			generateVolumeClaimTemplate(instance, dataName),
			generateVolumeClaimTemplate(instance, logName),
		)

	}

	container.VolumeMounts = append(container.VolumeMounts, instance.GetExtraVolumeMounts()...)
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, instance.GetExtraVolumes()...)

	sts.Spec.Template.Spec.Containers = []corev1.Container{container}
	return sts
}

func generateEmptyDirVolume(Name string) corev1.Volume {
	return corev1.Volume{
		Name: Name,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
}

func generateVolumeClaimTemplate(instance appsv1beta3.Emqx, Name string) corev1.PersistentVolumeClaim {
	template := instance.GetPersistent()
	pvc := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Name,
			Namespace: instance.GetNamespace(),
		},
		Spec: template,
	}
	if pvc.Spec.AccessModes == nil {
		pvc.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}
	if pvc.Spec.VolumeMode == nil {
		fileSystem := corev1.PersistentVolumeFilesystem
		pvc.Spec.VolumeMode = &fileSystem
	}
	return pvc
}

func addOwnerRefToObject(obj metav1.Object, ownerRef metav1.OwnerReference) {
	obj.SetOwnerReferences(append(obj.GetOwnerReferences(), ownerRef))
}
