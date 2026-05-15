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
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/rory-z/go-hocon"
	corev1 "k8s.io/api/core/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

const EMQXDefaults string = `
### Minimal default configuration
# This is generally provided by 'emqx.conf' but defined here instead, as 'emqx.conf' should now be empty.
dashboard.listeners.http.bind = 18083

### User-supplied configuration
`

type EMQX struct {
	*hocon.Config
}

func WithDefaults(config string) string {
	return EMQXDefaults + config
}

func EMQXConfigWithDefaults(config string) (*EMQX, error) {
	return EMQXConfig(WithDefaults(config))
}

func EMQXConfig(config string) (*EMQX, error) {
	c := &EMQX{}
	err := c.LoadEMQXConf(config)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *EMQX) LoadEMQXConf(config string) error {
	hoconConfig, err := hocon.ParseString(config)
	if err != nil {
		return err
	}
	c.Config = hoconConfig
	return nil
}

func (c *EMQX) Copy() *EMQX {
	root := hocon.Object{}
	return &EMQX{
		// Should deep-copy `c.config`.
		root.ToConfig().WithFallback(c.Config),
	}
}

func (c *EMQX) Print() string {
	return printObject(c.GetRoot().(hocon.Object), true)
}

func printValue(v hocon.Value) string {
	switch v.Type() {
	case hocon.ObjectType:
		return printObject(v.(hocon.Object), false)
	case hocon.ArrayType:
		return printArray(v.(hocon.Array))
	default:
		return v.String()
	}
}

