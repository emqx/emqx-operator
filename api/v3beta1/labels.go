/*
Copyright 2025-2026.

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
	"maps"

	"k8s.io/apimachinery/pkg/labels"
)

const DefaultManagedBy = "emqx-operator"

func (instance *EMQX) DefaultLabels() map[string]string {
	labels := map[string]string{}
	labels[LabelInstance] = instance.Name
	labels[LabelManagedBy] = DefaultManagedBy
	return labels
}

func DefaultCacheLabels() labels.Set {
	return labels.Set{
		LabelManagedBy: DefaultManagedBy,
	}
}

func DefaultCacheSelector() labels.Selector {
	return labels.SelectorFromSet(DefaultCacheLabels())
}

func (instance *EMQX) DefaultLabelsWith(extraLabels ...map[string]string) map[string]string {
	labels := instance.DefaultLabels()
	for _, ls := range extraLabels {
		maps.Copy(labels, ls)
	}
	return labels
}

func CoreLabels() map[string]string {
	return map[string]string{LabelMriaRole: RoleCore}
}

func ReplicantLabels() map[string]string {
	return map[string]string{LabelMriaRole: RoleReplicant}
}
