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
	dst.Spec.EmqxTemplate.ServiceTemplate = convertToListener(src.Spec.EmqxTemplate.Listener)

	dst.Spec.Persistent = src.Spec.Storage

	dst.Spec.EmqxTemplate.ACL = src.Spec.EmqxTemplate.ACL
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
	dst.Spec.EmqxTemplate.Env = src.Spec.Env

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
	dst.Spec.EmqxTemplate.ACL = src.Spec.EmqxTemplate.ACL
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

func convertToListener(src Listener) v1beta3.ServiceTemplate {
	var dst v1beta3.ServiceTemplate

	dst.Labels = src.Labels
	dst.Annotations = src.Annotations
	dst.Spec.Type = src.Type
	dst.Spec.LoadBalancerIP = src.LoadBalancerIP
	dst.Spec.LoadBalancerSourceRanges = src.LoadBalancerSourceRanges
	dst.Spec.ExternalIPs = src.ExternalIPs

	ports := []corev1.ServicePort{
		{
			Name:       "management-listener-http",
			Port:       src.Ports.API,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.IntOrString{Type: 0, IntVal: src.Ports.API},
			NodePort:   src.NodePorts.API,
		},
		{
			Name:       "dashboard-listener-http",
			Port:       src.Ports.Dashboard,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.IntOrString{Type: 0, IntVal: src.Ports.Dashboard},
			NodePort:   src.NodePorts.Dashboard,
		},
		{
			Name:       "listener-tcp-external",
			Port:       src.Ports.MQTT,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.IntOrString{Type: 0, IntVal: src.Ports.MQTT},
			NodePort:   src.NodePorts.MQTT,
		},
		{
			Name:       "listener-ssl-external",
			Port:       src.Ports.MQTTS,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.IntOrString{Type: 0, IntVal: src.Ports.MQTTS},
			NodePort:   src.NodePorts.MQTTS,
		},
		{
			Name:       "listener-ws-external",
			Port:       src.Ports.WS,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.IntOrString{Type: 0, IntVal: src.Ports.WS},
			NodePort:   src.NodePorts.WS,
		},
		{
			Name:       "listener-wss-external",
			Port:       src.Ports.WSS,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.IntOrString{Type: 0, IntVal: src.Ports.WSS},
			NodePort:   src.NodePorts.WSS,
		},
	}

	dst.Spec.Ports = append(dst.Spec.Ports, ports...)
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

	lookup := func(port string) corev1.ServicePort {
		for _, p := range serviceTemplate.Spec.Ports {
			if p.TargetPort.String() == port {
				return p
			}
		}
		return corev1.ServicePort{}
	}

	for _, env := range src.GetEnv() {
		if env.Name == "EMQX_MANAGEMENT_LISTENER__HTTP" {
			svcPort := lookup(env.Value)
			dst.Ports.API = svcPort.Port
			dst.NodePorts.API = svcPort.NodePort
		}
		if env.Name == "EMQX_DASHBOARD__LISTENER__HTTP" {
			svcPort := lookup(env.Value)
			dst.Ports.Dashboard = svcPort.Port
			dst.NodePorts.Dashboard = svcPort.NodePort
		}
		if env.Name == "EMQX_LISTENER__TCP__EXTERNAL" {
			svcPort := lookup(env.Value)
			dst.Ports.MQTT = svcPort.Port
			dst.NodePorts.MQTT = svcPort.NodePort
		}
		if env.Name == "EMQX_LISTENER__SSL__EXTERNAL" {
			svcPort := lookup(env.Value)
			dst.Ports.MQTTS = svcPort.Port
			dst.NodePorts.MQTTS = svcPort.NodePort
		}
		if env.Name == "EMQX_WS__LISTENER__TCP" {
			svcPort := lookup(env.Value)
			dst.Ports.WS = svcPort.Port
			dst.NodePorts.WS = svcPort.NodePort
		}
		if env.Name == "EMQX_WSS__LISTENER__TCP" {
			svcPort := lookup(env.Value)
			dst.Ports.WSS = svcPort.Port
			dst.NodePorts.WSS = svcPort.NodePort
		}

	}

	return dst
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
