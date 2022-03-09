package v1beta2

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Environments struct {
	Items []corev1.EnvVar
}

func (e *Environments) Merge(envs []corev1.EnvVar) {
	for _, env := range envs {
		_, index := e.Lookup(env.Name)
		if index == -1 {
			e.Items = append(e.Items, env)
		} else {
			e.Items[index].Value = env.Value
		}
	}
}

func (e *Environments) Lookup(name string) (*corev1.EnvVar, int) {
	for index, env := range e.Items {
		if env.Name == name {
			return &env, index
		}
	}
	return nil, -1
}

func (e *Environments) ClusterForDNS(emqx client.Object) {
	names := &Names{emqx}
	e.Items = []corev1.EnvVar{
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
}

func (e *Environments) ClusterForK8S(emqx client.Object) {
	names := &Names{emqx}
	e.Items = []corev1.EnvVar{
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
}
