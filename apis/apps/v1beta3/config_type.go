package v1beta3

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EmqxConfig map[string]string

func (config EmqxConfig) Default(emqx client.Object) {
	names := &Names{emqx}

	clusterConfig := make(map[string]string)
	clusterConfig["log.to"] = "both"
	clusterConfig["node.name"] = emqx.GetName()
	clusterConfig["listener.tcp.external"] = "1883"
	clusterConfig["listener.ssl.external"] = "8883"
	clusterConfig["listener.ws.external"] = "8083"
	clusterConfig["listener.wss.external"] = "8084"
	clusterConfig["cluster.discovery"] = "dns"
	clusterConfig["cluster.dns.type"] = "srv"
	clusterConfig["cluster.dns.app"] = emqx.GetName()
	clusterConfig["cluster.dns.name"] = fmt.Sprintf("%s.%s.svc.cluster.local", names.HeadlessSvc(), emqx.GetNamespace())

	for k, v := range clusterConfig {
		if _, ok := config[k]; !ok {
			config[k] = v
		}
	}
}
