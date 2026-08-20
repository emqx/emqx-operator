/*
Copyright 2026.

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

package e2e

import (
	"fmt"

	"github.com/emqx/emqx-operator/test/util"
)

const defaultConfigYAML = `
license:
  key: evaluation
log:
  console:
    level: info
`

const configDS = `
durable_sessions:
  enable: true
durable_storage:
  messages:
    backend: builtin_raft
    n_shards: 8
`

type specBuilder struct {
	base     []byte
	overlays [][]byte
}

func SpecFromYAMLFile(filename string) *specBuilder {
	return &specBuilder{base: util.FromYAMLFile(filename)}
}

func (sb *specBuilder) WithCores(numReplicas int) *specBuilder {
	sb.overlays = append(sb.overlays, withCores(numReplicas))
	return sb
}

func (sb *specBuilder) WithReplicants(numReplicas int) *specBuilder {
	sb.overlays = append(sb.overlays, withReplicants(numReplicas))
	return sb
}

func (sb *specBuilder) WithImage(image string) *specBuilder {
	sb.overlays = append(sb.overlays, withImage(image))
	return sb
}

func (sb *specBuilder) WithConfig(snippets ...string) *specBuilder {
	sb.overlays = append(sb.overlays, withConfig(snippets...))
	return sb
}

func (sb *specBuilder) WithDS() *specBuilder {
	sb.overlays = append(sb.overlays, withDS())
	return sb
}

func (sb *specBuilder) ToJSONDocument() []byte {
	return util.PatchDocument(sb.base, sb.overlays...)
}

func withCores(numReplicas int) []byte {
	return fmt.Appendf(nil,
		`{"spec": {"coreTemplate": {"spec": {"replicas": %d}}}}`,
		numReplicas,
	)
}

func withReplicants(numReplicas int) []byte {
	return fmt.Appendf(nil,
		`{"spec": {"replicantTemplate": {"spec": {"minReadySeconds": 3, "replicas": %d}}}}`,
		numReplicas,
	)
}

func withImage(image string) []byte {
	return fmt.Appendf(nil, `{"spec": {"image": "%s"}}`, image)
}

func withConfig(snippets ...string) []byte {
	snippets = append([]string{defaultConfigYAML}, snippets...)
	return fmt.Appendf(nil, `{"spec": {"config": {"roots": %s}}}`, buildConfigRoots(snippets...))
}

func withDS() []byte {
	return withConfig(configDS)
}

func buildConfigRoots(snippets ...string) []byte {
	roots := []byte(`{}`)
	for _, snippet := range snippets {
		roots = util.PatchDocument(roots, util.FromYAMLString(snippet))
	}
	return roots
}
