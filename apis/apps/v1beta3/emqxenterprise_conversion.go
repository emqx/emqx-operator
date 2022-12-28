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

package v1beta3

import (
	"reflect"

	"github.com/emqx/emqx-operator/apis/apps/v1beta4"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts this version to the Hub version (v1).
func (src *EmqxEnterprise) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1beta4.EmqxEnterprise)
	dst.ObjectMeta = src.ObjectMeta
	// Replicas
	dst.Spec.Replicas = src.Spec.Replicas
	// ServiceTemplate
	if !reflect.ValueOf(src.Spec.EmqxTemplate.ServiceTemplate).IsZero() {
		dst.Spec.ServiceTemplate = v1beta4.ServiceTemplate(src.Spec.EmqxTemplate.ServiceTemplate)
	}
	// Persistent
	if !reflect.ValueOf(src.Spec.Persistent).IsZero() {
		names := Names{Object: src}
		dst.Spec.Persistent = &corev1.PersistentVolumeClaimTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name: names.Data(),
			},
			Spec: src.Spec.Persistent,
		}
	}
	// Template
	dst.Spec.Template.ObjectMeta.Labels = src.Labels
	dst.Spec.Template.ObjectMeta.Annotations = src.Annotations
	dst.Spec.Template.Spec.EmqxContainer.Name = "emqx"
	dst.Spec.Template.Spec.EmqxContainer.Image = src.Spec.EmqxTemplate.Image
	if !reflect.ValueOf(src.Spec.EmqxTemplate.License).IsZero() {
		dst.Spec.License = v1beta4.EmqxLicense(src.Spec.EmqxTemplate.License)
	}
	if src.Spec.EmqxTemplate.EmqxConfig != nil {
		dst.Spec.Template.Spec.EmqxContainer.EmqxConfig = src.Spec.EmqxTemplate.EmqxConfig
	}
	if len(src.Spec.EmqxTemplate.ACL) != 0 {
		dst.Spec.Template.Spec.EmqxContainer.EmqxACL = src.Spec.EmqxTemplate.ACL
	}
	if len(src.Spec.EmqxTemplate.ImagePullPolicy) != 0 {
		dst.Spec.Template.Spec.EmqxContainer.ImagePullPolicy = src.Spec.EmqxTemplate.ImagePullPolicy
	}
	if len(src.Spec.EmqxTemplate.Args) != 0 {
		dst.Spec.Template.Spec.EmqxContainer.Args = src.Spec.EmqxTemplate.Args
	}
	if !reflect.ValueOf(src.Spec.EmqxTemplate.Resources).IsZero() {
		dst.Spec.Template.Spec.EmqxContainer.Resources = src.Spec.EmqxTemplate.Resources
	}
	if src.Spec.EmqxTemplate.ReadinessProbe != nil {
		dst.Spec.Template.Spec.EmqxContainer.ReadinessProbe = src.Spec.EmqxTemplate.ReadinessProbe
	}
	if src.Spec.EmqxTemplate.LivenessProbe != nil {
		dst.Spec.Template.Spec.EmqxContainer.LivenessProbe = src.Spec.EmqxTemplate.LivenessProbe
	}
	if src.Spec.EmqxTemplate.StartupProbe != nil {
		dst.Spec.Template.Spec.EmqxContainer.StartupProbe = src.Spec.EmqxTemplate.StartupProbe
	}
	if len(src.Spec.EmqxTemplate.ExtraVolumeMounts) != 0 {
		dst.Spec.Template.Spec.EmqxContainer.VolumeMounts = src.Spec.EmqxTemplate.ExtraVolumeMounts
	}
	if len(src.Spec.EmqxTemplate.ExtraVolumes) != 0 {
		dst.Spec.Template.Spec.Volumes = src.Spec.EmqxTemplate.ExtraVolumes
	}
	if src.Spec.EmqxTemplate.SecurityContext != nil {
		dst.Spec.Template.Spec.PodSecurityContext = src.Spec.EmqxTemplate.SecurityContext
	}
	if len(src.Spec.InitContainers) != 0 {
		dst.Spec.Template.Spec.InitContainers = src.Spec.InitContainers
	}
	if len(src.Spec.ExtraContainers) != 0 {
		dst.Spec.Template.Spec.ExtraContainers = src.Spec.ExtraContainers
	}
	if len(src.Spec.ImagePullSecrets) != 0 {
		dst.Spec.Template.Spec.ImagePullSecrets = src.Spec.ImagePullSecrets
	}
	if len(src.Spec.Env) != 0 {
		dst.Spec.Template.Spec.EmqxContainer.Env = append(dst.Spec.Template.Spec.EmqxContainer.Env, src.Spec.Env...)
	}
	if len(src.Spec.ToleRations) != 0 {
		dst.Spec.Template.Spec.Tolerations = src.Spec.ToleRations
	}
	if len(src.Spec.NodeName) != 0 {
		dst.Spec.Template.Spec.NodeName = src.Spec.NodeName
	}
	if src.Spec.NodeSelector != nil {
		dst.Spec.Template.Spec.NodeSelector = src.Spec.NodeSelector
	}
	if src.Spec.Affinity != nil {
		dst.Spec.Template.Spec.Affinity = src.Spec.Affinity
	}

	// +kubebuilder:docs-gen:collapse=rote conversion
	return nil
}

