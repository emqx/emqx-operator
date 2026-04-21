/*
Copyright 2021-2026.

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

package v3beta1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/types"
)

func (instance *EMQX) NamespacedName(name string) types.NamespacedName {
	return types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      name,
	}
}

func (instance *EMQX) CoreName() string {
	return fmt.Sprintf("%s-core", instance.Name)
}

func (instance *EMQX) CoreNamespacedName() types.NamespacedName {
	return instance.NamespacedName(instance.CoreName())
}

func (instance *EMQX) ReplicantName() string {
	return fmt.Sprintf("%s-replicant", instance.Name)
}

func (instance *EMQX) ReplicantNamespacedName() types.NamespacedName {
	return instance.NamespacedName(instance.ReplicantName())
}

func (instance *EMQX) HeadlessServiceNamespacedName() types.NamespacedName {
	return instance.NamespacedName(fmt.Sprintf("%s-headless", instance.Name))
}

func (instance *EMQX) DashboardServiceNamespacedName() types.NamespacedName {
	return instance.NamespacedName(fmt.Sprintf("%s-dashboard", instance.Name))
}

func (instance *EMQX) ListenersServiceNamespacedName() types.NamespacedName {
	return instance.NamespacedName(fmt.Sprintf("%s-listeners", instance.Name))
}

func (instance *EMQX) BootstrapAPIKeyNamespacedName() types.NamespacedName {
	return instance.NamespacedName(fmt.Sprintf("%s-bootstrap-api-key", instance.Name))
}

func (instance *EMQX) NodeCookieNamespacedName() types.NamespacedName {
	return instance.NamespacedName(fmt.Sprintf("%s-node-cookie", instance.Name))
}

func (instance *EMQX) ConfigsNamespacedName() types.NamespacedName {
	return instance.NamespacedName(fmt.Sprintf("%s-configs", instance.Name))
}
