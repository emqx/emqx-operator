//go:build !ignore_autogenerated

/*
Copyright 2025.

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

package v2beta1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BootstrapAPIKey) DeepCopyInto(out *BootstrapAPIKey) {
	*out = *in
	if in.SecretRef != nil {
		in, out := &in.SecretRef, &out.SecretRef
		*out = new(SecretRef)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BootstrapAPIKey.
func (in *BootstrapAPIKey) DeepCopy() *BootstrapAPIKey {
	if in == nil {
		return nil
	}
	out := new(BootstrapAPIKey)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Config) DeepCopyInto(out *Config) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Config.
func (in *Config) DeepCopy() *Config {
	if in == nil {
		return nil
	}
	out := new(Config)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DSDBReplicationStatus) DeepCopyInto(out *DSDBReplicationStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DSDBReplicationStatus.
func (in *DSDBReplicationStatus) DeepCopy() *DSDBReplicationStatus {
	if in == nil {
		return nil
	}
	out := new(DSDBReplicationStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DSReplicationStatus) DeepCopyInto(out *DSReplicationStatus) {
	*out = *in
	if in.DBs != nil {
		in, out := &in.DBs, &out.DBs
		*out = make([]DSDBReplicationStatus, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DSReplicationStatus.
func (in *DSReplicationStatus) DeepCopy() *DSReplicationStatus {
	if in == nil {
		return nil
	}
	out := new(DSReplicationStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EMQX) DeepCopyInto(out *EMQX) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EMQX.
func (in *EMQX) DeepCopy() *EMQX {
	if in == nil {
		return nil
	}
	out := new(EMQX)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EMQX) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EMQXCoreTemplate) DeepCopyInto(out *EMQXCoreTemplate) {
	*out = *in
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EMQXCoreTemplate.
func (in *EMQXCoreTemplate) DeepCopy() *EMQXCoreTemplate {
	if in == nil {
		return nil
	}
	out := new(EMQXCoreTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EMQXCoreTemplateSpec) DeepCopyInto(out *EMQXCoreTemplateSpec) {
	*out = *in
	in.EMQXReplicantTemplateSpec.DeepCopyInto(&out.EMQXReplicantTemplateSpec)
	in.VolumeClaimTemplates.DeepCopyInto(&out.VolumeClaimTemplates)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EMQXCoreTemplateSpec.
func (in *EMQXCoreTemplateSpec) DeepCopy() *EMQXCoreTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(EMQXCoreTemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EMQXList) DeepCopyInto(out *EMQXList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]EMQX, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EMQXList.
func (in *EMQXList) DeepCopy() *EMQXList {
	if in == nil {
		return nil
	}
	out := new(EMQXList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EMQXList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EMQXNode) DeepCopyInto(out *EMQXNode) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EMQXNode.
func (in *EMQXNode) DeepCopy() *EMQXNode {
	if in == nil {
		return nil
	}
	out := new(EMQXNode)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EMQXNodesStatus) DeepCopyInto(out *EMQXNodesStatus) {
	*out = *in
	if in.CollisionCount != nil {
		in, out := &in.CollisionCount, &out.CollisionCount
		*out = new(int32)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EMQXNodesStatus.
func (in *EMQXNodesStatus) DeepCopy() *EMQXNodesStatus {
	if in == nil {
		return nil
	}
	out := new(EMQXNodesStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EMQXReplicantTemplate) DeepCopyInto(out *EMQXReplicantTemplate) {
	*out = *in
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EMQXReplicantTemplate.
func (in *EMQXReplicantTemplate) DeepCopy() *EMQXReplicantTemplate {
	if in == nil {
		return nil
	}
	out := new(EMQXReplicantTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EMQXReplicantTemplateSpec) DeepCopyInto(out *EMQXReplicantTemplateSpec) {
	*out = *in
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
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
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.TopologySpreadConstraints != nil {
		in, out := &in.TopologySpreadConstraints, &out.TopologySpreadConstraints
		*out = make([]v1.TopologySpreadConstraint, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	if in.MinAvailable != nil {
		in, out := &in.MinAvailable, &out.MinAvailable
		*out = new(intstr.IntOrString)
		**out = **in
	}
	if in.MaxUnavailable != nil {
		in, out := &in.MaxUnavailable, &out.MaxUnavailable
		*out = new(intstr.IntOrString)
		**out = **in
	}
	if in.Command != nil {
		in, out := &in.Command, &out.Command
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Args != nil {
		in, out := &in.Args, &out.Args
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Ports != nil {
		in, out := &in.Ports, &out.Ports
		*out = make([]v1.ContainerPort, len(*in))
		copy(*out, *in)
	}
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]v1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.EnvFrom != nil {
		in, out := &in.EnvFrom, &out.EnvFrom
		*out = make([]v1.EnvFromSource, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.Resources.DeepCopyInto(&out.Resources)
	if in.PodSecurityContext != nil {
		in, out := &in.PodSecurityContext, &out.PodSecurityContext
		*out = new(v1.PodSecurityContext)
		(*in).DeepCopyInto(*out)
	}
	if in.ContainerSecurityContext != nil {
		in, out := &in.ContainerSecurityContext, &out.ContainerSecurityContext
		*out = new(v1.SecurityContext)
		(*in).DeepCopyInto(*out)
	}
	if in.InitContainers != nil {
		in, out := &in.InitContainers, &out.InitContainers
		*out = make([]v1.Container, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.ExtraContainers != nil {
		in, out := &in.ExtraContainers, &out.ExtraContainers
		*out = make([]v1.Container, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
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
	if in.LivenessProbe != nil {
		in, out := &in.LivenessProbe, &out.LivenessProbe
		*out = new(v1.Probe)
		(*in).DeepCopyInto(*out)
	}
	if in.ReadinessProbe != nil {
		in, out := &in.ReadinessProbe, &out.ReadinessProbe
		*out = new(v1.Probe)
		(*in).DeepCopyInto(*out)
	}
	if in.StartupProbe != nil {
		in, out := &in.StartupProbe, &out.StartupProbe
		*out = new(v1.Probe)
		(*in).DeepCopyInto(*out)
	}
	if in.Lifecycle != nil {
		in, out := &in.Lifecycle, &out.Lifecycle
		*out = new(v1.Lifecycle)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EMQXReplicantTemplateSpec.
func (in *EMQXReplicantTemplateSpec) DeepCopy() *EMQXReplicantTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(EMQXReplicantTemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EMQXSpec) DeepCopyInto(out *EMQXSpec) {
	*out = *in
	if in.ImagePullSecrets != nil {
		in, out := &in.ImagePullSecrets, &out.ImagePullSecrets
		*out = make([]v1.LocalObjectReference, len(*in))
		copy(*out, *in)
	}
	if in.BootstrapAPIKeys != nil {
		in, out := &in.BootstrapAPIKeys, &out.BootstrapAPIKeys
		*out = make([]BootstrapAPIKey, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	out.Config = in.Config
	if in.RevisionHistoryLimit != nil {
		in, out := &in.RevisionHistoryLimit, &out.RevisionHistoryLimit
		*out = new(int32)
		**out = **in
	}
	out.UpdateStrategy = in.UpdateStrategy
	in.CoreTemplate.DeepCopyInto(&out.CoreTemplate)
	if in.ReplicantTemplate != nil {
		in, out := &in.ReplicantTemplate, &out.ReplicantTemplate
		*out = new(EMQXReplicantTemplate)
		(*in).DeepCopyInto(*out)
	}
	if in.DashboardServiceTemplate != nil {
		in, out := &in.DashboardServiceTemplate, &out.DashboardServiceTemplate
		*out = new(ServiceTemplate)
		(*in).DeepCopyInto(*out)
	}
	if in.ListenersServiceTemplate != nil {
		in, out := &in.ListenersServiceTemplate, &out.ListenersServiceTemplate
		*out = new(ServiceTemplate)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EMQXSpec.
func (in *EMQXSpec) DeepCopy() *EMQXSpec {
	if in == nil {
		return nil
	}
	out := new(EMQXSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EMQXStatus) DeepCopyInto(out *EMQXStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.CoreNodes != nil {
		in, out := &in.CoreNodes, &out.CoreNodes
		*out = make([]EMQXNode, len(*in))
		copy(*out, *in)
	}
	in.CoreNodesStatus.DeepCopyInto(&out.CoreNodesStatus)
	if in.ReplicantNodes != nil {
		in, out := &in.ReplicantNodes, &out.ReplicantNodes
		*out = make([]EMQXNode, len(*in))
		copy(*out, *in)
	}
	in.ReplicantNodesStatus.DeepCopyInto(&out.ReplicantNodesStatus)
	if in.NodeEvacuationsStatus != nil {
		in, out := &in.NodeEvacuationsStatus, &out.NodeEvacuationsStatus
		*out = make([]NodeEvacuationStatus, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.DSReplication.DeepCopyInto(&out.DSReplication)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EMQXStatus.
func (in *EMQXStatus) DeepCopy() *EMQXStatus {
	if in == nil {
		return nil
	}
	out := new(EMQXStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EvacuationStrategy) DeepCopyInto(out *EvacuationStrategy) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EvacuationStrategy.
func (in *EvacuationStrategy) DeepCopy() *EvacuationStrategy {
	if in == nil {
		return nil
	}
	out := new(EvacuationStrategy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KeyRef) DeepCopyInto(out *KeyRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KeyRef.
func (in *KeyRef) DeepCopy() *KeyRef {
	if in == nil {
		return nil
	}
	out := new(KeyRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeEvacuationStats) DeepCopyInto(out *NodeEvacuationStats) {
	*out = *in
	if in.InitialSessions != nil {
		in, out := &in.InitialSessions, &out.InitialSessions
		*out = new(int32)
		**out = **in
	}
	if in.InitialConnected != nil {
		in, out := &in.InitialConnected, &out.InitialConnected
		*out = new(int32)
		**out = **in
	}
	if in.CurrentSessions != nil {
		in, out := &in.CurrentSessions, &out.CurrentSessions
		*out = new(int32)
		**out = **in
	}
	if in.CurrentConnected != nil {
		in, out := &in.CurrentConnected, &out.CurrentConnected
		*out = new(int32)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeEvacuationStats.
func (in *NodeEvacuationStats) DeepCopy() *NodeEvacuationStats {
	if in == nil {
		return nil
	}
	out := new(NodeEvacuationStats)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeEvacuationStatus) DeepCopyInto(out *NodeEvacuationStatus) {
	*out = *in
	in.Stats.DeepCopyInto(&out.Stats)
	if in.SessionRecipients != nil {
		in, out := &in.SessionRecipients, &out.SessionRecipients
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeEvacuationStatus.
func (in *NodeEvacuationStatus) DeepCopy() *NodeEvacuationStatus {
	if in == nil {
		return nil
	}
	out := new(NodeEvacuationStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Rebalance) DeepCopyInto(out *Rebalance) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Rebalance.
func (in *Rebalance) DeepCopy() *Rebalance {
	if in == nil {
		return nil
	}
	out := new(Rebalance)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Rebalance) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RebalanceCondition) DeepCopyInto(out *RebalanceCondition) {
	*out = *in
	in.LastUpdateTime.DeepCopyInto(&out.LastUpdateTime)
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RebalanceCondition.
func (in *RebalanceCondition) DeepCopy() *RebalanceCondition {
	if in == nil {
		return nil
	}
	out := new(RebalanceCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RebalanceList) DeepCopyInto(out *RebalanceList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Rebalance, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RebalanceList.
func (in *RebalanceList) DeepCopy() *RebalanceList {
	if in == nil {
		return nil
	}
	out := new(RebalanceList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RebalanceList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RebalanceSpec) DeepCopyInto(out *RebalanceSpec) {
	*out = *in
	out.RebalanceStrategy = in.RebalanceStrategy
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RebalanceSpec.
func (in *RebalanceSpec) DeepCopy() *RebalanceSpec {
	if in == nil {
		return nil
	}
	out := new(RebalanceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RebalanceState) DeepCopyInto(out *RebalanceState) {
	*out = *in
	if in.Recipients != nil {
		in, out := &in.Recipients, &out.Recipients
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Donors != nil {
		in, out := &in.Donors, &out.Donors
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RebalanceState.
func (in *RebalanceState) DeepCopy() *RebalanceState {
	if in == nil {
		return nil
	}
	out := new(RebalanceState)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RebalanceStatus) DeepCopyInto(out *RebalanceStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]RebalanceCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.RebalanceStates != nil {
		in, out := &in.RebalanceStates, &out.RebalanceStates
		*out = make([]RebalanceState, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.StartedTime.DeepCopyInto(&out.StartedTime)
	in.CompletedTime.DeepCopyInto(&out.CompletedTime)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RebalanceStatus.
func (in *RebalanceStatus) DeepCopy() *RebalanceStatus {
	if in == nil {
		return nil
	}
	out := new(RebalanceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RebalanceStrategy) DeepCopyInto(out *RebalanceStrategy) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RebalanceStrategy.
func (in *RebalanceStrategy) DeepCopy() *RebalanceStrategy {
	if in == nil {
		return nil
	}
	out := new(RebalanceStrategy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecretRef) DeepCopyInto(out *SecretRef) {
	*out = *in
	out.Key = in.Key
	out.Secret = in.Secret
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecretRef.
func (in *SecretRef) DeepCopy() *SecretRef {
	if in == nil {
		return nil
	}
	out := new(SecretRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceTemplate) DeepCopyInto(out *ServiceTemplate) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceTemplate.
func (in *ServiceTemplate) DeepCopy() *ServiceTemplate {
	if in == nil {
		return nil
	}
	out := new(ServiceTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UpdateStrategy) DeepCopyInto(out *UpdateStrategy) {
	*out = *in
	out.EvacuationStrategy = in.EvacuationStrategy
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UpdateStrategy.
func (in *UpdateStrategy) DeepCopy() *UpdateStrategy {
	if in == nil {
		return nil
	}
	out := new(UpdateStrategy)
	in.DeepCopyInto(out)
	return out
}
