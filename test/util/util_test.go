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

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromYAMLString(t *testing.T) {
	t.Run("FromYAMLString", func(t *testing.T) {
		json := FromYAMLString(`
			apiVersion: autoscaling/v2
			kind: HorizontalPodAutoscaler
			metadata:
			  name: test-json
			spec:
			  minReplicas: 1
			  maxReplicas: 42
			  scaleTargetRef:
			    apiVersion: apps.emqx.io/v3beta1
			    kind: EMQX
			    name: emqx
			  metrics:
			  - type: Resource
			    resource:
			      name: cpu
			      target:
			        averageUtilization: 80
			        type: Utilization
			  behavior:
			    scaleUp:
				  stabilizationWindowSeconds: 0
			    scaleDown:
				  stabilizationWindowSeconds: 30
			`)
		assert.NotEmpty(t, json)
	})
}
