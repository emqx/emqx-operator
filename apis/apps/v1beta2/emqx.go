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
	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//+kubebuilder:object:generate=false
type EmqxSpec interface {
	GetReplicas() *int32
	SetReplicas(replicas *int32)

	GetImage() string
	SetImage(image string)

	GetImagePullPolicy() corev1.PullPolicy
	SetImagePullPolicy(pullPolicy corev1.PullPolicy)

	GetImagePullSecrets() []corev1.LocalObjectReference
	SetImagePullSecrets([]corev1.LocalObjectReference)

	GetServiceAccountName() string
	SetServiceAccountName(serviceAccountName string)

	GetResource() corev1.ResourceRequirements
	SetResource(resource corev1.ResourceRequirements)

	GetStorage() corev1.PersistentVolumeClaimSpec
	SetStorage(storage corev1.PersistentVolumeClaimSpec)

	GetNodeName() string
	SetNodeName(nodeName string)

	GetNodeSelector() map[string]string
	SetNodeSelector(nodeSelector map[string]string)

	GetAnnotations() map[string]string
	SetAnnotations(annotations map[string]string)

	GetListener() Listener
	SetListener(Listener)

	GetAffinity() *corev1.Affinity
	SetAffinity(affinity *corev1.Affinity)

	GetToleRations() []corev1.Toleration
	SetToleRations(tolerations []corev1.Toleration)

	GetExtraVolumes() []corev1.Volume
	GetExtraVolumeMounts() []corev1.VolumeMount

	GetACL() []v1beta3.ACL
	SetACL(acl []v1beta3.ACL)

	GetEnv() []corev1.EnvVar
	SetEnv(env []corev1.EnvVar)

	GetPlugins() []Plugin
	SetPlugins(plugins []Plugin)

	GetSecurityContext() *corev1.PodSecurityContext
	SetSecurityContext(securityContext *corev1.PodSecurityContext)

	GetTelegrafTemplate() *TelegrafTemplate
	SetTelegrafTemplate(telegraftedTemplate *TelegrafTemplate)
}

// +kubebuilder:object:generate=false
type Emqx interface {
	v1.Type
	v1.Object

	EmqxSpec
	EmqxStatus

	client.Object
}
