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
	"testing"

	"github.com/emqx/emqx-operator/apis/apps/v1beta4"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var v1beta3Plugin = &EmqxPlugin{
	ObjectMeta: metav1.ObjectMeta{
		Name:        "test",
		Namespace:   "test",
		Labels:      map[string]string{"test": "test"},
		Annotations: map[string]string{"test": "test"},
	},
	Spec: EmqxPluginSpec{
		PluginName: "test",
		Selector:   map[string]string{"test": "test"},
		Config:     map[string]string{"test": "test"},
	},
}

var v1beta4Plugin = &v1beta4.EmqxPlugin{
	ObjectMeta: metav1.ObjectMeta{
		Name:        "test",
		Namespace:   "test",
		Labels:      map[string]string{"test": "test"},
		Annotations: map[string]string{"test": "test"},
	},
	Spec: v1beta4.EmqxPluginSpec{
		PluginName: "test",
		Selector:   map[string]string{"test": "test"},
		Config:     map[string]string{"test": "test"},
	},
}

func TestPluginConvertTo(t *testing.T) {
	plugin := &v1beta4.EmqxPlugin{}
	err := v1beta3Plugin.ConvertTo(plugin)
	assert.Nil(t, err)
	assert.ObjectsAreEqualValues(v1beta4Plugin, plugin)
}

func TestPluginConvertFrom(t *testing.T) {
	plugin := &EmqxPlugin{}
	err := plugin.ConvertFrom(v1beta4Plugin)
	assert.Nil(t, err)
	assert.ObjectsAreEqualValues(v1beta3Plugin, plugin)
}
