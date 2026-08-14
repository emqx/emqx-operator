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
	"fmt"
	"sort"
	"strings"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	jsoniter "github.com/json-iterator/go"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var json = jsoniter.Config{
	EscapeHTML:  false,
	SortMapKeys: true,
	UseNumber:   true,
}.Froze()

const EMQXDefaults string = `
### Minimal default configuration
# This is generally provided by 'emqx.conf' but defined here instead, as 'emqx.conf' should now be empty.
dashboard {
  listeners {
    http.bind = "0.0.0.0:18083"
  }
}

### User-supplied configuration
`

func WithDefaults(config string) string {
	return EMQXDefaults + config
}

// RenderRoots serializes top-level configuration roots as deterministic HOCON
// assignments. Root names and values use JSON syntax, which is valid HOCON.
func RenderRoots(roots crd.ConfigRoots) string {
	var builder strings.Builder
	keys := make([]string, 0, len(roots))
	for key := range roots {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		canonical := canonicalJSON(roots[key].Raw)
		quotedKey, _ := json.Marshal(key)
		builder.Write(quotedKey)
		builder.WriteString(" = ")
		builder.Write(canonical)
		builder.WriteByte('\n')
	}
	return builder.String()
}

func RenderBaseConfig(roots crd.ConfigRoots) string {
	return WithDefaults(RenderRoots(roots))
}

func ValidateRoots(roots crd.ConfigRoots) error {
	node, ok := roots["node"]
	if !ok {
		return nil
	}
	if json.Get(node.Raw, "cookie").ValueType() != jsoniter.InvalidValue {
		return fmt.Errorf("spec.config.roots.node.cookie is reserved for the Operator")
	}
	return nil
}

// SplitRuntimeRoots returns the configuration that can be sent to the EMQX
// runtime Configs API and the paths that require a Pod restart.
func SplitRuntimeRoots(roots crd.ConfigRoots) (crd.ConfigRoots, []string) {
	runtimeRoots := make(crd.ConfigRoots, len(roots))
	restartRequired := []string{}
	for root, value := range roots {
		switch root {
		case "node", "rpc", "durable_sessions", "durable_storage":
			restartRequired = append(restartRequired, root)
		case "cluster":
			rootValue := map[string]any{}
			err := json.Unmarshal(value.Raw, &rootValue)
			if err != nil {
				restartRequired = append(restartRequired, root)
				continue
			}
			if links, exists := rootValue["links"]; exists {
				rootJson, _ := json.Marshal(map[string]any{"links": links})
				runtimeRoots[root] = apiextv1.JSON{Raw: rootJson}
			}
			for key := range rootValue {
				if key != "links" {
					restartRequired = append(restartRequired, root+"."+key)
				}
			}
		default:
			runtimeRoots[root] = *value.DeepCopy()
		}
	}
	sort.Strings(restartRequired)
	return runtimeRoots, restartRequired
}

func DashboardPortMap(roots crd.ConfigRoots) map[string]int {
	portMap := map[string]int{"dashboard": 18083}
	dashboard, ok := roots["dashboard"]
	if !ok {
		return portMap
	}

	bind := json.Get(dashboard.Raw, "listeners", "http", "bind")
	if bind.ValueType() != jsoniter.InvalidValue {
		if port := dashboardBindPort(bind); port != 0 {
			portMap["dashboard"] = port
		} else {
			delete(portMap, "dashboard")
		}
	}

	bind = json.Get(dashboard.Raw, "listeners", "https", "bind")
	if bind.ValueType() != jsoniter.InvalidValue {
		if port := dashboardBindPort(bind); port != 0 {
			portMap["dashboard-https"] = port
		}
	}

	return portMap
}

func DashboardServicePorts(roots crd.ConfigRoots) []corev1.ServicePort {
	portMap := DashboardPortMap(roots)
	ports := make([]corev1.ServicePort, 0, len(portMap))
	for name, port := range portMap {
		ports = append(ports, corev1.ServicePort{
			Name:       name,
			Protocol:   corev1.ProtocolTCP,
			Port:       int32(port),
			TargetPort: intstr.FromInt(port),
		})
	}
	sort.Slice(ports, func(i, j int) bool {
		return ports[i].Name < ports[j].Name
	})
	return ports
}

func dashboardBindPort(value jsoniter.Any) int {
	switch value.ValueType() {
	case jsoniter.StringValue, jsoniter.NumberValue:
	default:
		return 0
	}
	port, err := util.ParseBindPort(value.ToString())
	if err != nil {
		return 0
	}
	return int(port)
}

func canonicalJSON(raw []byte) []byte {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		panic(fmt.Errorf("invalid apiextv1.JSON value: %w", err))
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Errorf("failed to marshal decoded apiextv1.JSON value: %w", err))
	}
	return canonical
}
