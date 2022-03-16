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
	"reflect"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
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

	// Listener
	dst.Spec.EmqxTemplate.Listener = convertToListener(src.Spec.EmqxTemplate.Listener)

	dst.Spec.Persistent = src.Spec.Storage

	dst.Spec.EmqxTemplate.ACL = src.Spec.EmqxTemplate.ACL
	dst.Spec.EmqxTemplate.Plugins = src.Spec.EmqxTemplate.Plugins
	dst.Spec.EmqxTemplate.Modules = src.Spec.EmqxTemplate.Modules

	// Spec
	dst.Spec.Replicas = src.Spec.Replicas
	dst.Spec.Image = src.Spec.Image
	dst.Spec.Resources = src.Spec.Resources
	dst.Spec.Affinity = src.Spec.Affinity
	dst.Spec.ToleRations = src.Spec.ToleRations
	dst.Spec.NodeSelector = src.Spec.NodeSelector
	dst.Spec.ImagePullPolicy = src.Spec.ImagePullPolicy
	dst.Spec.ExtraVolumes = src.Spec.ExtraVolumes
	dst.Spec.ExtraVolumeMounts = src.Spec.ExtraVolumeMounts
	dst.Spec.Env = src.Spec.Env
	dst.Spec.TelegrafTemplate = src.Spec.TelegrafTemplate

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
	dst.Spec.EmqxTemplate.Listener = convertFromListener(src.Spec.EmqxTemplate.Listener)

	if !reflect.ValueOf(src.Spec.Persistent).IsZero() {
		dst.Spec.Storage = src.Spec.Persistent
	}
	dst.Spec.EmqxTemplate.ACL = src.Spec.EmqxTemplate.ACL
	dst.Spec.EmqxTemplate.Plugins = src.Spec.EmqxTemplate.Plugins
	dst.Spec.EmqxTemplate.Modules = src.Spec.EmqxTemplate.Modules

	// Spec
	dst.Spec.Replicas = src.Spec.Replicas
	dst.Spec.Image = src.Spec.Image
	dst.Spec.Resources = src.Spec.Resources
	dst.Spec.Affinity = src.Spec.Affinity
	dst.Spec.ToleRations = src.Spec.ToleRations
	dst.Spec.NodeSelector = src.Spec.NodeSelector
	dst.Spec.ImagePullPolicy = src.Spec.ImagePullPolicy
	dst.Spec.ExtraVolumes = src.Spec.ExtraVolumes
	dst.Spec.ExtraVolumeMounts = src.Spec.ExtraVolumeMounts
	dst.Spec.Env = src.Spec.Env
	dst.Spec.TelegrafTemplate = src.Spec.TelegrafTemplate

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

func convertToListener(src Listener) v1beta3.Listener {
	var dst v1beta3.Listener

	dst.Labels = src.Labels
	dst.Annotations = src.Annotations
	dst.Type = src.Type
	dst.LoadBalancerIP = src.LoadBalancerIP
	dst.LoadBalancerSourceRanges = src.LoadBalancerSourceRanges
	dst.ExternalIPs = src.ExternalIPs

	dst.API.Port = src.Ports.API
	dst.Dashboard.Port = src.Ports.Dashboard
	dst.MQTT.Port = src.Ports.MQTT
	dst.MQTTS.Port = src.Ports.MQTTS
	dst.WS.Port = src.Ports.WS
	dst.WSS.Port = src.Ports.WSS

	dst.API.NodePort = src.NodePorts.API
	dst.Dashboard.NodePort = src.NodePorts.Dashboard
	dst.MQTT.NodePort = src.NodePorts.MQTT
	dst.MQTTS.NodePort = src.NodePorts.MQTTS
	dst.WS.NodePort = src.NodePorts.WS
	dst.WSS.NodePort = src.NodePorts.WSS

	dst.MQTTS.Cert.Data.CaCert = src.Certificate.MQTTS.Data.CaCert
	dst.MQTTS.Cert.Data.TLSCert = src.Certificate.MQTTS.Data.TLSCert
	dst.MQTTS.Cert.Data.TLSKey = src.Certificate.MQTTS.Data.TLSKey
	dst.MQTTS.Cert.StringData.CaCert = src.Certificate.MQTTS.StringData.CaCert
	dst.MQTTS.Cert.StringData.TLSCert = src.Certificate.MQTTS.StringData.TLSCert
	dst.MQTTS.Cert.StringData.TLSKey = src.Certificate.MQTTS.StringData.TLSKey

	dst.WSS.Cert.Data.CaCert = src.Certificate.WSS.Data.CaCert
	dst.WSS.Cert.Data.TLSCert = src.Certificate.WSS.Data.TLSCert
	dst.WSS.Cert.Data.TLSKey = src.Certificate.WSS.Data.TLSKey
	dst.WSS.Cert.StringData.CaCert = src.Certificate.WSS.StringData.CaCert
	dst.WSS.Cert.StringData.TLSCert = src.Certificate.WSS.StringData.TLSCert
	dst.WSS.Cert.StringData.TLSKey = src.Certificate.WSS.StringData.TLSKey

	return dst
}

func convertFromListener(src v1beta3.Listener) Listener {
	var dst Listener

	dst.Labels = src.Labels
	dst.Annotations = src.Annotations
	dst.Type = src.Type
	dst.LoadBalancerIP = src.LoadBalancerIP
	dst.LoadBalancerSourceRanges = src.LoadBalancerSourceRanges
	dst.ExternalIPs = src.ExternalIPs

	dst.Ports.API = src.API.Port
	dst.Ports.Dashboard = src.Dashboard.Port
	dst.Ports.MQTT = src.MQTT.Port
	dst.Ports.MQTTS = src.MQTTS.Port
	dst.Ports.WS = src.WS.Port
	dst.Ports.WSS = src.WSS.Port
	dst.NodePorts.API = src.API.NodePort
	dst.NodePorts.Dashboard = src.Dashboard.NodePort
	dst.NodePorts.MQTT = src.MQTT.NodePort
	dst.NodePorts.MQTTS = src.MQTTS.NodePort
	dst.NodePorts.WS = src.WS.NodePort
	dst.NodePorts.WSS = src.WSS.NodePort

	dst.Certificate.MQTTS.Data.CaCert = src.MQTTS.Cert.Data.CaCert
	dst.Certificate.MQTTS.Data.TLSCert = src.MQTTS.Cert.Data.TLSCert
	dst.Certificate.MQTTS.Data.TLSKey = src.MQTTS.Cert.Data.TLSKey
	dst.Certificate.MQTTS.StringData.CaCert = src.MQTTS.Cert.StringData.CaCert
	dst.Certificate.MQTTS.StringData.TLSCert = src.MQTTS.Cert.StringData.TLSCert
	dst.Certificate.MQTTS.StringData.TLSKey = src.MQTTS.Cert.StringData.TLSKey

	dst.Certificate.WSS.Data.CaCert = src.WSS.Cert.Data.CaCert
	dst.Certificate.WSS.Data.TLSCert = src.WSS.Cert.Data.TLSCert
	dst.Certificate.WSS.Data.TLSKey = src.WSS.Cert.Data.TLSKey
	dst.Certificate.WSS.StringData.CaCert = src.WSS.Cert.StringData.CaCert
	dst.Certificate.WSS.StringData.TLSCert = src.WSS.Cert.StringData.TLSCert
	dst.Certificate.WSS.StringData.TLSKey = src.WSS.Cert.StringData.TLSKey

	return dst
}
