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

package v1beta4

import (
	appsv1 "k8s.io/api/apps/v1"
)

// StatefulSetsByCreationTimestamp sorts a list of StatefulSet by creation timestamp, using their names as a tie breaker.
type StatefulSetsByCreationTimestamp []*appsv1.StatefulSet

func (o StatefulSetsByCreationTimestamp) Len() int      { return len(o) }
func (o StatefulSetsByCreationTimestamp) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o StatefulSetsByCreationTimestamp) Less(i, j int) bool {
	if o[i].CreationTimestamp.Equal(&o[j].CreationTimestamp) {
		return o[i].Name < o[j].Name
	}
	return o[i].CreationTimestamp.Before(&o[j].CreationTimestamp)
}

// StatefulSetsBySizeOlder sorts a list of StatefulSet by size in descending order, using their creation timestamp or name as a tie breaker.
// By using the creation timestamp, this sorts from old to new replica sets.
type StatefulSetsBySizeOlder []*appsv1.StatefulSet

func (o StatefulSetsBySizeOlder) Len() int      { return len(o) }
func (o StatefulSetsBySizeOlder) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o StatefulSetsBySizeOlder) Less(i, j int) bool {
	if *(o[i].Spec.Replicas) == *(o[j].Spec.Replicas) {
		return StatefulSetsByCreationTimestamp(o).Less(i, j)
	}
	return *(o[i].Spec.Replicas) > *(o[j].Spec.Replicas)
}

// StatefulSetsBySizeNewer sorts a list of StatefulSet by size in descending order, using their creation timestamp or name as a tie breaker.
// By using the creation timestamp, this sorts from new to old replica sets.
type StatefulSetsBySizeNewer []*appsv1.StatefulSet

func (o StatefulSetsBySizeNewer) Len() int      { return len(o) }
func (o StatefulSetsBySizeNewer) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o StatefulSetsBySizeNewer) Less(i, j int) bool {
	if *(o[i].Spec.Replicas) == *(o[j].Spec.Replicas) {
		return StatefulSetsByCreationTimestamp(o).Less(j, i)
	}
	return *(o[i].Spec.Replicas) > *(o[j].Spec.Replicas)
}
