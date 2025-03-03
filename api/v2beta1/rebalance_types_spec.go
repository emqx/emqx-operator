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

package v2beta1

type RebalanceSpec struct {
	// InstanceKind is used to distinguish between EMQX and EMQXEnterprise.
	// When it is set to "EMQX", it means that the EMQX CR is v2beta1,
	// and when it is set to "EmqxEnterprise", it means that the EmqxEnterprise CR is v1beta4.
	// +kubebuilder:default:="EMQX"
	InstanceKind string `json:"instanceKind"`
	// InstanceName represents the name of EMQX CR, just work for EMQX Enterprise
	// +kubebuilder:validation:Required
	InstanceName string `json:"instanceName"`
	// RebalanceStrategy represents the strategy of EMQX rebalancing
	// More info: https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing
	// +kubebuilder:validation:Required
	RebalanceStrategy RebalanceStrategy `json:"rebalanceStrategy"`
}

type RebalanceStrategy struct {
	// ConnEvictRate represents the source node client disconnect rate per second.
	// same to conn-evict-rate in [EMQX Rebalancing](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing)
	// The value must be greater than 0
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	ConnEvictRate int32 `json:"connEvictRate"`
	// SessEvictRate represents the source node session evacuation rate per second.
	// same to sess-evict-rate in [EMQX Rebalancing](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing)
	// The value must be greater than 0
	// Defaults to 500.
	// +kubebuilder:default:=500
	SessEvictRate int32 `json:"sessEvictRate,omitempty"`
	// WaitTakeover represents the time in seconds to wait for a client to
	// reconnect to take over the session after all connections are disconnected.
	// same to wait-takeover in [EMQX Rebalancing](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing)
	// The value must be greater than 0
	// Defaults to 60 seconds.
	// +kubebuilder:default:=60
	WaitTakeover int32 `json:"waitTakeover,omitempty"`
	// WaitHealthCheck represents the time (in seconds) to wait for the LB to
	// remove the source node from the list of active backend nodes. After the
	// specified waiting time is exceeded,the rebalancing task will start.
	// same to wait-health-check in [EMQX Rebalancing](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing)
	// The value must be greater than 0
	// Defaults to 60 seconds.
	// +kubebuilder:default:=60
	WaitHealthCheck int32 `json:"waitHealthCheck,omitempty"`
	// AbsConnThreshold represents the absolute threshold for checking connection balance.
	// same to abs-conn-threshold in [EMQX Rebalancing](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing)
	// The value must be greater than 0
	// Defaults to 1000.
	// +kubebuilder:default:=1000
	AbsConnThreshold int32 `json:"absConnThreshold,omitempty"`
	// RelConnThreshold represents the relative threshold for checkin connection balance.
	// same to rel-conn-threshold in [EMQX Rebalancing](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing)
	// the usage of float highly discouraged, as support for them varies across languages.
	// So we define the RelConnThreshold field as string type and you not float type
	// The value must be greater than "1.0"
	// Defaults to "1.1".
	// +kubebuilder:default:="1.1"
	RelConnThreshold string `json:"relConnThreshold,omitempty"`
	// AbsSessThreshold represents the absolute threshold for checking session connection balance.
	// same to abs-sess-threshold in [EMQX Rebalancing](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing)
	// The value must be greater than 0
	// Default to 1000.
	// +kubebuilder:default:=1000
	AbsSessThreshold int32 `json:"absSessThreshold,omitempty"`
	// RelSessThreshold represents the relative threshold for checking session connection balance.
	// same to rel-sess-threshold in [EMQX Rebalancing](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing)
	// the usage of float highly discouraged, as support for them varies across languages.
	// So we define the RelSessThreshold field as string type and you not float type
	// The value must be greater than "1.0"
	// Defaults to "1.1".
	// +kubebuilder:default:="1.1"
	RelSessThreshold string `json:"relSessThreshold,omitempty"`
}
