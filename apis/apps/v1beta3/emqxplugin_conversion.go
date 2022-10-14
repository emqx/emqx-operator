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

package v1beta3

import (
	"github.com/emqx/emqx-operator/apis/apps/v1beta4"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts this version to the Hub version (v1).
func (src *EmqxPlugin) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1beta4.EmqxPlugin)
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec = v1beta4.EmqxPluginSpec{
		PluginName: src.Spec.PluginName,
		Selector:   src.Spec.Selector,
		Config:     src.Spec.Config,
	}

	// +kubebuilder:docs-gen:collapse=rote conversion
	return nil
}

// ConvertFrom converts from the Hub version (v1) to this version.
func (dst *EmqxPlugin) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1beta4.EmqxPlugin)
	src.ObjectMeta = dst.ObjectMeta
	src.Spec = v1beta4.EmqxPluginSpec{
		PluginName: dst.Spec.PluginName,
		Selector:   dst.Spec.Selector,
		Config:     dst.Spec.Config,
	}

	// +kubebuilder:docs-gen:collapse=rote conversion
	return nil
}
