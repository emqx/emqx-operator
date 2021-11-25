package util

import (
	corev1 "k8s.io/api/core/v1"
)

func GenerateEnv(name, namespace string, env []corev1.EnvVar) []corev1.EnvVar {
	return MergeEnv(env, defaultEnv(name, namespace))
}

func MergeEnv(env1, env2 []corev1.EnvVar) []corev1.EnvVar {
	for index, value := range env2 {
		r := contains(env1, value.Name)
		if r == -1 {
			env1 = append(env1, env2[index])
		}
	}
	return env1
}

func contains(Env []corev1.EnvVar, Name string) int {
	for index, value := range Env {
		if value.Name == Name {
			return index
		}
	}
	return -1
}

func defaultEnv(name, namespace string) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "EMQX_NAME",
			Value: name,
		},
		{
			Name:  "EMQX_CLUSTER__DISCOVERY",
			Value: "k8s",
		},
		{
			Name:  "EMQX_CLUSTER__K8S__APP_NAME",
			Value: name,
		},
		{
			Name:  "EMQX_CLUSTER__K8S__SERVICE_NAME",
			Value: GenerateHeadelssServiceName(name),
		},
		{
			Name:  "EMQX_CLUSTER__K8S__NAMESPACE",
			Value: namespace,
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
