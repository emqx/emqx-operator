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

package v2alpha1

import (
	"net"
	"strconv"
	"strings"

	"github.com/gurkankaymak/hocon"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func GetDashboardServicePort(instance *EMQX) corev1.ServicePort {
	hoconConfig, _ := hocon.ParseString(instance.Spec.BootstrapConfig)
	dashboardPort := strings.Trim(hoconConfig.GetString("dashboard.listeners.http.bind"), `"`)

	_, strPort, err := net.SplitHostPort(dashboardPort)
	if err != nil {
		strPort = dashboardPort
	}
	intPort, _ := strconv.Atoi(strPort)

	return corev1.ServicePort{
		Name:       "dashboard-listeners-http-bind",
		Protocol:   corev1.ProtocolTCP,
		Port:       int32(intPort),
		TargetPort: intstr.FromInt(intPort),
	}
}

func MergeServicePorts(ports1, ports2 []corev1.ServicePort) []corev1.ServicePort {
	ports := append(ports1, ports2...)

	result := make([]corev1.ServicePort, 0, len(ports))
	temp := map[string]struct{}{}

	for _, item := range ports {
		if _, ok := temp[item.Name]; !ok {
			temp[item.Name] = struct{}{}
			result = append(result, item)
		}
	}

	return result
}
