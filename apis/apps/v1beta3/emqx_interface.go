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
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:object:generate=false
type EmqxSpec interface {
	Default()
	ValidateCreate() error
	ValidateUpdate(runtime.Object) error
	ValidateDelete() error

	GetReplicas() *int32
	SetReplicas(replicas *int32)

	GetPersistent() corev1.PersistentVolumeClaimSpec
	SetPersistent(persistent corev1.PersistentVolumeClaimSpec)

	GetNodeName() string
	SetNodeName(nodeName string)

	GetNodeSelector() map[string]string
	SetNodeSelector(nodeSelector map[string]string)

	GetAnnotations() map[string]string
	SetAnnotations(annotations map[string]string)

	GetAffinity() *corev1.Affinity
	SetAffinity(affinity *corev1.Affinity)

	GetToleRations() []corev1.Toleration
	SetToleRations(tolerations []corev1.Toleration)

	GetInitContainers() []corev1.Container
	SetInitContainers(containers []corev1.Container)

	GetExtraContainers() []corev1.Container
	SetExtraContainers(containers []corev1.Container)

	GetImage() string
	SetImage(image string)

	GetImagePullPolicy() corev1.PullPolicy
	SetImagePullPolicy(pullPolicy corev1.PullPolicy)

	GetImagePullSecrets() []corev1.LocalObjectReference
	SetImagePullSecrets([]corev1.LocalObjectReference)

	GetSecurityContext() *corev1.PodSecurityContext
	SetSecurityContext(securityContext *corev1.PodSecurityContext)

	GetResource() corev1.ResourceRequirements
	SetResource(resource corev1.ResourceRequirements)

	GetExtraVolumes() []corev1.Volume
	GetExtraVolumeMounts() []corev1.VolumeMount

	GetReadinessProbe() *corev1.Probe
	SetReadinessProbe(probe *corev1.Probe)

	GetLivenessProbe() *corev1.Probe
	SetLivenessProbe(probe *corev1.Probe)

	GetStartupProbe() *corev1.Probe
	SetStartupProbe(probe *corev1.Probe)

	GetEmqxConfig() EmqxConfig
	SetEmqxConfig(config EmqxConfig)

	GetEnv() []corev1.EnvVar
	SetEnv(env []corev1.EnvVar)

	GetArgs() []string
	SetArgs(args []string)

	GetACL() []string
	SetACL(acl []string)

	GetServiceTemplate() ServiceTemplate
	SetServiceTemplate(ServiceTemplate)

	GetUsername() string
	SetUsername(username string)

	GetPassword() string
	SetPassword(password string)

	GetStatus() Status
	SetStatus(status Status)

	GetRegistry() string
	SetRegistry(registry string)
}

// +kubebuilder:object:generate=false
type Emqx interface {
	v1.Type
	v1.Object

	EmqxSpec
	EmqxStatus

	client.Object
}
