# Use Telegraf to collect EMQX cluster logs


## Task target
- Learn how to collect EMQX cluster logs through Telegraf

## Deploy telegraf-operator

Telegraf is a plugin-driven agent that collects, processes, aggregates and writes metrics. It supports four categories of plugins, including input, output, aggregator, processor, and external. For more articles about Telegraf, please refer to: [telegraf](https://docs.influxdata.com/telegraf/v1.24), for the documentation of telegraf-operator, please refer to: [telegraf-operator](https://github.com/influxdata/telegraf-operator), execute the following command to deploy telegraf-operator

```
helm repo add influxdata https://helm.influxdata.com/
helm upgrade --install telegraf-operator influxdata/telegraf-operator
```
## Deploy EMQX cluster

Telegraf uses annotations to inject the sidecar of log collection into the Pod. In this article, the telegraf input plugin we use is tail. For the configuration of the tail plugin, please refer to: [tail plugin](https://github.com/influxdata/telegraf/blob/release-1.24/plugins/inputs/tail/README.md), and to use other input plugins, please refer to the documentation: [other input plugins](https://docs.influxdata.com/telegraf/v1.24/plugins/)
 

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
  name: emqx-ee
  annotations:
    telegraf.influxdata.com/inputs: |+
      [[inputs.tail]]
        files = ["/opt/emqx/log/emqx.log.[1-9]"]
        from_beginning = false
        max_undelivered_lines = 64
        character_encoding = "utf-8"
        data_format = "grok"
        grok_patterns = ['^%{TIMESTAMP_ISO8601:timestamp:ts-"2006-01-02T15:04:05.999999999-07:00"} \[%{LOGLEVEL:level}\] (?m)%{GREEDYDATA:messages}$']
spec:
  emqxTemplate:
    image: emqx/emqx-ee:4.4.8
```

Save the above content as: emqx-telegraf.yaml and deploy the EMQX cluster

```
kubectl apply -f emqx-telegraf.yaml 
```

- Check whether telegraf sidecar is injected into EMQX pod

```
kubectl get pods  -l  apps.emqx.io/instance=emqx-ee
```

The output is similar to:

```
NAME        READY   STATUS    RESTARTS   AGE
emqx-ee-0   3/3     Running   0          48m
emqx-ee-1   3/3     Running   0          48m
emqx-ee-2   3/3     Running   0          48m
```

Note: When the telegraf sidecar is injected into the EMQX pod, the number of containers in the EQMX pod will reach 3

