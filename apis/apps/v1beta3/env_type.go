package v1beta3

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EnvList struct {
	Items []corev1.EnvVar
}

func (list *EnvList) Default(emqx client.Object) {
	defaultEnvs := []corev1.EnvVar{
		{
			Name:  "EMQX_LOG__TO",
			Value: "both",
		},
		{
			Name:  "EMQX_NAME",
			Value: emqx.GetName(),
		},
	}
	list.Append(defaultEnvs)
}

func (list *EnvList) Append(envs []corev1.EnvVar) {
	for _, env := range envs {
		_, index := list.Lookup(env.Name)
		if index == -1 {
			list.Items = append(list.Items, env)
		}
	}
}

func (list *EnvList) Overwrite(envs []corev1.EnvVar) {
	for _, env := range envs {
		_, index := list.Lookup(env.Name)
		if index == -1 {
			list.Items = append(list.Items, env)
		} else {
			list.Items[index].Value = env.Value
		}
	}
}

func (list *EnvList) Lookup(name string) (*corev1.EnvVar, int) {
	for index, env := range list.Items {
		if env.Name == name {
			return &env, index
		}
	}
	return nil, -1
}

func (list *EnvList) ClusterForDNS(emqx client.Object) {
	names := &Names{emqx}
	clusterEnvs := []corev1.EnvVar{
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
			Value: fmt.Sprintf("%s.%s.svc.cluster.local", names.HeadlessSvc(), emqx.GetNamespace()),
		},
	}
	list.Default(emqx)
	list.Append(clusterEnvs)
}

func (list *EnvList) ClusterForK8S(emqx client.Object) {
	names := &Names{emqx}
	clusterEnvs := []corev1.EnvVar{
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
			Value: names.HeadlessSvc(),
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
	list.Default(emqx)
	list.Append(clusterEnvs)
}
