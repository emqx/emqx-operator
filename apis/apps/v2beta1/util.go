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

package v2beta1

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	emperror "emperror.dev/errors"
	// "github.com/gurkankaymak/hocon"
	hocon "github.com/rory-z/go-hocon"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func IsExistReplicant(instance *EMQX) bool {
	return instance.Spec.ReplicantTemplate != nil && instance.Spec.ReplicantTemplate.Spec.Replicas != nil && *instance.Spec.ReplicantTemplate.Spec.Replicas > 0
}

func DefaultLabels(instance *EMQX) map[string]string {
	labels := map[string]string{}
	labels[LabelsInstanceKey] = instance.Name
	labels[LabelsManagedByKey] = "emqx-operator"
	return labels
}

func DefaultCoreLabels(instance *EMQX) map[string]string {
	return CloneAndAddLabel(
		DefaultLabels(instance),
		LabelsDBRoleKey,
		"core",
	)
}

func DefaultReplicantLabels(instance *EMQX) map[string]string {
	return CloneAndAddLabel(
		DefaultLabels(instance),
		LabelsDBRoleKey,
		"replicant",
	)
}

func CloneAndMergeMap(dst map[string]string, src ...map[string]string) map[string]string {
	new := map[string]string{}

	for key, value := range dst {
		new[key] = value
	}

	for _, m := range src {
		for key, value := range m {
			if _, ok := new[key]; !ok {
				new[key] = value
			}
		}
	}

	return new
}

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

func GetDashboardPortMap(hoconString string) (map[string]int32, error) {
	portMap := make(map[string]int32)
	portMap["dashboard"] = 18083 // default port

	hoconConfig, err := hocon.ParseString(hoconString)
	if err != nil {
		return nil, emperror.Wrapf(err, "failed to parse %s", hoconString)
	}

	if dashboardPort := strings.Trim(hoconConfig.GetString("dashboard.listeners.http.bind"), `"`); dashboardPort != "" {
		if !strings.Contains(dashboardPort, ":") {
			// example: ":18083"
			dashboardPort = fmt.Sprintf(":%s", dashboardPort)
		}
		_, strPort, _ := net.SplitHostPort(dashboardPort)
		if port, _ := strconv.Atoi(strPort); port != 0 {
			portMap["dashboard"] = int32(port)
		} else {
			// port = 0 means disable dashboard
			// delete default port
			delete(portMap, "dashboard")
		}
	}

	if dashboardHttpsPort := strings.Trim(hoconConfig.GetString("dashboard.listeners.https.bind"), `"`); dashboardHttpsPort != "" {
		if !strings.Contains(dashboardHttpsPort, ":") {
			// example: ":18084"
			dashboardHttpsPort = fmt.Sprintf(":%s", dashboardHttpsPort)
		}
		_, strPort, _ := net.SplitHostPort(dashboardHttpsPort)
		if port, _ := strconv.Atoi(strPort); port != 0 {
			portMap["dashboard-https"] = int32(port)
		} else {
			// port = 0 means disable dashboard
			// delete default port
			delete(portMap, "dashboard-https")
		}
	}

	return portMap, nil
}

func GetDashboardServicePort(hoconString string) ([]corev1.ServicePort, error) {
	dashboardSvcPortList := []corev1.ServicePort{}
	portMap, err := GetDashboardPortMap(hoconString)
	if err != nil {
		return nil, emperror.Wrapf(err, "failed to get dashboard port map")
	}

	for name, port := range portMap {
		dashboardSvcPortList = append(dashboardSvcPortList, corev1.ServicePort{
			Name:       name,
			Protocol:   corev1.ProtocolTCP,
			Port:       port,
			TargetPort: intstr.FromInt(int(port)),
		})
	}

	return dashboardSvcPortList, nil
}

func GetListenersServicePorts(hoconString string) ([]corev1.ServicePort, error) {
	hoconConfig, err := hocon.ParseString(hoconString)
	if err != nil {
		return nil, emperror.Wrapf(err, "failed to parse %s", hoconString)
	}
	svcPorts := []corev1.ServicePort{}

	// Get listeners.tcp.default.bind
	for t, listener := range hoconConfig.GetObject("listeners") {
		if listener.Type() != hocon.ObjectType {
			continue
		}

		listenerConfig, err := hocon.ParseString(listener.String())
		if err != nil {
			continue
		}

		configs := listenerConfig.GetRoot()
		if configs.Type() != hocon.ObjectType {
			continue
		}

		for name, config := range configs.(hocon.Object) {
			// Wait fix this issue: https://github.com/gurkankaymak/hocon/issues/39
			// c, err := hocon.ParseString(config.String())
			obj := config.(hocon.Object)
			cutConfig := hocon.Object{}
			if v, ok := obj["enable"]; ok {
				cutConfig["enable"] = v
			}
			if v, ok := obj["enabled"]; ok {
				cutConfig["enabled"] = v
			}
			if v, ok := obj["bind"]; ok {
				cutConfig["bind"] = v
			}
			c, err := hocon.ParseString(cutConfig.String())
			if err != nil {
				return nil, emperror.Wrapf(err, "failed to parse %s", config.String())
			}
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

	// Get gateway.lwm2m.listeners.udp.default.bind
	for proto, gateway := range hoconConfig.GetObject("gateway") {
		c, _ := hocon.ParseString(gateway.String())
		// Compatible with "enable" and "enabled"
		// the default value of them both is true
		if c.GetString("enable") == "false" || c.GetString("enabled") == "false" {
			continue
		}
		for t, listener := range c.GetObject("listeners") {
			if listener.Type() != hocon.ObjectType {
				continue
			}

			listenerConfig, err := hocon.ParseString(listener.String())
			if err != nil {
				continue
			}

			configs := listenerConfig.GetRoot()
			if configs.Type() != hocon.ObjectType {
				continue
			}

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

func TransServicePortsToContainerPorts(ports []corev1.ServicePort) []corev1.ContainerPort {
	result := make([]corev1.ContainerPort, 0, len(ports))
	for _, item := range ports {
		result = append(result, corev1.ContainerPort{
			Name:          item.Name,
			ContainerPort: item.Port,
			Protocol:      item.Protocol,
		})
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
