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

package v1beta2

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts this version to the Hub version (v1).
func (src *EmqxBroker) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1beta3.EmqxBroker)

	dst.ObjectMeta = src.ObjectMeta

	labels := make(map[string]string)
	for k, v := range src.ObjectMeta.Labels {
		labels[k] = v
	}
	for k, v := range src.Spec.Labels {
		labels[k] = v
	}
	dst.ObjectMeta.Labels = labels

	annotations := make(map[string]string)
	for k, v := range src.ObjectMeta.Annotations {
		annotations[k] = v
	}
	for k, v := range src.Spec.Annotations {
		annotations[k] = v
	}
	dst.ObjectMeta.Annotations = annotations

	// ServiceTemplate
	dst.Spec.EmqxTemplate.ServiceTemplate = convertToListener(src)

	// EmqxConfig
	dst.Spec.EmqxTemplate.EmqxConfig, dst.Spec.EmqxTemplate.Env = conversionToEmqxConfig(src.Spec.Env)

	dst.Spec.Persistent = src.Spec.Storage

	aclList := &ACLList{
		Items: src.Spec.EmqxTemplate.ACL,
	}
	dst.Spec.EmqxTemplate.ACL = aclList.Strings()
	dst.Spec.EmqxTemplate.Modules = src.Spec.EmqxTemplate.Modules

	// Spec
	dst.Spec.Replicas = src.Spec.Replicas
	dst.Spec.Affinity = src.Spec.Affinity
	dst.Spec.ToleRations = src.Spec.ToleRations
	dst.Spec.NodeSelector = src.Spec.NodeSelector

	dst.Spec.EmqxTemplate.Image = src.Spec.Image
	dst.Spec.EmqxTemplate.Resources = src.Spec.Resources
	dst.Spec.EmqxTemplate.ImagePullPolicy = src.Spec.ImagePullPolicy
	dst.Spec.EmqxTemplate.ExtraVolumes = src.Spec.ExtraVolumes
	dst.Spec.EmqxTemplate.ExtraVolumeMounts = src.Spec.ExtraVolumeMounts

	// Status
	for _, condition := range src.Status.Conditions {
		dst.Status.Conditions = append(
			dst.Status.Conditions,
			v1beta3.Condition{
				Type:               v1beta3.ConditionType(condition.Type),
				Status:             condition.Status,
				LastUpdateTime:     condition.LastUpdateTime,
				LastUpdateAt:       condition.LastUpdateAt,
				LastTransitionTime: condition.LastTransitionTime,
				Reason:             condition.Reason,
				Message:            condition.Message,
			},
		)
	}

	// +kubebuilder:docs-gen:collapse=rote conversion
	return nil
}

// ConvertFrom converts from the Hub version (v1) to this version.
func (dst *EmqxBroker) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1beta3.EmqxBroker)

	dst.ObjectMeta = src.ObjectMeta
	dst.Spec.Labels = src.Labels
	dst.Spec.Annotations = src.Annotations

	// Listener
	dst.Spec.EmqxTemplate.Listener = convertFromListener(src)

	if !reflect.ValueOf(src.Spec.Persistent).IsZero() {
		dst.Spec.Storage = src.Spec.Persistent
	}
	// dst.Spec.EmqxTemplate.ACL = src.Spec.EmqxTemplate.ACL
	dst.Spec.EmqxTemplate.Modules = src.Spec.EmqxTemplate.Modules

	// Spec
	dst.Spec.Replicas = src.Spec.Replicas
	dst.Spec.Affinity = src.Spec.Affinity
	dst.Spec.ToleRations = src.Spec.ToleRations
	dst.Spec.NodeSelector = src.Spec.NodeSelector

	dst.Spec.Image = src.Spec.EmqxTemplate.Image
	dst.Spec.Resources = src.Spec.EmqxTemplate.Resources
	dst.Spec.ImagePullPolicy = src.Spec.EmqxTemplate.ImagePullPolicy
	dst.Spec.ExtraVolumes = src.Spec.EmqxTemplate.ExtraVolumes
	dst.Spec.ExtraVolumeMounts = src.Spec.EmqxTemplate.ExtraVolumeMounts
	//dst.Spec.Env = src.Spec.EmqxTemplate.Env
	dst.Spec.Env = converFromEnvAndConfig(src.Spec.EmqxTemplate.Env, src.Spec.EmqxTemplate.EmqxConfig)

	// Status
	for _, condition := range src.Status.Conditions {
		dst.Status.Conditions = append(
			dst.Status.Conditions,
			Condition{
				Type:               ConditionType(condition.Type),
				Status:             condition.Status,
				LastUpdateTime:     condition.LastUpdateTime,
				LastUpdateAt:       condition.LastUpdateAt,
				LastTransitionTime: condition.LastTransitionTime,
				Reason:             condition.Reason,
				Message:            condition.Message,
			},
		)
	}

	// +kubebuilder:docs-gen:collapse=rote conversion
	return nil
}

