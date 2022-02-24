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

	v1beta1 "github.com/emqx/emqx-operator/apis/apps/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts this version to the Hub version (v1).
func (src *EmqxBroker) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1beta1.EmqxBroker)

	if !reflect.ValueOf(src.Spec.Storage).IsZero() {
		dst.Spec.Storage = &v1beta1.Storage{
			VolumeClaimTemplate: v1beta1.EmbeddedPersistentVolumeClaim{
				Spec: src.Spec.Storage,
			},
		}
	}
	dst.Spec.Listener = src.Spec.EmqxTemplate.Listener
	dst.Spec.ACL = src.Spec.EmqxTemplate.ACL
	dst.Spec.Plugins = src.Spec.EmqxTemplate.Plugins
	dst.Spec.Modules = src.Spec.EmqxTemplate.Modules

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta
	// Spec
	dst.Spec.Replicas = src.Spec.Replicas
	dst.Spec.Image = src.Spec.Image
	dst.Spec.ServiceAccountName = src.Spec.ServiceAccountName
	dst.Spec.Resources = src.Spec.Resources
	dst.Spec.Labels = src.Spec.Labels
	dst.Spec.Annotations = src.Spec.Annotations
	dst.Spec.Affinity = src.Spec.Affinity
	dst.Spec.ToleRations = src.Spec.ToleRations
	dst.Spec.NodeSelector = src.Spec.NodeSelector
	dst.Spec.ImagePullPolicy = src.Spec.ImagePullPolicy
	dst.Spec.ExtraVolumes = src.Spec.ExtraVolumes
	dst.Spec.ExtraVolumeMounts = src.Spec.ExtraVolumeMounts
	dst.Spec.Env = src.Spec.Env
	dst.Spec.TelegrafTemplate = src.Spec.TelegrafTemplate

	// Status
	dst.Status.Conditions = src.Status.Conditions

	// +kubebuilder:docs-gen:collapse=rote conversion
	return nil
}

// ConvertFrom converts from the Hub version (v1) to this version.
func (dst *EmqxBroker) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1beta1.EmqxBroker)

	if !reflect.ValueOf(src.Spec.Storage).IsZero() {
		dst.Spec.Storage = src.Spec.Storage.VolumeClaimTemplate.Spec
	}
	dst.Spec.EmqxTemplate.Listener = src.Spec.Listener
	dst.Spec.EmqxTemplate.ACL = src.Spec.ACL
	dst.Spec.EmqxTemplate.Plugins = src.Spec.Plugins
	dst.Spec.EmqxTemplate.Modules = src.Spec.Modules

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta
	// Spec
	dst.Spec.Replicas = src.Spec.Replicas
	dst.Spec.Image = src.Spec.Image
	dst.Spec.ServiceAccountName = src.Spec.ServiceAccountName
	dst.Spec.Resources = src.Spec.Resources
	dst.Spec.Labels = src.Spec.Labels
	dst.Spec.Annotations = src.Spec.Annotations
	dst.Spec.Affinity = src.Spec.Affinity
	dst.Spec.ToleRations = src.Spec.ToleRations
	dst.Spec.NodeSelector = src.Spec.NodeSelector
	dst.Spec.ImagePullPolicy = src.Spec.ImagePullPolicy
	dst.Spec.ExtraVolumes = src.Spec.ExtraVolumes
	dst.Spec.ExtraVolumeMounts = src.Spec.ExtraVolumeMounts
	dst.Spec.Env = src.Spec.Env
	dst.Spec.TelegrafTemplate = src.Spec.TelegrafTemplate

	// Status
	dst.Status.Conditions = src.Status.Conditions

	// +kubebuilder:docs-gen:collapse=rote conversion
	return nil
}
