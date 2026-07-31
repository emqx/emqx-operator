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

	"github.com/emqx/emqx-operator/hocon"
	json "github.com/json-iterator/go"
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
	Config hocon.Object
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
	doc, err := hocon.ParseDocument(config)
	if err != nil {
		return err
	}
	root, err := doc.Evaluate()
	if err != nil {
		return err
	}
	c.Config = root
	return nil
}

func (c *EMQX) Copy() *EMQX {
	return &EMQX{
		Config: c.Config.DeepCopy(),
	}
}

func (c *EMQX) String() string {
	var jsonc = json.Config{
		EscapeHTML:  false,
		SortMapKeys: true,
	}.Froze()
	var sb strings.Builder
	roots := make([]string, 0, len(c.Config))
	for r := range c.Config {
		roots = append(roots, r)
	}
	sort.Strings(roots)
	for _, root := range roots {
		sb.WriteString(strconv.Quote(root))
		sb.WriteString(" = ")
		json, _ := jsonc.Marshal(c.Config[root])
		sb.Write(json)
		sb.WriteString("\n")
	}
	return sb.String()
}

func (c *EMQX) Get(path string) (hocon.Value, bool) {
	return c.Config.Lookup(path)
}

func (c *EMQX) Find(path string) hocon.Value {
	v, ok := c.Config.Lookup(path)
	if ok {
		return v
	}
	return nil
}

func AsString(v hocon.Value) (string, error) {
	if v == nil {
		return "", nil
	}
	switch v.Type() {
	case hocon.StringType:
		return v.(hocon.StringValue).String(), nil
	case hocon.BooleanType:
		return strconv.FormatBool(bool(v.(hocon.Bool))), nil
	case hocon.IntegerType:
		return strconv.FormatInt(int64(v.(hocon.Int)), 10), nil
	}
	return "", fmt.Errorf("%T has no string representation", v)
}

func (c *EMQX) StripReadOnlyConfig() []string {
	root := c.Config
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
	object := c.Config
	keys := strings.Split(path, ".")
	if len(keys) > 0 {
		return stripRecursive(object, keys)
	}
	return false
}

func stripRecursive(object hocon.Object, keys []string) bool {
	key := keys[0]
	val, ok := object[key]
	if !ok {
		return false
	}
	if len(keys) == 1 {
		delete(object, key)
		return true
	}
	inner, ok := val.(hocon.Object)
	if ok && stripRecursive(inner, keys[1:]) {
		if len(inner) == 0 {
			delete(object, key)
		}
		return true
	}
	return false
}

func (c *EMQX) Replace(path string, with hocon.Value) {
	object := c.Config
	keys := strings.Split(path, ".")
	l := len(keys)
	if l > 0 {
		replaceRecursive(object, keys, with)
	}
}

func replaceRecursive(object hocon.Object, keys []string, with hocon.Value) {
	k := keys[0]
	v, ok := object[k]
	switch {
	case len(keys) == 1:
		object[k] = with
	case ok && v.Type() == hocon.ObjectType:
		inner, _ := v.(hocon.Object)
		replaceRecursive(inner, keys[1:], with)
	default:
		object[k] = hocon.Object{}
		replaceRecursive(object, keys, with)
	}
}

func (c *EMQX) GetNodeCookie() string {
	v, _ := c.Get("node.cookie")
	cookie, _ := AsString(v)
	return cookie
}

func (c *EMQX) GetDashboardPortMap() map[string]int {
	portMap := make(map[string]int)
	portMap["dashboard"] = 18083 // default port

	v, ok := c.Get("dashboard.listeners.http.bind")
	httpBind, err := AsString(v)
	if ok && err == nil {
		port := extractPortNumber(httpBind)
		if port > 0 {
			portMap["dashboard"] = port
		} else {
			// port = 0 means disable dashboard
			// delete default port
			delete(portMap, "dashboard")
		}
	}

	v, ok = c.Get("dashboard.listeners.https.bind")
	httpsBind, err := AsString(v)
	if ok && err == nil {
		port := extractPortNumber(httpsBind)
		if port > 0 {
			portMap["dashboard-https"] = port
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
	listeners, ok := c.Get("listeners")
	if ok && listeners != nil && listeners.Type() == hocon.ObjectType {
		for t, listener := range listeners.(hocon.Object) {
			if listener.Type() != hocon.ObjectType {
				continue
			}
			for name, lc := range listener.(hocon.Object) {
				lconf, ok := lc.(hocon.Object)
				if !ok || !isEnabled(lconf) {
					continue
				}
				bind, err := AsString(byDefault(lconf["bind"], hocon.String(":0")))
				if err != nil {
					continue
				}
				port := extractPortNumber(bind)
				protocol := corev1.ProtocolTCP
				if t == "quic" {
					protocol = corev1.ProtocolUDP
				}
				portList = append(portList, corev1.ServicePort{
					Name:       fmt.Sprintf("%s-%s", t, name),
					Protocol:   protocol,
					Port:       int32(port),
					TargetPort: intstr.FromInt(port),
				})
			}
		}
	}

	gateways, ok := c.Get("gateway")
	if ok && gateways != nil && gateways.Type() == hocon.ObjectType {
		for proto, gc := range gateways.(hocon.Object) {
			gateway, ok := gc.(hocon.Object)
			if !ok || !isEnabled(gateway) {
				continue
			}
			listeners := gateway["listeners"].(hocon.Object)
			for t, listener := range listeners {
				if listener.Type() != hocon.ObjectType {
					continue
				}
				for name, lc := range listener.(hocon.Object) {
					lconf, ok := lc.(hocon.Object)
					// Compatible with "enable" and "enabled"
					// the default value of them both is true
					if !ok || !isEnabled(lconf) {
						continue
					}
					bind, err := AsString(byDefault(lconf["bind"], hocon.String(":0")))
					if err != nil {
						continue
					}
					port := extractPortNumber(bind)
					protocol := corev1.ProtocolTCP
					if t == "udp" || t == "dtls" {
						protocol = corev1.ProtocolUDP
					}
					portList = append(portList, corev1.ServicePort{
						Name:       fmt.Sprintf("%s-%s-%s", proto, t, name),
						Protocol:   protocol,
						Port:       int32(port),
						TargetPort: intstr.FromInt(port),
					})
				}
			}
		}
	}

	sort.Slice(portList, func(i, j int) bool {
		return portList[i].Name < portList[j].Name
	})

	return portList
}

func extractPortNumber(bind string) int {
	if !strings.Contains(bind, ":") {
		// example: ":1883"
		bind = fmt.Sprintf(":%s", bind)
	}
	_, portString, _ := net.SplitHostPort(bind)
	port, err := strconv.ParseInt(portString, 10, 32)
	if err != nil {
		return -1
	}
	return int(port)
}

/* HOCON helper functions */

func isEnabled(conf hocon.Object) bool {
	// Compatible with "enable" and "enabled", default value of both is true.
	enabled := byDefault(conf["enable"], byDefault(conf["enabled"], hocon.Bool(true)))
	if enabled.Type() == hocon.BooleanType {
		return bool(enabled.(hocon.Bool))
	}
	return false
}

func byDefault(v hocon.Value, def hocon.Value) hocon.Value {
	if v == nil {
		return def
	}
	return v
}
