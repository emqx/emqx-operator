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
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/lithammer/dedent"
	"github.com/rory-z/go-hocon"
	corev1 "k8s.io/api/core/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

type Conf struct {
	config *hocon.Config
}

func EMQXConf(config string) (*Conf, error) {
	c := &Conf{}
	err := c.LoadEMQXConf(config)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Conf) LoadEMQXConf(config string) error {
	hoconConfig, err := hocon.ParseString(config)
	if err != nil {
		return err
	}
	c.config = hoconConfig
	return nil
}

func (c *Conf) Copy() *Conf {
	root := hocon.Object{}
	return &Conf{
		// Should deep-copy `c.config`.
		config: root.ToConfig().WithFallback(c.config),
	}
}

func (c *Conf) Print() string {
	return c.config.String()
}

func (c *Conf) StripReadOnlyConfig() []string {
	root := c.config.GetRoot().(hocon.Object)
	stripped := []string{}
	for _, key := range []string{
		"node",
		"cluster",
		"dashboard",
		"rpc",
		"durable_sessions",
		"durable_storage",
	} {
		if _, ok := root[key]; ok {
			stripped = append(stripped, key)
			delete(root, key)
		}
	}
	return stripped
}

func (c *Conf) GetNodeCookie() string {
	return toString(byDefault(c.config.Get("node.cookie"), ""))
}

func (c *Conf) IsDSEnabled() bool {
	flag := c.config.Get("durable_sessions.enable")
	if isTrue(byDefault(flag, false)) {
		return true
	}
	zones := c.config.GetObject("zones")
	if zones == nil {
		return false
	}
	for zone := range zones {
		flag = c.config.Get("zones." + zone + ".durable_sessions.enable")
		if isTrue(byDefault(flag, false)) {
			return true
		}
	}
	return false
}

func (c *Conf) GetDashboardPortMap() map[string]int {
	portMap := make(map[string]int)
	portMap["dashboard"] = 18083 // default port

	httpBind := byDefault(c.config.Get("dashboard.listeners.http.bind"), "")
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

	httpsBind := byDefault(c.config.Get("dashboard.listeners.https.bind"), "")
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

func (c *Conf) GetDashboardServicePort() []corev1.ServicePort {
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

func (c *Conf) GetListenersServicePorts() []corev1.ServicePort {
	portList := []corev1.ServicePort{}

	// Should be non-empty, see `mergeDefaultConfig`
	for t, listener := range c.config.GetObject("listeners") {
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
	for proto, gc := range c.config.GetObject("gateway") {
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

func MergeDefaults(config string) string {
	template := dedent.Dedent(`
	# emqx-operator default config
	listeners.tcp.default.bind = 1883
	listeners.ssl.default.bind = 8883
	listeners.ws.default.bind  = 8083
	listeners.wss.default.bind = 8084

	# user config
	%s
	`)
	return fmt.Sprintf(template, config)
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
