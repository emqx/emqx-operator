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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

//+kubebuilder:object:root=true
//+kubebuilder:resource:shortName=emqx-ee
//+kubebuilder:subresource:status
//+kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas
// EmqxEnterprise is the Schema for the emqxenterprises API
type EmqxEnterprise struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EmqxBrokerSpec `json:"spec,omitempty"`
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

func (emqx *EmqxEnterprise) GetAclConf() string        { return emqx.Spec.AclConf }
func (emqx *EmqxEnterprise) SetAclConf(aclConf string) { emqx.Spec.AclConf = aclConf }

func (emqx *EmqxEnterprise) GetLoadedPluginConf() string { return emqx.Spec.LoadedPluginConf }
func (emqx *EmqxEnterprise) SetLoadedPluginConf(loadedPluginConf string) {
	emqx.Spec.LoadedPluginConf = loadedPluginConf
}

func (emqx *EmqxEnterprise) GetLoadedModulesConf() string { return emqx.Spec.LoadedModulesConf }
func (emqx *EmqxEnterprise) SetLoadedModulesConf(loadedModulesConf string) {
	emqx.Spec.LoadedModulesConf = loadedModulesConf
}

func (emqx *EmqxEnterprise) GetSecretName() string {
	return fmt.Sprintf("%s-%s", emqx.Name, "secret")
}

func (emqx *EmqxEnterprise) GetHeadlessServiceName() string {
	return fmt.Sprintf("%s-%s", emqx.Name, "headless")
}

func (emqx *EmqxEnterprise) GetAcl() map[string]string {
	var config string
	if emqx.Spec.AclConf != "" {
		config = emqx.Spec.AclConf
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
	var config string
	if emqx.Spec.LoadedPluginConf != "" {
		config = emqx.Spec.LoadedPluginConf
	} else {
		config = `
{emqx_management, true}.
{emqx_recon, true}.
{emqx_retainer, true}.
{emqx_dashboard, true}.
{emqx_telemetry, true}.
{emqx_rule_engine, true}.
`
	}
	return map[string]string{
		"name":      fmt.Sprintf("%s-%s", emqx.Name, "loaded-plugins"),
		"mountPath": "/opt/emqx/data/loaded_plugins",
		"subPath":   "loaded_plugins",
		"conf":      config,
	}
}

func (emqx *EmqxEnterprise) GetLoadedModules() map[string]string {
	var config string
	if emqx.Spec.LoadedModulesConf != "" {
		config = emqx.Spec.LoadedModulesConf
	} else {
		config = `
[{
	"name": "internal_acl",
	"enable": true,
	"configs": {"acl_rule_file": "etc/acl.conf"}
},
{
	"name": "presence",
	"enable": true,
	"configs": {"qos": 0}
},
{
	"name": "recon",
	"enable": true,
	"configs": {}
},
{
	"name": "retainer",
	"enable": true,
	"configs": {
		"expiry_interval": 0,
		"max_payload_size": "1MB",
		"max_retained_messages": 0,
		"storage_type": "ram"
	}
}]
`
	}
	return map[string]string{
		"name":      fmt.Sprintf("%s-%s", emqx.Name, "loaded-modules"),
		"mountPath": "/opt/emqx/data/loaded_modules",
		"subPath":   "loaded_modules",
		"conf":      config,
	}
}

func (emqx *EmqxEnterprise) GetDataVolumeName() string {
	return fmt.Sprintf("%s-%s", emqx.Name, "data")
}

func (emqx *EmqxEnterprise) GetLogVolumeName() string {
	return fmt.Sprintf("%s-%s", emqx.Name, "log")
}
