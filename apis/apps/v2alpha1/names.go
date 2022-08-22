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

package v2alpha1

import "fmt"

func (instance *EMQX) NameOfCoreNode() string {
	return fmt.Sprintf("%s-core", instance.Name)
}

func (instance *EMQX) NameOfCoreNodeData() string {
	return fmt.Sprintf("%s-core-data", instance.Name)
}

func (instance *EMQX) NameOfReplicantNode() string {
	return fmt.Sprintf("%s-replicant", instance.Name)
}

func (instance *EMQX) NameOfReplicantNodeData() string {
	return fmt.Sprintf("%s-replicant-data", instance.Name)
}

func (instance *EMQX) NameOfHeadlessService() string {
	return fmt.Sprintf("%s-headless", instance.Name)
}

func (instance *EMQX) NameOfDashboardService() string {
	return fmt.Sprintf("%s-dashboard", instance.Name)
}

func (instance *EMQX) NameOfListenersService() string {
	return fmt.Sprintf("%s-listeners", instance.Name)
}

func (instance *EMQX) NameOfBootStrapUser() string {
	return fmt.Sprintf("%s-bootstrap-user", instance.Name)
}

func (instance *EMQX) NameOfBootStrapConfig() string {
	return fmt.Sprintf("%s-bootstrap-config", instance.Name)
}
