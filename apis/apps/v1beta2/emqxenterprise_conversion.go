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
	"encoding/base64"
	"reflect"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts this version to the Hub version (v1).
func (src *EmqxEnterprise) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1beta3.EmqxEnterprise)

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

	// License
	dst.Spec.EmqxTemplate.License.StringData = src.Spec.EmqxTemplate.License

	// ServiceTemplate
	dst.Spec.EmqxTemplate.ServiceTemplate = convertToListener(src)

	// EmqxConfig
	dst.Spec.EmqxTemplate.EmqxConfig, dst.Spec.Env = conversionToEmqxConfig(src.Spec.Env)

	if !reflect.ValueOf(src.Spec.Storage).IsZero() {
		dst.Spec.Persistent = src.Spec.Storage
	}

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
func (dst *EmqxEnterprise) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1beta3.EmqxEnterprise)

	dst.ObjectMeta = src.ObjectMeta
	dst.Spec.Labels = src.Labels
	dst.Spec.Annotations = src.Annotations

	// License
	if len(src.Spec.EmqxTemplate.License.Data) != 0 {
		license, _ := base64.StdEncoding.DecodeString(string(src.Spec.EmqxTemplate.License.Data))
		dst.Spec.EmqxTemplate.License = string(license)
	}
	if src.Spec.EmqxTemplate.License.StringData != "" {
		dst.Spec.EmqxTemplate.License = src.Spec.EmqxTemplate.License.StringData
	}

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
	dst.Spec.Env = src.Spec.Env
	//dst.Spec.Env = src.Spec.EmqxTemplate.Env
	dst.Spec.Env = converFromEnvAndConfig(src.Spec.Env, src.Spec.EmqxTemplate.EmqxConfig)

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
