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

package v1alpha2

import (
	"github.com/emqx/emqx-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//+kubebuilder:object:generate=false
type EmqxSpec interface {
	GetReplicas() *int32
	SetReplicas(replcas int32)

	GetImage() string
	SetImage(image string)

	GetServiceAccountName() string
	SetServiceAccountName(serviceAccountName string)

	GetResource() corev1.ResourceRequirements
	SetResource(resource corev1.ResourceRequirements)

	GetLicense() string
	SetLicense(license string)

	GetStorage() *Storage
	SetStorage(stroage *Storage)

	GetLabels() map[string]string
	SetLabels(labels map[string]string)

	GetListener() util.Listener

	GetAffinity() *corev1.Affinity
	SetAffinity(affinity *corev1.Affinity)

	GetToleRations() []corev1.Toleration
	SetToleRations(tolerations []corev1.Toleration)

	GetNodeSelector() map[string]string
	SetNodeSelector(nodeSelector map[string]string)

	GetImagePullPolicy() corev1.PullPolicy
	SetImagePullPolicy(pullPolicy corev1.PullPolicy)

	GetEnv() []corev1.EnvVar

	GetAcl() map[string]string
	GetLoadedPlugins() map[string]string
	GetLoadedModules() map[string]string

	GetSecretName() string
	GetDataVolumeName() string
	GetLogVolumeName() string
}

// +kubebuilder:object:generate=false
type Emqx interface {
	v1.Type
	v1.Object

	EmqxSpec
	EmqxStatus

	client.Object

	// validate
	Validate() error
}
