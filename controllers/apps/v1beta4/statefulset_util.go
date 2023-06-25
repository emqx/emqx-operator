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
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash"
	"hash/fnv"

	emperror "emperror.dev/errors"
	"github.com/cisco-open/k8s-objectmatcher/patch"
	"github.com/davecgh/go-spew/spew"
	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	"github.com/tidwall/gjson"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/rand"
)

// Helper functions
func getEmqxNodeName(instance appsv1beta4.Emqx, pod *corev1.Pod) string {
	return fmt.Sprintf(
		"%s@%s.%s",
		instance.GetSpec().GetTemplate().Spec.EmqxContainer.EmqxConfig["name"],
		pod.Name,
		instance.GetSpec().GetTemplate().Spec.EmqxContainer.EmqxConfig["cluster.dns.name"],
	)
}

// JustCheckPodTemplate will check only the differences between the podTemplate of the two statefulSets
func justCheckPodTemplate() patch.CalculateOption {
	getPodTemplate := func(obj []byte) ([]byte, error) {
		podTemplateJson := gjson.GetBytes(obj, "spec.template")
		podTemplate := &corev1.PodTemplateSpec{}
		_ = json.Unmarshal([]byte(podTemplateJson.String()), podTemplate)

		emptySts := &appsv1.StatefulSet{}
		emptySts.Spec.Template = *podTemplate
		return json.Marshal(emptySts)
	}

	return func(current, modified []byte) ([]byte, []byte, error) {
		current, err := getPodTemplate(current)
		if err != nil {
			return []byte{}, []byte{}, emperror.Wrap(err, "could not delete the field from current byte sequence")
		}

		modified, err = getPodTemplate(modified)
		if err != nil {
			return []byte{}, []byte{}, emperror.Wrap(err, "could not delete the field from modified byte sequence")
		}

		return current, modified, nil
	}
}

// ComputeHash returns a hash value calculated from pod template and
// a collisionCount to avoid hash collision. The hash will be safe encoded to
// avoid bad words.
func computeHash(template *corev1.PodTemplateSpec, collisionCount *int32) string {
	templateSpecHasher := fnv.New32a()
	deepHashObject(templateSpecHasher, *template)

	// Add collisionCount in the hash if it exists.
	if collisionCount != nil {
		collisionCountBytes := make([]byte, 8)
		binary.LittleEndian.PutUint32(collisionCountBytes, uint32(*collisionCount))
		templateSpecHasher.Write(collisionCountBytes)
	}

	return rand.SafeEncodeString(fmt.Sprint(templateSpecHasher.Sum32()))
}

// DeepHashObject writes specified object to hash using the spew library
// which follows pointers and prints actual values of the nested objects
// ensuring the hash does not change when a pointer changes.
func deepHashObject(hasher hash.Hash, objectToWrite interface{}) {
	hasher.Reset()
	printer := spew.ConfigState{
		Indent:         " ",
		SortKeys:       true,
		DisableMethods: true,
		SpewKeys:       true,
	}
	printer.Fprintf(hasher, "%#v", objectToWrite)
}

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

// PodsByCreationTimestamp sorts a list of Pod by creation timestamp, using their names as a tie breaker.
type PodsByCreationTimestamp []*corev1.Pod

func (o PodsByCreationTimestamp) Len() int      { return len(o) }
func (o PodsByCreationTimestamp) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o PodsByCreationTimestamp) Less(i, j int) bool {
	if o[i].CreationTimestamp.Equal(&o[j].CreationTimestamp) {
		return o[i].Name < o[j].Name
	}
	return o[i].CreationTimestamp.Before(&o[j].CreationTimestamp)
}

// PodsByNameOlder sorts a list of Pod by size in descending order, using their creation timestamp or name as a tie breaker.
// By using the creation timestamp, this sorts from old to new replica sets.
type PodsByNameOlder []*corev1.Pod

func (o PodsByNameOlder) Len() int      { return len(o) }
func (o PodsByNameOlder) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o PodsByNameOlder) Less(i, j int) bool {
	if o[i].Name == o[j].Name {
		return PodsByCreationTimestamp(o).Less(i, j)
	}
	return o[i].Name > o[j].Name
}

// PodsByNameNewer sorts a list of Pod by size in descending order, using their creation timestamp or name as a tie breaker.
// By using the creation timestamp, this sorts from new to old replica sets.
type PodsByNameNewer []*corev1.Pod

func (o PodsByNameNewer) Len() int      { return len(o) }
func (o PodsByNameNewer) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o PodsByNameNewer) Less(i, j int) bool {
	if o[i].Name == o[j].Name {
		return PodsByCreationTimestamp(o).Less(j, i)
	}
	return o[i].Name > o[j].Name
}
