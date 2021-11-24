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
	"fmt"

	"github.com/emqx/emqx-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EmqxEnterpriseSpec defines the desired state of EmqxEnterprise
type EmqxEnterpriseSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The fields of Broker.
	//The replicas of emqx broker
	Replicas *int32 `json:"replicas,omitempty"`

	//+kubebuilder:validation:Required
	Image string `json:"image,omitempty"`

	//+kubebuilder:validation:Required
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// The service account name which is being binded with the service
	// account of the crd instance.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	License   string                      `json:"license,omitempty"`

	Storage *Storage `json:"storage,omitempty"`

	// The labels configure must be specified.
	//+kubebuilder:validation:Required
	Labels map[string]string `json:"labels,omitempty"`

	Affinity        *corev1.Affinity    `json:"affinity,omitempty"`
	ToleRations     []corev1.Toleration `json:"toleRations,omitempty"`
	NodeSelector    map[string]string   `json:"nodeSelector,omitempty"`
	ImagePullPolicy corev1.PullPolicy   `json:"imagePullPolicy,omitempty"`

	Env []corev1.EnvVar `json:"env,omitempty"`

	ACL string `json:"acl,omitempty"`

	Plugins []util.Plugin `json:"plugins,omitempty"`

	Modules []util.EmqxEnterpriseModules `json:"modules,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:shortName=emqx-ee
//+kubebuilder:subresource:status
//+kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas
// EmqxEnterprise is the Schema for the emqxenterprises API
type EmqxEnterprise struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EmqxEnterpriseSpec `json:"spec,omitempty"`
	Status `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
// EmqxEnterpriseList contains a list of EmqxEnterprise
type EmqxEnterpriseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EmqxEnterprise `json:"items"`
}

func (emqx *EmqxEnterprise) String() string {
	return fmt.Sprintf("EmqxEnterprise instance [%s],Image [%s]",
		emqx.ObjectMeta.Name,
		emqx.Spec.Image,
	)
}

func init() {
	SchemeBuilder.Register(&EmqxEnterprise{}, &EmqxEnterpriseList{})
}

func (emqx *EmqxEnterprise) GetAPIVersion() string        { return emqx.APIVersion }
func (emqx *EmqxEnterprise) SetAPIVersion(version string) { emqx.APIVersion = version }
func (emqx *EmqxEnterprise) GetKind() string              { return emqx.Kind }
func (emqx *EmqxEnterprise) SetKind(kind string)          { emqx.Kind = kind }

func (emqx *EmqxEnterprise) GetReplicas() *int32        { return emqx.Spec.Replicas }
func (emqx *EmqxEnterprise) SetReplicas(replicas int32) { emqx.Spec.Replicas = &replicas }

func (emqx *EmqxEnterprise) GetImage() string      { return emqx.Spec.Image }
func (emqx *EmqxEnterprise) SetImage(image string) { emqx.Spec.Image = image }

func (emqx *EmqxEnterprise) GetServiceAccountName() string { return emqx.Spec.ServiceAccountName }
func (emqx *EmqxEnterprise) SetServiceAccountName(serviceAccountName string) {
	emqx.Spec.ServiceAccountName = serviceAccountName
}

func (emqx *EmqxEnterprise) GetResource() corev1.ResourceRequirements { return emqx.Spec.Resources }
func (emqx *EmqxEnterprise) SetResource(resource corev1.ResourceRequirements) {
	emqx.Spec.Resources = resource
}

func (emqx *EmqxEnterprise) GetLicense() string        { return emqx.Spec.License }
func (emqx *EmqxEnterprise) SetLicense(license string) { emqx.Spec.License = license }

func (emqx *EmqxEnterprise) GetStorage() *Storage        { return emqx.Spec.Storage }
func (emqx *EmqxEnterprise) SetStorage(stroage *Storage) { emqx.Spec.Storage = stroage }