func convertToListener(src Emqx) v1beta3.ServiceTemplate {
	var dst v1beta3.ServiceTemplate
	srcListener := src.GetListener()

	dst.Name = src.GetName()
	dst.Namespace = src.GetNamespace()
	dst.Labels = srcListener.Labels
	dst.Annotations = srcListener.Annotations
	dst.Spec.Selector = src.GetLabels()
	dst.Spec.Type = srcListener.Type
	dst.Spec.LoadBalancerIP = srcListener.LoadBalancerIP
	dst.Spec.LoadBalancerSourceRanges = srcListener.LoadBalancerSourceRanges
	dst.Spec.ExternalIPs = srcListener.ExternalIPs

	if srcListener.Ports.API != 0 {
		dst.Spec.Ports = append(dst.Spec.Ports, corev1.ServicePort{
			Name:       "management-listener-http",
			Port:       srcListener.Ports.API,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.IntOrString{Type: 0, IntVal: srcListener.Ports.API},
			NodePort:   srcListener.NodePorts.API,
		})
	}
	if srcListener.Ports.Dashboard != 0 {
		dst.Spec.Ports = append(dst.Spec.Ports, corev1.ServicePort{
			Name:       "dashboard-listener-http",
			Port:       srcListener.Ports.Dashboard,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.IntOrString{Type: 0, IntVal: srcListener.Ports.Dashboard},
			NodePort:   srcListener.NodePorts.Dashboard,
		})
	}
	if srcListener.Ports.MQTT != 0 {
		dst.Spec.Ports = append(dst.Spec.Ports, corev1.ServicePort{
			Name:       "listener-tcp-external",
			Port:       srcListener.Ports.MQTT,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.IntOrString{Type: 0, IntVal: srcListener.Ports.MQTT},
			NodePort:   srcListener.NodePorts.MQTT,
		})
	}
	if srcListener.Ports.MQTTS != 0 {
		dst.Spec.Ports = append(dst.Spec.Ports, corev1.ServicePort{
			Name:       "listener-ssl-external",
			Port:       srcListener.Ports.MQTTS,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.IntOrString{Type: 0, IntVal: srcListener.Ports.MQTTS},
			NodePort:   srcListener.NodePorts.MQTTS,
		})
	}
	if srcListener.Ports.WS != 0 {
		dst.Spec.Ports = append(dst.Spec.Ports, corev1.ServicePort{
			Name:       "listener-ws-external",
			Port:       srcListener.Ports.WS,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.IntOrString{Type: 0, IntVal: srcListener.Ports.WS},
			NodePort:   srcListener.NodePorts.WS,
		})
	}
	if srcListener.Ports.WSS != 0 {
		dst.Spec.Ports = append(dst.Spec.Ports, corev1.ServicePort{
			Name:       "listener-wss-external",
			Port:       srcListener.Ports.WSS,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.IntOrString{Type: 0, IntVal: srcListener.Ports.WSS},
			NodePort:   srcListener.NodePorts.WSS,
		})
	}

	return dst
}

func convertFromListener(src v1beta3.Emqx) Listener {
	var dst Listener

	serviceTemplate := src.GetServiceTemplate()

	dst.Labels = serviceTemplate.Labels
	dst.Annotations = serviceTemplate.Annotations
	dst.Type = serviceTemplate.Spec.Type
	dst.LoadBalancerIP = serviceTemplate.Spec.LoadBalancerIP
	dst.LoadBalancerSourceRanges = serviceTemplate.Spec.LoadBalancerSourceRanges
	dst.ExternalIPs = serviceTemplate.Spec.ExternalIPs

	for _, port := range serviceTemplate.Spec.Ports {
		if port.Name == "management-listener-http" {
			dst.Ports.API = port.Port
			dst.NodePorts.API = port.NodePort
		}
		if port.Name == "dashboard-listener-http" {
			dst.Ports.Dashboard = port.Port
			dst.NodePorts.Dashboard = port.NodePort
		}
		if port.Name == "listener-tcp-external" {
			dst.Ports.MQTT = port.Port
			dst.NodePorts.MQTT = port.NodePort
		}
		if port.Name == "listener-ssl-external" {
			dst.Ports.MQTTS = port.Port
			dst.NodePorts.MQTTS = port.NodePort
		}
		if port.Name == "listener-ws-external" {
			dst.Ports.WS = port.Port
			dst.NodePorts.WS = port.NodePort
		}
		if port.Name == "listener-wss-external" {
			dst.Ports.WSS = port.Port
			dst.NodePorts.WSS = port.NodePort
		}
	}

	return dst
}

func conversionToEmqxConfig(evns []corev1.EnvVar) (v1beta3.EmqxConfig, []corev1.EnvVar) {
	config := make(v1beta3.EmqxConfig)

	otherEnv := []corev1.EnvVar{}

	emqxEnv, _ := regexp.Compile("^EMQX_")
	for _, env := range evns {
		if emqxEnv.MatchString(env.Name) {
			configName := strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(env.Name, "EMQX_"), "__", "."))
			config[configName] = env.Value
		} else {
			otherEnv = append(otherEnv, env)
		}
	}
	return config, otherEnv
}

func converFromEnvAndConfig(envs []corev1.EnvVar, emqxConfig v1beta3.EmqxConfig) (ret []corev1.EnvVar) {
	for k, v := range emqxConfig {
		key := fmt.Sprintf("EMQX_%s", strings.ToUpper(strings.ReplaceAll(k, ".", "__")))
		ret = append(ret, corev1.EnvVar{Name: key, Value: v})
	}

	ret = append(ret, envs...)
	tags := make(map[string]int)
	for i := len(ret) - 1; i >= 0; i-- {
		if _, ok := tags[ret[i].Name]; ok {
			ret = append(ret[:i], ret[i+1:]...)
		}
		tags[ret[i].Name] = i
	}

	return
}
