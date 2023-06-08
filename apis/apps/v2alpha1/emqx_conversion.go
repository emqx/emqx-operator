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
	structAssign(dst, src)
	dst.SetGroupVersionKind(v2alpha2.GroupVersion.WithKind("EMQX"))

	// +kubebuilder:docs-gen:collapse=rote conversion
	return nil
}

// ConvertFrom converts from the Hub version (v1) to this version.
func (dst *EMQX) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v2alpha2.EMQX)
	structAssign(dst, src)
	dst.SetGroupVersionKind(GroupVersion.WithKind("EMQX"))

	// +kubebuilder:docs-gen:collapse=rote conversion
	return nil
}

func structAssign(dist, src interface{}) {
	dVal := reflect.ValueOf(dist).Elem()
	sVal := reflect.ValueOf(src).Elem()

	switch sVal.Type().Kind() {
	case reflect.Struct:
		for i := 0; i < sVal.NumField(); i++ {
			name := sVal.Type().Field(i).Name
			if dVal.FieldByName(name).IsValid() && dVal.FieldByName(name).CanSet() {
				structAssign(dVal.FieldByName(name).Addr().Interface(), sVal.FieldByName(name).Addr().Interface())
			}
		}
	case reflect.Array, reflect.Slice:
		for i := 0; i < sVal.Len(); i++ {
			dVal.Set(reflect.Append(dVal, sVal.Index(i)))
		}
	default:
		dVal.Set(reflect.ValueOf(sVal.Interface()))
	}
}