func (emqx *EmqxEnterprise) GetLabels() map[string]string       { return emqx.Spec.Labels }
func (emqx *EmqxEnterprise) SetLabels(labels map[string]string) { emqx.Spec.Labels = labels }

func (emqx *EmqxEnterprise) GetListener() *Listener        { return emqx.Spec.Listener }
func (emqx *EmqxEnterprise) SetListener(listener Listener) { emqx.Spec.Listener = &listener }

func (emqx *EmqxEnterprise) GetAffinity() *corev1.Affinity         { return emqx.Spec.Affinity }
func (emqx *EmqxEnterprise) SetAffinity(affinity *corev1.Affinity) { emqx.Spec.Affinity = affinity }

func (emqx *EmqxEnterprise) GetToleRations() []corev1.Toleration { return emqx.Spec.ToleRations }
func (emqx *EmqxEnterprise) SetToleRations(tolerations []corev1.Toleration) {
	emqx.Spec.ToleRations = tolerations
}

func (emqx *EmqxEnterprise) GetNodeSelector() map[string]string { return emqx.Spec.NodeSelector }
func (emqx *EmqxEnterprise) SetNodeSelector(nodeSelector map[string]string) {
	emqx.Spec.NodeSelector = nodeSelector
}

func (emqx *EmqxEnterprise) GetImagePullPolicy() corev1.PullPolicy { return emqx.Spec.ImagePullPolicy }
func (emqx *EmqxEnterprise) SetImagePullPolicy(pullPolicy corev1.PullPolicy) {
	emqx.Spec.ImagePullPolicy = pullPolicy
}

func (emqx *EmqxEnterprise) GetEnv() []corev1.EnvVar    { return emqx.Spec.Env }
func (emqx *EmqxEnterprise) SetEnv(env []corev1.EnvVar) { emqx.Spec.Env = env }

func (emqx *EmqxEnterprise) GetSecretName() string {
	return fmt.Sprintf("%s-%s", emqx.Name, "secret")
}

func (emqx *EmqxEnterprise) GetHeadlessServiceName() string {
	return fmt.Sprintf("%s-%s", emqx.Name, "headless")
}

func (emqx *EmqxEnterprise) GetAcl() map[string]string {
	var config string
	if emqx.Spec.ACL != "" {
		config = emqx.Spec.ACL
	} else {
		config = `
{allow, {user, "dashboard"}, subscribe, ["$SYS/#"]}.
{allow, {ipaddr, "127.0.0.1"}, pubsub, ["$SYS/#", "#"]}.
{deny, all, subscribe, ["$SYS/#", {eq, "#"}]}.
{allow, all}.
`
	}
	return map[string]string{
		"name":      emqx.Name,
		"mountPath": "/opt/emqx/etc/acl.conf",
		"subPath":   "acl.conf",
		"conf":      config,
	}

}

func (emqx *EmqxEnterprise) GetLoadedPlugins() map[string]string {
	return map[string]string{
		"name":      fmt.Sprintf("%s-%s", emqx.Name, "loaded-plugins"),
		"mountPath": "/opt/emqx/data/loaded_plugins",
		"subPath":   "loaded_plugins",
		"conf":      util.GenLoadedPlugins(emqx.Spec.Plugins),
	}
}

func (emqx *EmqxEnterprise) GetLoadedModules() map[string]string {
	return map[string]string{
		"name":      fmt.Sprintf("%s-%s", emqx.Name, "loaded-modules"),
		"mountPath": "/opt/emqx/data/loaded_modules",
		"subPath":   "loaded_modules",
		"conf":      util.GenEmqxEnterpriseLoadedModules(emqx.Spec.Modules),
	}
}

func (emqx *EmqxEnterprise) GetDataVolumeName() string {
	return fmt.Sprintf("%s-%s", emqx.Name, "data")
}

func (emqx *EmqxEnterprise) GetLogVolumeName() string {
	return fmt.Sprintf("%s-%s", emqx.Name, "log")
}
