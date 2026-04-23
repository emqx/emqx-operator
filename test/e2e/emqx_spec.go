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
