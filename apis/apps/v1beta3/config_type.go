package v1beta3

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EmqxConfig map[string]string

func (config EmqxConfig) Default(emqx client.Object) {
	names := &Names{emqx}

	clusterConfig := make(map[string]string)
	clusterConfig["name"] = emqx.GetName()
	clusterConfig["log.to"] = "console"
	clusterConfig["cluster.discovery"] = "dns"
	clusterConfig["cluster.dns.type"] = "srv"
	clusterConfig["cluster.dns.app"] = emqx.GetName()
	clusterConfig["cluster.dns.name"] = fmt.Sprintf("%s.%s.svc.cluster.local", names.HeadlessSvc(), emqx.GetNamespace())
	clusterConfig["listener.tcp.internal"] = ""
	for k, v := range clusterConfig {
		if _, ok := config[k]; !ok {
			config[k] = v
		}
	}
}
