package v1beta1

import (
	"fmt"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

func generateEnv(emqx Emqx) []corev1.EnvVar {
	env := clusterEnvForK8S(emqx)

	str := strings.Split(emqx.GetImage(), ":")
	if len(str) > 1 {
		match, _ := regexp.MatchString("^[4-9].[4-9]+.[0-9]+$", str[1])
		if match {
			// 4.4.x uses dns clustering by default
			env = clusterEnvForDNS(emqx)
		}
	}

	emqxEnv := emqx.GetEnv()
	for _, e := range env {
		index := containsEnv(emqx.GetEnv(), e.Name)
		if index == -1 {
			emqxEnv = append(emqxEnv, e)
		}
	}
	return emqxEnv
}

func containsEnv(env []corev1.EnvVar, name string) int {
	for index, value := range env {
		if value.Name == name {
			return index
		}
	}
	return -1
}

func getEnvValueFromNameExisted(env []corev1.EnvVar, name string) string {
	var value string
	for _, item := range env {
		if item.Name == name {
			value = item.Value
			break
		}
	}
	return value
}

func clusterEnvForK8S(emqx Emqx) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "EMQX_NAME",
			Value: emqx.GetName(),
		},
		{
			Name:  "EMQX_CLUSTER__DISCOVERY",
			Value: "k8s",
		},
		{
			Name:  "EMQX_CLUSTER__K8S__APP_NAME",
			Value: emqx.GetName(),
		},
		{
			Name:  "EMQX_CLUSTER__K8S__SERVICE_NAME",
			Value: emqx.GetHeadlessServiceName(),
		},
		{
			Name:  "EMQX_CLUSTER__K8S__NAMESPACE",
			Value: emqx.GetNamespace(),
		},
		{
			Name:  "EMQX_CLUSTER__K8S__APISERVER",
			Value: "https://kubernetes.default.svc:443",
		},
		{
			Name:  "EMQX_CLUSTER__K8S__ADDRESS_TYPE",
			Value: "hostname",
		},
		{
			Name:  "EMQX_CLUSTER__K8S__SUFFIX",
			Value: "svc.cluster.local",
		},
	}
}

func clusterEnvForDNS(emqx Emqx) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "EMQX_NAME",
			Value: emqx.GetName(),
		},
		{
			Name:  "EMQX_CLUSTER__DISCOVERY",
			Value: "dns",
		},
		{
			Name:  "EMQX_CLUSTER__DNS__TYPE",
			Value: "srv",
		},
		{
			Name:  "EMQX_CLUSTER__DNS__APP",
			Value: emqx.GetName(),
		},
		{
			Name:  "EMQX_CLUSTER__DNS__NAME",
			Value: fmt.Sprintf("%s.%s.svc.cluster.local", emqx.GetHeadlessServiceName(), emqx.GetNamespace()),
		},
	}
}

func GenerateCommandForTelegrafReadinessProbe(env []corev1.EnvVar) []string {
	var managementId, managementSecret string
	if containsEnv(env, "EMQX_MANAGEMENT__DEFAULT_APPLICATION__ID") == -1 {
		managementId = "admin"
	} else {
		managementId = getEnvValueFromNameExisted(env, "EMQX_MANAGEMENT__DEFAULT_APPLICATION__ID")
	}

	if containsEnv(env, "EMQX_MANAGEMENT__DEFAULT_APPLICATION__SECRET") == -1 {
		managementSecret = "public"
	} else {
		managementSecret = "xx"
	}
	command := []string{
		"sh",
		"-c",
		fmt.Sprintf("curl -u %s:%s 'http://127.0.0.1:8081/api/v4/emqx_prometheus'", managementId, managementSecret),
	}
	return command
}
