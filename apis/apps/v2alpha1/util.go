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

	emperror "emperror.dev/errors"
	// "github.com/gurkankaymak/hocon"
	hocon "github.com/rory-z/go-hocon"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func GetDashboardServicePort(instance *EMQX) (*corev1.ServicePort, error) {
	hoconConfig, err := hocon.ParseString(instance.Spec.BootstrapConfig)
	if err != nil {
		return nil, emperror.Wrapf(err, "failed to parse %s", instance.Spec.BootstrapConfig)
	}
	dashboardPort := strings.Trim(hoconConfig.GetString("dashboard.listeners.http.bind"), `"`)
	if dashboardPort == "" {
		return nil, emperror.Errorf("failed to get dashboard.listeners.http.bind in %s", hoconConfig.String())
	}

	_, strPort, err := net.SplitHostPort(dashboardPort)
	if err != nil {
		strPort = dashboardPort
	}
	port, _ := strconv.Atoi(strPort)

	return &corev1.ServicePort{
		Name:       "dashboard-listeners-http-bind",
		Protocol:   corev1.ProtocolTCP,
		Port:       int32(port),
		TargetPort: intstr.FromInt(port),
	}, nil
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
