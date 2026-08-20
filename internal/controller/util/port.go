package controller

import (
	"cmp"
	"fmt"
	"net"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

func ParseBindPort(bind string) (int32, error) {
	portString := bind
	if strings.Contains(bind, ":") {
		var err error
		_, portString, err = net.SplitHostPort(bind)
		if err != nil {
			return 0, err
		}
	}
	if portString == "" {
		return 0, fmt.Errorf("missing port")
	}
	port, err := strconv.ParseInt(portString, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("non-numeric port %q: %w", portString, err)
	}
	return int32(port), nil
}

func AppendMissingServicePorts(existing, additional []corev1.ServicePort) []corev1.ServicePort {
	result := append([]corev1.ServicePort(nil), existing...)
	names := make(map[string]struct{}, len(existing))
	for _, port := range existing {
		names[port.Name] = struct{}{}
	}
	for _, port := range additional {
		if _, exists := names[port.Name]; !exists {
			result = append(result, port)
		}
	}
	return result
}

func AppendMissingContainerPorts(existing, additional []corev1.ContainerPort) []corev1.ContainerPort {
	result := append([]corev1.ContainerPort(nil), existing...)
	names := make(map[string]struct{}, len(existing))
	for _, port := range existing {
		names[port.Name] = struct{}{}
	}
	for _, port := range additional {
		if _, exists := names[port.Name]; !exists {
			result = append(result, port)
		}
	}
	return result
}

func MapServicePortsToContainerPorts(ports []corev1.ServicePort) []corev1.ContainerPort {
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

func CompareServicePort(a corev1.ServicePort, b corev1.ServicePort) int {
	if c := cmp.Compare(a.Name, b.Name); c != 0 {
		return c
	}
	if c := cmp.Compare(a.Protocol, b.Protocol); c != 0 {
		return c
	}
	return cmp.Compare(a.Port, b.Port)
}