// ConvertFrom converts from the Hub version (v1) to this version.
func (dst *EmqxEnterprise) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1beta4.EmqxEnterprise)
	dst.ObjectMeta = src.ObjectMeta
	//Replicas
	dst.Spec.Replicas = src.Spec.Replicas
	// Persistent
	if !reflect.ValueOf(src.Spec.Persistent).IsZero() {
		dst.Spec.Persistent = src.Spec.Persistent.Spec
	}
	// ServiceTemplate
	if !reflect.ValueOf(src.Spec.ServiceTemplate).IsZero() {
		dst.Spec.EmqxTemplate.ServiceTemplate = ServiceTemplate(src.Spec.ServiceTemplate)
	}
	// Template
	dst.Spec.EmqxTemplate.Image = src.Spec.Template.Spec.EmqxContainer.Image
	if !reflect.ValueOf(dst.Spec.EmqxTemplate.License).IsZero() {
		dst.Spec.EmqxTemplate.License = License(src.Spec.License)
	}
	if src.Spec.Template.Spec.EmqxContainer.EmqxConfig != nil {
		dst.Spec.EmqxTemplate.EmqxConfig = src.Spec.Template.Spec.EmqxContainer.EmqxConfig
	}
	if len(src.Spec.Template.Spec.EmqxContainer.EmqxACL) != 0 {
		dst.Spec.EmqxTemplate.ACL = src.Spec.Template.Spec.EmqxContainer.EmqxACL
	}
	if len(src.Spec.Template.Spec.EmqxContainer.ImagePullPolicy) != 0 {
		dst.Spec.EmqxTemplate.ImagePullPolicy = src.Spec.Template.Spec.EmqxContainer.ImagePullPolicy
	}
	if len(src.Spec.Template.Spec.EmqxContainer.Args) != 0 {
		dst.Spec.EmqxTemplate.Args = src.Spec.Template.Spec.EmqxContainer.Args
	}
	if !reflect.ValueOf(src.Spec.Template.Spec.EmqxContainer.Resources).IsZero() {
		dst.Spec.EmqxTemplate.Resources = src.Spec.Template.Spec.EmqxContainer.Resources
	}
	if src.Spec.Template.Spec.EmqxContainer.ReadinessProbe != nil {
		dst.Spec.EmqxTemplate.ReadinessProbe = src.Spec.Template.Spec.EmqxContainer.ReadinessProbe
	}
	if src.Spec.Template.Spec.EmqxContainer.LivenessProbe != nil {
		dst.Spec.EmqxTemplate.LivenessProbe = src.Spec.Template.Spec.EmqxContainer.LivenessProbe
	}
	if src.Spec.Template.Spec.EmqxContainer.StartupProbe != nil {
		dst.Spec.EmqxTemplate.StartupProbe = src.Spec.Template.Spec.EmqxContainer.StartupProbe
	}
	if len(src.Spec.Template.Spec.EmqxContainer.VolumeMounts) != 0 {
		dst.Spec.EmqxTemplate.ExtraVolumeMounts = src.Spec.Template.Spec.EmqxContainer.VolumeMounts
	}
	if len(src.Spec.Template.Spec.Volumes) != 0 {
		dst.Spec.EmqxTemplate.ExtraVolumes = src.Spec.Template.Spec.Volumes
	}
	if src.Spec.Template.Spec.PodSecurityContext != nil {
		dst.Spec.EmqxTemplate.SecurityContext = src.Spec.Template.Spec.PodSecurityContext
	}
	if len(src.Spec.Template.Spec.InitContainers) != 0 {
		dst.Spec.InitContainers = src.Spec.Template.Spec.InitContainers
	}
	if len(src.Spec.Template.Spec.ExtraContainers) != 0 {
		dst.Spec.ExtraContainers = src.Spec.Template.Spec.ExtraContainers
	}
	if len(src.Spec.Template.Spec.ImagePullSecrets) != 0 {
		dst.Spec.ImagePullSecrets = src.Spec.Template.Spec.ImagePullSecrets
	}
	if len(src.Spec.Template.Spec.EmqxContainer.Env) != 0 {
		dst.Spec.Env = src.Spec.Template.Spec.EmqxContainer.Env
	}
	if len(src.Spec.Template.Spec.Tolerations) != 0 {
		dst.Spec.ToleRations = src.Spec.Template.Spec.Tolerations
	}
	if len(src.Spec.Template.Spec.NodeName) != 0 {
		dst.Spec.NodeName = src.Spec.Template.Spec.NodeName
	}
	if src.Spec.Template.Spec.NodeSelector != nil {
		dst.Spec.NodeSelector = src.Spec.Template.Spec.NodeSelector
	}
	if src.Spec.Template.Spec.Affinity != nil {
		dst.Spec.Affinity = src.Spec.Template.Spec.Affinity
	}

	// +kubebuilder:docs-gen:collapse=rote conversion
	return nil
}
