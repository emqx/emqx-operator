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
	"testing"

	"github.com/emqx/emqx-operator/apis/apps/v2beta1"
	"github.com/stretchr/testify/assert"
)

func TestConvertTo(t *testing.T) {
	dst := &v2beta1.Rebalance{}
	src := &Rebalance{}

	assert.Nil(t, src.ConvertTo(dst))
}

func TestConvertFrom(t *testing.T) {
	dst := &Rebalance{}
	src := &v2beta1.Rebalance{}

	assert.Nil(t, dst.ConvertFrom(src))
}
