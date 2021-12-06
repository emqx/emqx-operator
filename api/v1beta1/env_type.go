package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
)

func (emqx *EmqxBroker) GetEnv() []corev1.EnvVar {
	return generateEnv(emqx, emqx.Spec.Env)
}

func (emqx *EmqxEnterprise) GetEnv() []corev1.EnvVar {
	return generateEnv(emqx, emqx.Spec.Env)
}

func generateEnv(emqx Emqx, env []corev1.EnvVar) []corev1.EnvVar {
	contains := func(Env []corev1.EnvVar, Name string) int {
		for index, value := range Env {
			if value.Name == Name {
				return index
			}
		}
		return -1
	}

	e := defaultEnv(emqx)
	for _, value := range e {
		r := contains(env, value.Name)
		if r == -1 {
			env = append(env, value)
		}
	}
	return env

}

func defaultEnv(emqx Emqx) []corev1.EnvVar {
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