func printObject(o hocon.Object, root bool) string {
	builder := strings.Builder{}
	n := len(o)
	if !root {
		builder.WriteString("{")
	}
	keys := make([]string, 0, n)
	for k := range o {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, key := range keys {
		value := o[key]
		builder.WriteString(key)
		if value.Type() != hocon.ObjectType {
			builder.WriteString(" = ")
		} else {
			builder.WriteString(" ")
		}
		builder.WriteString(printValue(value))
		if i < n-1 {
			builder.WriteString(", ")
		}
	}
	if !root {
		builder.WriteString("}")
	}
	return builder.String()
}

func printArray(a hocon.Array) string {
	var builder strings.Builder
	builder.WriteString("[")
	for i, value := range a {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(printValue(value))
	}
	builder.WriteString("]")
	return builder.String()
}

func (c *EMQX) StripReadOnlyConfig() []string {
	root := c.GetRoot().(hocon.Object)
	stripped := []string{}
	for _, key := range []string{
		// Explicitly considered read-only:
		"node",
		"rpc",
		// Effectively read-only, not changeable in runtime:
		"durable_sessions",
		"durable_storage",
	} {
		if _, ok := root[key]; ok {
			stripped = append(stripped, key)
			delete(root, key)
		}
	}
	// Cluster configuration is also read-only, except for:
	// * `cluster.links` is changeable in runtime.
	if subrootValue, ok := root["cluster"]; ok {
		subroot, ok := subrootValue.(hocon.Object)
		if ok {
			for key := range subroot {
				if key == "links" {
					continue
				}
				stripped = append(stripped, "cluster."+key)
				delete(subroot, key)
			}
		}
	}
	return stripped
}

func (c *EMQX) Strip(path string) bool {
	object := c.GetRoot().(hocon.Object)
	keys := strings.Split(path, ".")
	l := len(keys)
	if l > 0 {
		return stripRecursive(object, keys, 0, l)
	}
	return false
}

func stripRecursive(object hocon.Object, keys []string, i int, l int) bool {
	key := keys[i]
	val, ok := object[key]
	if !ok {
		return false
	}
	if i == l-1 {
		delete(object, key)
		return true
	}
	inner, ok := val.(hocon.Object)
	if ok && stripRecursive(inner, keys, i+1, l) {
		if len(inner) == 0 {
			delete(object, key)
		}
		return true
	}
	return false
}

func (c *EMQX) GetNodeCookie() string {
	return toString(byDefault(c.Get("node.cookie"), ""))
}

func (c *EMQX) GetDashboardPortMap() map[string]int {
	portMap := make(map[string]int)
	portMap["dashboard"] = 18083 // default port

	httpBind := byDefault(c.Get("dashboard.listeners.http.bind"), "")
	dashboardPort := toString(httpBind)
	if dashboardPort != "" {
		if !strings.Contains(dashboardPort, ":") {
			// example: ":18083"
			dashboardPort = fmt.Sprintf(":%s", dashboardPort)
		}
		_, strPort, _ := net.SplitHostPort(dashboardPort)
		if port, _ := strconv.Atoi(strPort); port != 0 {
			portMap["dashboard"] = port
		} else {
			// port = 0 means disable dashboard
			// delete default port
			delete(portMap, "dashboard")
		}
	}

	httpsBind := byDefault(c.Get("dashboard.listeners.https.bind"), "")
	dashboardHttpsPort := toString(httpsBind)
	if dashboardHttpsPort != "" {
		if !strings.Contains(dashboardHttpsPort, ":") {
			// example: ":18084"
			dashboardHttpsPort = fmt.Sprintf(":%s", dashboardHttpsPort)
		}
		_, strPort, _ := net.SplitHostPort(dashboardHttpsPort)
		if port, _ := strconv.Atoi(strPort); port != 0 {
			portMap["dashboard-https"] = port
		} else {
			// port = 0 means disable dashboard
			// delete default port
			delete(portMap, "dashboard-https")
		}
	}

	return portMap
}

func (c *EMQX) GetDashboardServicePorts() []corev1.ServicePort {
	portList := []corev1.ServicePort{}
	portMap := c.GetDashboardPortMap()

	for name, port := range portMap {
		portList = append(portList, corev1.ServicePort{
			Name:       name,
			Protocol:   corev1.ProtocolTCP,
			Port:       int32(port),
			TargetPort: intstr.FromInt(port),
		})
	}

	sort.Slice(portList, func(i, j int) bool {
		return portList[i].Name < portList[j].Name
	})

	return portList
}

func (c *EMQX) GetListenersServicePorts() []corev1.ServicePort {
	portList := []corev1.ServicePort{}

	// May be empty
	for t, listener := range c.GetObject("listeners") {
		if listener.Type() != hocon.ObjectType {
			continue
		}
		for name, lc := range listener.(hocon.Object) {
			lconf := lc.(hocon.Object)
			// Compatible with "enable" and "enabled"
			// the default value of them both is true
			enabled := byDefault(lconf["enable"], byDefault(lconf["enabled"], true))
			if isFalse(enabled) {
				continue
			}
			bind := toString(byDefault(lconf["bind"], ":0"))
			if !strings.Contains(bind, ":") {
				// example: ":1883"
				bind = fmt.Sprintf(":%s", bind)
			}
			_, strPort, _ := net.SplitHostPort(bind)
			intStrValue := intstr.Parse(strPort)

			protocol := corev1.ProtocolTCP
			if t == "quic" {
				protocol = corev1.ProtocolUDP
			}

			portList = append(portList, corev1.ServicePort{
				Name:       fmt.Sprintf("%s-%s", t, name),
				Protocol:   protocol,
				Port:       int32(intStrValue.IntValue()),
				TargetPort: intStrValue,
			})
		}
	}

	// Get gateway.lwm2m.listeners.udp.default.bind
	for proto, gc := range c.GetObject("gateway") {
		gateway := gc.(hocon.Object)
		// Compatible with "enable" and "enabled"
		// the default value of them both is true
		enabled := byDefault(gateway["enable"], byDefault(gateway["enabled"], true))
		if isFalse(enabled) {
			continue
		}
		listeners := gateway["listeners"].(hocon.Object)
		for t, listener := range listeners {
			if listener.Type() != hocon.ObjectType {
				continue
			}
			for name, lc := range listener.(hocon.Object) {
				lconf := lc.(hocon.Object)
				// Compatible with "enable" and "enabled"
				// the default value of them both is true
				enabled := byDefault(lconf["enable"], byDefault(lconf["enabled"], true))
				if isFalse(enabled) {
					continue
				}
				bind := toString(byDefault(lconf["bind"], ":0"))
				if !strings.Contains(bind, ":") {
					// example: ":1883"
					bind = fmt.Sprintf(":%s", bind)
				}
				_, strPort, _ := net.SplitHostPort(bind)
				intStrValue := intstr.Parse(strPort)

				protocol := corev1.ProtocolTCP
				if t == "udp" || t == "dtls" {
					protocol = corev1.ProtocolUDP
				}

				portList = append(portList, corev1.ServicePort{
					Name:       fmt.Sprintf("%s-%s-%s", proto, t, name),
					Protocol:   protocol,
					Port:       int32(intStrValue.IntValue()),
					TargetPort: intStrValue,
				})
			}
		}
	}

	sort.Slice(portList, func(i, j int) bool {
		return portList[i].Name < portList[j].Name
	})

	return portList
}

/* hocon.Config helper functions */

func byDefault(v hocon.Value, def any) hocon.Value {
	if v == nil {
		switch def := def.(type) {
		case hocon.Value:
			return def
		case bool:
			return hocon.Boolean(def)
		case string:
			return hocon.String(def)
		case int:
			return hocon.Int(def)
		default:
			panic(fmt.Sprintf("unsupported type: %T", def))
		}
	}
	return v
}

func isTrue(v hocon.Value) bool {
	if v.Type() != hocon.BooleanType {
		return false
	}
	return bool(v.(hocon.Boolean))
}

func isFalse(v hocon.Value) bool {
	return !isTrue(v)
}

func toString(v hocon.Value) string {
	switch v.Type() {
	case hocon.StringType:
		return string(v.(hocon.String))
	case hocon.BooleanType:
		return v.String()
	case hocon.NumberType:
		return v.String()
	}
	return ""
}
