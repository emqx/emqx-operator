package v1beta3

import (
	"fmt"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EmqxConfig map[string]string

type ConfigList struct {
	Items EmqxConfig
}

func (list *ConfigList) Default(emqx client.Object) {
	var once sync.Once
	once.Do(func() {
		if list.Items == nil {
			list.Items = make(EmqxConfig)
		}
	})

	names := &Names{emqx}

	clusterConfig := make(map[string]string)
	clusterConfig["cluster.discovery"] = "dns"
	clusterConfig["cluster.dns.type"] = "srv"
	clusterConfig["log.to"] = "both"
	clusterConfig["name"] = emqx.GetName()
	clusterConfig["cluster.dns.app"] = emqx.GetName()
	clusterConfig["cluster.dns.name"] = fmt.Sprintf("%s.%s.svc.cluster.local", names.HeadlessSvc(), emqx.GetNamespace())

	for k, v := range clusterConfig {
		if _, ok := list.Items[k]; !ok {
			list.Items[k] = v
		}
	}
}

func (list *ConfigList) FormatItems2Env() (ret []corev1.EnvVar) {
	ret = EmqxConfig2EnvVar(list.Items)

	return
}

func EmqxConfig2EnvVar(config EmqxConfig) (ret []corev1.EnvVar) {
	for k, v := range config {
		key := fmt.Sprintf("EMQX_%s", strings.ToUpper(strings.ReplaceAll(k, ".", "__")))
		ret = append(ret, corev1.EnvVar{Name: key, Value: v})
	}
	return
}
