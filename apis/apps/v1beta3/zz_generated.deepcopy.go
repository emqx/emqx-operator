//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1beta3

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ACL) DeepCopyInto(out *ACL) {
	*out = *in
	in.Topics.DeepCopyInto(&out.Topics)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ACL.
func (in *ACL) DeepCopy() *ACL {
	if in == nil {
		return nil
	}
	out := new(ACL)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Cert) DeepCopyInto(out *Cert) {
	*out = *in
	in.Data.DeepCopyInto(&out.Data)
	out.StringData = in.StringData
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Cert.
func (in *Cert) DeepCopy() *Cert {
	if in == nil {
		return nil
	}
	out := new(Cert)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CertConf) DeepCopyInto(out *CertConf) {
	*out = *in
	in.Data.DeepCopyInto(&out.Data)
	out.StringData = in.StringData
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CertConf.
func (in *CertConf) DeepCopy() *CertConf {
	if in == nil {
		return nil
	}
	out := new(CertConf)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CertData) DeepCopyInto(out *CertData) {
	*out = *in
	if in.CaCert != nil {
		in, out := &in.CaCert, &out.CaCert
		*out = make([]byte, len(*in))
		copy(*out, *in)
	}
	if in.TLSCert != nil {
		in, out := &in.TLSCert, &out.TLSCert
		*out = make([]byte, len(*in))
		copy(*out, *in)
	}
	if in.TLSKey != nil {
		in, out := &in.TLSKey, &out.TLSKey
		*out = make([]byte, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CertData.
func (in *CertData) DeepCopy() *CertData {
	if in == nil {
		return nil
	}
	out := new(CertData)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CertStringData) DeepCopyInto(out *CertStringData) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CertStringData.
func (in *CertStringData) DeepCopy() *CertStringData {
	if in == nil {
		return nil
	}
	out := new(CertStringData)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Condition) DeepCopyInto(out *Condition) {
	*out = *in
	in.LastUpdateAt.DeepCopyInto(&out.LastUpdateAt)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Condition.
func (in *Condition) DeepCopy() *Condition {
	if in == nil {
		return nil
	}
	out := new(Condition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EmqxBroker) DeepCopyInto(out *EmqxBroker) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EmqxBroker.
func (in *EmqxBroker) DeepCopy() *EmqxBroker {
	if in == nil {
		return nil
	}
	out := new(EmqxBroker)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EmqxBroker) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EmqxBrokerList) DeepCopyInto(out *EmqxBrokerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]EmqxBroker, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EmqxBrokerList.
func (in *EmqxBrokerList) DeepCopy() *EmqxBrokerList {
	if in == nil {
		return nil
	}
	out := new(EmqxBrokerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EmqxBrokerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EmqxBrokerModule) DeepCopyInto(out *EmqxBrokerModule) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EmqxBrokerModule.
func (in *EmqxBrokerModule) DeepCopy() *EmqxBrokerModule {
	if in == nil {
		return nil
	}
	out := new(EmqxBrokerModule)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EmqxBrokerSpec) DeepCopyInto(out *EmqxBrokerSpec) {
	*out = *in
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	if in.ImagePullSecrets != nil {
		in, out := &in.ImagePullSecrets, &out.ImagePullSecrets
		*out = make([]v1.LocalObjectReference, len(*in))
		copy(*out, *in)
	}
	in.Persistent.DeepCopyInto(&out.Persistent)
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(v1.Affinity)
		(*in).DeepCopyInto(*out)
	}
	if in.ToleRations != nil {
		in, out := &in.ToleRations, &out.ToleRations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.InitContainers != nil {
		in, out := &in.InitContainers, &out.InitContainers
		*out = make([]v1.Container, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.EmqxTemplate.DeepCopyInto(&out.EmqxTemplate)
	if in.TelegrafTemplate != nil {
		in, out := &in.TelegrafTemplate, &out.TelegrafTemplate
		*out = new(TelegrafTemplate)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EmqxBrokerSpec.
func (in *EmqxBrokerSpec) DeepCopy() *EmqxBrokerSpec {
	if in == nil {
		return nil
	}
	out := new(EmqxBrokerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EmqxBrokerTemplate) DeepCopyInto(out *EmqxBrokerTemplate) {
	*out = *in
	if in.ExtraVolumes != nil {
		in, out := &in.ExtraVolumes, &out.ExtraVolumes
		*out = make([]v1.Volume, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.ExtraVolumeMounts != nil {
		in, out := &in.ExtraVolumeMounts, &out.ExtraVolumeMounts
		*out = make([]v1.VolumeMount, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]v1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.SecurityContext != nil {
		in, out := &in.SecurityContext, &out.SecurityContext
		*out = new(v1.PodSecurityContext)
		(*in).DeepCopyInto(*out)
	}
	in.Resources.DeepCopyInto(&out.Resources)
	in.Listener.DeepCopyInto(&out.Listener)
	if in.ACL != nil {
		in, out := &in.ACL, &out.ACL
		*out = make([]ACL, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Plugins != nil {
		in, out := &in.Plugins, &out.Plugins
		*out = make([]Plugin, len(*in))
		copy(*out, *in)
	}
	if in.Modules != nil {
		in, out := &in.Modules, &out.Modules
		*out = make([]EmqxBrokerModule, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EmqxBrokerTemplate.
func (in *EmqxBrokerTemplate) DeepCopy() *EmqxBrokerTemplate {
	if in == nil {
		return nil
	}
	out := new(EmqxBrokerTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EmqxEnterprise) DeepCopyInto(out *EmqxEnterprise) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EmqxEnterprise.
func (in *EmqxEnterprise) DeepCopy() *EmqxEnterprise {
	if in == nil {
		return nil
	}
	out := new(EmqxEnterprise)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EmqxEnterprise) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EmqxEnterpriseList) DeepCopyInto(out *EmqxEnterpriseList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]EmqxEnterprise, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EmqxEnterpriseList.
func (in *EmqxEnterpriseList) DeepCopy() *EmqxEnterpriseList {
	if in == nil {
		return nil
	}
	out := new(EmqxEnterpriseList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EmqxEnterpriseList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EmqxEnterpriseModule) DeepCopyInto(out *EmqxEnterpriseModule) {
	*out = *in
	in.Configs.DeepCopyInto(&out.Configs)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EmqxEnterpriseModule.
func (in *EmqxEnterpriseModule) DeepCopy() *EmqxEnterpriseModule {
	if in == nil {
		return nil
	}
	out := new(EmqxEnterpriseModule)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EmqxEnterpriseSpec) DeepCopyInto(out *EmqxEnterpriseSpec) {
	*out = *in
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	if in.ImagePullSecrets != nil {
		in, out := &in.ImagePullSecrets, &out.ImagePullSecrets
		*out = make([]v1.LocalObjectReference, len(*in))
		copy(*out, *in)
	}
	in.Persistent.DeepCopyInto(&out.Persistent)
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(v1.Affinity)
		(*in).DeepCopyInto(*out)
	}
	if in.ToleRations != nil {
		in, out := &in.ToleRations, &out.ToleRations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.InitContainers != nil {
		in, out := &in.InitContainers, &out.InitContainers
		*out = make([]v1.Container, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.EmqxTemplate.DeepCopyInto(&out.EmqxTemplate)
	if in.TelegrafTemplate != nil {
		in, out := &in.TelegrafTemplate, &out.TelegrafTemplate
		*out = new(TelegrafTemplate)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EmqxEnterpriseSpec.
func (in *EmqxEnterpriseSpec) DeepCopy() *EmqxEnterpriseSpec {
	if in == nil {
		return nil
	}
	out := new(EmqxEnterpriseSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EmqxEnterpriseTemplate) DeepCopyInto(out *EmqxEnterpriseTemplate) {
	*out = *in
	if in.ExtraVolumes != nil {
		in, out := &in.ExtraVolumes, &out.ExtraVolumes
		*out = make([]v1.Volume, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.ExtraVolumeMounts != nil {
		in, out := &in.ExtraVolumeMounts, &out.ExtraVolumeMounts
		*out = make([]v1.VolumeMount, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]v1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.SecurityContext != nil {
		in, out := &in.SecurityContext, &out.SecurityContext
		*out = new(v1.PodSecurityContext)
		(*in).DeepCopyInto(*out)
	}
	in.Resources.DeepCopyInto(&out.Resources)
	in.Listener.DeepCopyInto(&out.Listener)
	if in.ACL != nil {
		in, out := &in.ACL, &out.ACL
		*out = make([]ACL, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Plugins != nil {
		in, out := &in.Plugins, &out.Plugins
		*out = make([]Plugin, len(*in))
		copy(*out, *in)
	}
	if in.Modules != nil {
		in, out := &in.Modules, &out.Modules
		*out = make([]EmqxEnterpriseModule, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.License.DeepCopyInto(&out.License)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EmqxEnterpriseTemplate.
func (in *EmqxEnterpriseTemplate) DeepCopy() *EmqxEnterpriseTemplate {
	if in == nil {
		return nil
	}
	out := new(EmqxEnterpriseTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EnvList) DeepCopyInto(out *EnvList) {
	*out = *in
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]v1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EnvList.
func (in *EnvList) DeepCopy() *EnvList {
	if in == nil {
		return nil
	}
	out := new(EnvList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *License) DeepCopyInto(out *License) {
	*out = *in
	if in.Data != nil {
		in, out := &in.Data, &out.Data
		*out = make([]byte, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new License.
func (in *License) DeepCopy() *License {
	if in == nil {
		return nil
	}
	out := new(License)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Listener) DeepCopyInto(out *Listener) {
	*out = *in
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.LoadBalancerSourceRanges != nil {
		in, out := &in.LoadBalancerSourceRanges, &out.LoadBalancerSourceRanges
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ExternalIPs != nil {
		in, out := &in.ExternalIPs, &out.ExternalIPs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	in.API.DeepCopyInto(&out.API)
	in.Dashboard.DeepCopyInto(&out.Dashboard)
	in.MQTT.DeepCopyInto(&out.MQTT)
	in.MQTTS.DeepCopyInto(&out.MQTTS)
	in.WS.DeepCopyInto(&out.WS)
	in.WSS.DeepCopyInto(&out.WSS)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Listener.
func (in *Listener) DeepCopy() *Listener {
	if in == nil {
		return nil
	}
	out := new(Listener)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ListenerPort) DeepCopyInto(out *ListenerPort) {
	*out = *in
	in.Cert.DeepCopyInto(&out.Cert)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ListenerPort.
func (in *ListenerPort) DeepCopy() *ListenerPort {
	if in == nil {
		return nil
	}
	out := new(ListenerPort)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Plugin) DeepCopyInto(out *Plugin) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Plugin.
func (in *Plugin) DeepCopy() *Plugin {
	if in == nil {
		return nil
	}
	out := new(Plugin)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Status) DeepCopyInto(out *Status) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Status.
func (in *Status) DeepCopy() *Status {
	if in == nil {
		return nil
	}
	out := new(Status)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TelegrafTemplate) DeepCopyInto(out *TelegrafTemplate) {
	*out = *in
	if in.Conf != nil {
		in, out := &in.Conf, &out.Conf
		*out = new(string)
		**out = **in
	}
	in.Resources.DeepCopyInto(&out.Resources)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TelegrafTemplate.
func (in *TelegrafTemplate) DeepCopy() *TelegrafTemplate {
	if in == nil {
		return nil
	}
	out := new(TelegrafTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Topics) DeepCopyInto(out *Topics) {
	*out = *in
	if in.Filter != nil {
		in, out := &in.Filter, &out.Filter
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Equal != nil {
		in, out := &in.Equal, &out.Equal
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Topics.
func (in *Topics) DeepCopy() *Topics {
	if in == nil {
		return nil
	}
	out := new(Topics)
	in.DeepCopyInto(out)
	return out
}
