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

package config

import (
	"strings"
	"testing"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	"github.com/emqx/emqx-operator/test/util"
	"github.com/lithammer/dedent"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestRenderRoots(t *testing.T) {
	first := crd.ConfigRoots{
		"listeners": apiextv1.JSON{Raw: util.FromYAMLString(`
			tcp:
				default:
					bind: 1883
					enabled: true
		`)},
		"authentication": apiextv1.JSON{Raw: util.FromYAMLString(`
			- mechanism: password_based
			  backend:   built_in_database
		`)},
	}
	second := crd.ConfigRoots{
		"authentication": apiextv1.JSON{Raw: util.FromYAMLString(`
			- backend:   built_in_database
			  mechanism: password_based
		`)},
		"listeners": apiextv1.JSON{Raw: util.FromYAMLString(`
			tcp:
				default:
					enabled: true
					bind: 1883
		`)},
	}
	want := strings.TrimLeft(dedent.Dedent(`
		"authentication" = [{"backend":"built_in_database","mechanism":"password_based"}]
		"listeners" = {"tcp":{"default":{"bind":1883,"enabled":true}}}
	`), "\n")
	assert.Equal(t, want, RenderRoots(first))
	assert.Equal(t, want, RenderRoots(second))
}

func TestRenderBaseConfig(t *testing.T) {
	roots := crd.ConfigRoots{
		"dashboard": apiextv1.JSON{Raw: util.FromYAMLString(`
			listeners:
				http:
					bind: 0
		`)},
	}
	got := RenderBaseConfig(roots)
	assert.Contains(t, got, `http.bind = "0.0.0.0:18083"`)
	assert.Greater(t, len(got), len(EMQXDefaults))
}

func TestValidateRoots(t *testing.T) {
	assert.NoError(t, ValidateRoots(crd.ConfigRoots{
		"node": apiextv1.JSON{Raw: util.FromYAMLString(`name: emqx@host`)},
	}))
	assert.EqualError(t,
		ValidateRoots(crd.ConfigRoots{
			"node": apiextv1.JSON{Raw: util.FromYAMLString(`cookie: secret`)},
		}),
		"spec.config.roots.node.cookie is reserved for the Operator",
	)
	assert.EqualError(t,
		ValidateRoots(crd.ConfigRoots{
			"node": apiextv1.JSON{Raw: util.FromYAMLString(`cookie: null`)},
		}),
		"spec.config.roots.node.cookie is reserved for the Operator",
	)
}

func TestSplitRuntimeRoots(t *testing.T) {
	roots := crd.ConfigRoots{
		"listeners": apiextv1.JSON{Raw: util.FromYAMLString(`
			tcp:
				default:
					bind: 1883
		`)},
		"cluster": apiextv1.JSON{Raw: util.FromYAMLString(`
			name: emqx
			links:
				- name: remote
		`)},
		"durable_sessions": apiextv1.JSON{Raw: util.FromYAMLString(`
			enable: true
		`)},
		"rpc": apiextv1.JSON{Raw: util.FromYAMLString(`
			port_discovery: stateless
		`)},
	}
	runtimeRoots, restartRequired := SplitRuntimeRoots(roots)
	assert.Equal(t, []string{"cluster.name", "durable_sessions", "rpc"}, restartRequired)
	rendered := RenderRoots(runtimeRoots)
	assert.Equal(t,
		strings.TrimLeft(dedent.Dedent(`
			"cluster" = {"links":[{"name":"remote"}]}
			"listeners" = {"tcp":{"default":{"bind":1883}}}
		`), "\n"),
		rendered,
	)
}

func TestDashboardPortMap(t *testing.T) {
	tests := []struct {
		name  string
		roots crd.ConfigRoots
		want  map[string]int
	}{
		{name: "default",
			roots: crd.ConfigRoots{},
			want:  map[string]int{"dashboard": 18083}},
		{name: "integer HTTP bind",
			roots: crd.ConfigRoots{
				"dashboard": apiextv1.JSON{Raw: util.FromYAMLString(`
					listeners:
						http: { bind: 28083 }
				`)},
			},
			want: map[string]int{"dashboard": 28083}},
		{name: "IPv4 HTTP bind",
			roots: crd.ConfigRoots{
				"dashboard": apiextv1.JSON{Raw: util.FromYAMLString(`
					listeners:
						http: { bind: "0.0.0.0:28083" }
				`)},
			},
			want: map[string]int{"dashboard": 28083}},
		{name: "IPv6 HTTP bind",
			roots: crd.ConfigRoots{
				"dashboard": apiextv1.JSON{Raw: util.FromYAMLString(`
					listeners:
						http: { bind: "[::]:28083" }
				`)},
			},
			want: map[string]int{"dashboard": 28083}},
		{name: "HTTPS with default HTTP",
			roots: crd.ConfigRoots{
				"dashboard": apiextv1.JSON{Raw: util.FromYAMLString(`
					listeners:
						https: { bind: ":18084" }
				`)},
			},
			want: map[string]int{"dashboard": 18083, "dashboard-https": 18084}},
		{name: "disabled HTTP",
			roots: crd.ConfigRoots{
				"dashboard": apiextv1.JSON{Raw: util.FromYAMLString(`
					listeners:
						http:  { bind: 0 }
						https: { bind: 18084 }
				`)},
			},
			want: map[string]int{"dashboard-https": 18084}},
		{name: "all disabled",
			roots: crd.ConfigRoots{
				"dashboard": apiextv1.JSON{Raw: util.FromYAMLString(`
					listeners:
						http:  { bind: 0 }
						https: { bind: 0 }
				`)},
			},
			want: map[string]int{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, DashboardPortMap(tt.roots))
		})
	}
}

func TestDashboardServicePorts(t *testing.T) {
	roots := crd.ConfigRoots{
		"dashboard": apiextv1.JSON{Raw: util.FromYAMLString(`
			listeners:
				http:
					bind: 28083
				https:
					bind: 28084
		`)},
	}
	assert.Equal(t,
		[]corev1.ServicePort{
			{Name: "dashboard", Protocol: corev1.ProtocolTCP, Port: 28083, TargetPort: intstr.FromInt(28083)},
			{Name: "dashboard-https", Protocol: corev1.ProtocolTCP, Port: 28084, TargetPort: intstr.FromInt(28084)},
		},
		DashboardServicePorts(roots),
	)
}
