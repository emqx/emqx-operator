/*
Copyright 2025.

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

package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripNonChangeableConfig(t *testing.T) {
	t.Run("empty configs", func(t *testing.T) {
		config, stripped := preserveNonChangeableConfig("", "")
		assert.Equal(t, "", config)
		assert.Equal(t, []string{}, stripped)
	})

	t.Run("http port changed", func(t *testing.T) {
		config, stripped := preserveNonChangeableConfig(
			"dashboard.listeners.http.bind = 18083",
			"dashboard.listeners.http.bind = 18084",
		)
		assert.Equal(t, `"dashboard" {"listeners":{"http":{"bind":18084}}}`+"\n", config)
		assert.Equal(t, []string{"dashboard.listeners.http.bind"}, stripped)
	})

	t.Run("http port unchanged", func(t *testing.T) {
		config, stripped := preserveNonChangeableConfig(
			"dashboard.listeners.http.bind = 18083",
			"dashboard.listeners.http.bind = 18083",
		)
		assert.Equal(t, "dashboard.listeners.http.bind = 18083", config)
		assert.Equal(t, []string{}, stripped)
	})

	t.Run("https port changed", func(t *testing.T) {
		config, stripped := preserveNonChangeableConfig(
			"dashboard.listeners.https.bind = 18083",
			"dashboard.listeners.https.bind = 18084",
		)
		assert.Equal(t, `"dashboard" {"listeners":{"https":{"bind":18084}}}`+"\n", config)
		assert.Equal(t, []string{"dashboard.listeners.https.bind"}, stripped)
	})

	t.Run("https port enabled", func(t *testing.T) {
		config, stripped := preserveNonChangeableConfig(
			"dashboard.listeners.https.bind = 18883",
			"dashboard.listeners.https.bind = 0",
		)
		assert.Equal(t, "dashboard.listeners.https.bind = 18883", config)
		assert.Equal(t, []string{}, stripped)
	})

	t.Run("both ports changed", func(t *testing.T) {
		config, stripped := preserveNonChangeableConfig(
			"dashboard.listeners { http.bind = 18883, https.bind = 18884 }",
			"dashboard.listeners { http.bind = 18083, https.bind = 18084 }",
		)
		assert.Equal(t, `"dashboard" {"listeners":{"http":{"bind":18083},"https":{"bind":18884}}}`+"\n", config)
		assert.Equal(t, []string{"dashboard.listeners.http.bind"}, stripped)
	})

	t.Run("http port unchanged but different type", func(t *testing.T) {
		config, stripped := preserveNonChangeableConfig(
			`dashboard.listeners.http.bind = 18083`,
			`dashboard.listeners.http.bind = "18083"`,
		)
		assert.Equal(t, "dashboard.listeners.http.bind = 18083", config)
		assert.Equal(t, []string{}, stripped)
	})

}
