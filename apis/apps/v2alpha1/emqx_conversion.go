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

import (
	"reflect"

	"github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts this version to the Hub version (v1).
func (src *EMQX) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v2alpha2.EMQX)
	dst.ObjectMeta = src.ObjectMeta
	structAssign(&dst.Spec, &src.Spec)

	// +kubebuilder:docs-gen:collapse=rote conversion
	return nil
}

// ConvertFrom converts from the Hub version (v1) to this version.
func (dst *EMQX) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v2alpha2.EMQX)
	dst.ObjectMeta = src.ObjectMeta
	structAssign(&dst.Spec, &src.Spec)

	// +kubebuilder:docs-gen:collapse=rote conversion
	return nil
}

func structAssign(dist, src interface{}) {
	dVal := reflect.ValueOf(dist).Elem()
	sVal := reflect.ValueOf(src).Elem()
	sType := sVal.Type()
	for i := 0; i < sVal.NumField(); i++ {
		// we need to check if the dist struct has the same field
		name := sType.Field(i).Name
		if ok := dVal.FieldByName(name).IsValid(); ok {
			dVal.FieldByName(name).Set(reflect.ValueOf(sVal.Field(i).Interface()))
		}
	}
}
