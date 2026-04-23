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
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/emqx/emqx-operator/test/util"
	"github.com/lithammer/dedent"
)

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
	defaults := []string{ConfigLicense(), ConfigConsoleLog("info")}
	config := slices.Concat(defaults, snippets)
	return fmt.Appendf(nil, `{"spec": {"config": {"data": %s}}}`, intoJsonString(config...))
}

func intoJsonString(snippets ...string) []byte {
	configStr := dedent.Dedent(strings.Join(snippets, ""))
	jsonStr, _ := json.Marshal(configStr)
	return jsonStr
}

func ConfigLicense() string {
	return `
		license { key = "evaluation" }
	`
}

func ConfigConsoleLog(level string) string {
	return `
		log.console { level = "` + level + `" }
	`
}

func ConfigDS() string {
	return `
		durable_sessions { enable = true }
		durable_storage { 
			messages {
				backend = builtin_raft
				n_shards = 8
			}
		}
	`
}
