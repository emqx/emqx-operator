/*
Copyright 2021.

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

package v2alpha2

import (
	"fmt"
	"net"
	"strings"

	emperror "emperror.dev/errors"
	// "github.com/gurkankaymak/hocon"
	hocon "github.com/rory-z/go-hocon"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Clones the given selector and returns a new selector with the given key and value added.
// Returns the given selector, if labelKey is empty.
func CloneSelectorAndAddLabel(selector *metav1.LabelSelector, labelKey, labelValue string) *metav1.LabelSelector {
	if labelKey == "" {
		// Don't need to add a label.
		return selector
	}

	// Clone.
	newSelector := new(metav1.LabelSelector)

	// TODO(madhusudancs): Check if you can use deepCopy_extensions_LabelSelector here.
	newSelector.MatchLabels = make(map[string]string)
	if selector.MatchLabels != nil {
		for key, val := range selector.MatchLabels {
			newSelector.MatchLabels[key] = val
		}
	}
	newSelector.MatchLabels[labelKey] = labelValue

	if selector.MatchExpressions != nil {
		newMExps := make([]metav1.LabelSelectorRequirement, len(selector.MatchExpressions))
		for i, me := range selector.MatchExpressions {
			newMExps[i].Key = me.Key
			newMExps[i].Operator = me.Operator
			if me.Values != nil {
				newMExps[i].Values = make([]string, len(me.Values))
				copy(newMExps[i].Values, me.Values)
			} else {
				newMExps[i].Values = nil
			}
		}
		newSelector.MatchExpressions = newMExps
	} else {
		newSelector.MatchExpressions = nil
	}

	return newSelector
}

// AddLabelToSelector returns a selector with the given key and value added to the given selector's MatchLabels.
func AddLabelToSelector(selector *metav1.LabelSelector, labelKey, labelValue string) *metav1.LabelSelector {
	if labelKey == "" {
		// Don't need to add a label.
		return selector
	}
	if selector.MatchLabels == nil {
		selector.MatchLabels = make(map[string]string)
	}
	selector.MatchLabels[labelKey] = labelValue
	return selector
}

// Clones the given map and returns a new map with the given key and value added.
// Returns the given map, if labelKey is empty.
func CloneAndAddLabel(labels map[string]string, labelKey, labelValue string) map[string]string {
	if labelKey == "" {
		// Don't need to add a label.
		return labels
	}
	// Clone.
	newLabels := map[string]string{}
	for key, value := range labels {
		newLabels[key] = value
	}
	newLabels[labelKey] = labelValue
	return newLabels
}

// CloneAndRemoveLabel clones the given map and returns a new map with the given key removed.
// Returns the given map, if labelKey is empty.
func CloneAndRemoveLabel(labels map[string]string, labelKey string) map[string]string {
	if labelKey == "" {
		// Don't need to add a label.
		return labels
	}
	// Clone.
	newLabels := map[string]string{}
	for key, value := range labels {
		newLabels[key] = value
	}
	delete(newLabels, labelKey)
	return newLabels
}

// AddLabel returns a map with the given key and value added to the given map.
func AddLabel(labels map[string]string, labelKey, labelValue string) map[string]string {
	if labelKey == "" {
		// Don't need to add a label.
		return labels
	}
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[labelKey] = labelValue
	return labels
}

func GetDashboardServicePort(hoconString string) (*corev1.ServicePort, error) {
	hoconConfig, err := hocon.ParseString(hoconString)
	if err != nil {
		return nil, emperror.Wrapf(err, "failed to parse %s", hoconString)
	}
	dashboardPort := strings.Trim(hoconConfig.GetString("dashboard.listeners.http.bind"), `"`)
	if dashboardPort == "" {
		return nil, emperror.Errorf("failed to get dashboard.listeners.http.bind in %s", hoconConfig.String())
	}
	if !strings.Contains(dashboardPort, ":") {
		// example: ":18083"
		dashboardPort = fmt.Sprintf(":%s", dashboardPort)
	}
	_, strPort, err := net.SplitHostPort(dashboardPort)
	if err != nil {
		return nil, emperror.Wrapf(err, "failed to split %s", dashboardPort)
	}
	intStrValue := intstr.Parse(strPort)

	return &corev1.ServicePort{
		Name:       "dashboard",
		Protocol:   corev1.ProtocolTCP,
		Port:       int32(intStrValue.IntValue()),
		TargetPort: intStrValue,
	}, nil
}

func GetListenersServicePorts(hoconString string) ([]corev1.ServicePort, error) {
	hoconConfig, err := hocon.ParseString(hoconString)
	if err != nil {
		return nil, emperror.Wrapf(err, "failed to parse %s", hoconString)
	}
	svcPorts := []corev1.ServicePort{}

	// Get listeners.tcp.default.bind
	for t, listener := range hoconConfig.GetObject("listeners") {
		listenerConfig, _ := hocon.ParseString(listener.String())

		configs := listenerConfig.GetRoot()
		if configs.Type() == hocon.ObjectType {
			for name, config := range configs.(hocon.Object) {
				c, _ := hocon.ParseString(config.String())
				// Compatible with "enable" and "enabled"
				// the default value of them both is true
				if c.GetString("enable") == "false" || c.GetString("enabled") == "false" {
					continue
				}
				bind := strings.Trim(c.GetString("bind"), `"`)
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

				svcPorts = append(svcPorts, corev1.ServicePort{
					Name:       fmt.Sprintf("%s-%s", t, name),
					Protocol:   protocol,
					Port:       int32(intStrValue.IntValue()),
					TargetPort: intStrValue,
				})
			}
		}
	}

	// Get gateway.lwm2m.listeners.udp.default.bind
	for proto, gateway := range hoconConfig.GetObject("gateway") {
		c, _ := hocon.ParseString(gateway.String())
		if c.GetString("enable") == "" || c.GetString("enable") == "true" {
			for t, listener := range c.GetObject("listeners") {
				c, _ := hocon.ParseString(listener.String())
				for name, config := range c.GetRoot().(hocon.Object) {
					c, _ := hocon.ParseString(config.String())
					// Compatible with "enable" and "enabled"
					// the default value of them both is true
					if c.GetString("enable") == "false" || c.GetString("enabled") == "false" {
						continue
					}
					bind := strings.Trim(c.GetString("bind"), `"`)
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

					svcPorts = append(svcPorts, corev1.ServicePort{
						Name:       fmt.Sprintf("%s-%s-%s", proto, t, name),
						Protocol:   protocol,
						Port:       int32(intStrValue.IntValue()),
						TargetPort: intStrValue,
					})
				}
			}
		}
	}

	return svcPorts, nil
}

func MergeServicePorts(ports1, ports2 []corev1.ServicePort) []corev1.ServicePort {
	ports := append(ports1, ports2...)

	result := make([]corev1.ServicePort, 0, len(ports))
	tempName := map[string]struct{}{}
	tempPort := map[int32]struct{}{}

	for _, item := range ports {
		_, nameOK := tempName[item.Name]
		_, portOK := tempPort[item.Port]

		if !nameOK && !portOK {
			tempName[item.Name] = struct{}{}
			tempPort[item.Port] = struct{}{}
			result = append(result, item)
		}
	}

	return result
}

func MergeContainerPorts(ports1, ports2 []corev1.ContainerPort) []corev1.ContainerPort {
	ports := append(ports1, ports2...)

	result := make([]corev1.ContainerPort, 0, len(ports))
	tempName := map[string]struct{}{}
	tempPort := map[int32]struct{}{}

	for _, item := range ports {
		_, nameOK := tempName[item.Name]
		_, portOK := tempPort[item.ContainerPort]

		if !nameOK && !portOK {
			tempName[item.Name] = struct{}{}
			tempPort[item.ContainerPort] = struct{}{}
			result = append(result, item)
		}
	}

	return result
}

func mergeMap(dst, src map[string]string) map[string]string {
	if dst == nil {
		dst = make(map[string]string)
	}

	for key, value := range src {
		if _, ok := dst[key]; !ok {
			dst[key] = value
		}
	}
	return dst
}
