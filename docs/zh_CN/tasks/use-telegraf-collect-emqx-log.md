# 使用 Telegraf 收集 EMQX 集群日志

## 任务目标
- 学习如何通过 Telegraf 收集 EMQX 集群日志

## 部署 telegraf-operator

Telegraf 是一个插件驱动的代理，可以收集、处理、聚合和写入指标。 它支持四类插件，包括输入、输出、聚合器、处理器和外部。 更多关于 Telegraf 的文章可以参考：[telegraf](https://docs.influxdata.com/telegraf/v1.24/)，telegraf-operator 的文档可以参考：[telegraf-operator ](https://github.com/influxdata/telegraf-operator)，执行如下命令部署 telegraf-operator

```
helm repo add influxdata https://helm.influxdata.com/
helm upgrade --install telegraf-operator influxdata/telegraf-operator
```

## 部署 EMQX 集群

Telegraf 使用注解的方式为 Pod 注入日志采集的 sidecar，在本文中我们使用的 telegraf 输入插件为 tail，tail 插件的配置可以参考：[tail 插件](https://github.com/influxdata/telegraf/blob/release-1.24/plugins/inputs/tail/README.md)，使用其他的输入插件可以参考文档：[其他输入插件](https://docs.influxdata.com/telegraf/v1.24/plugins/)

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

将上述内容保存为：emqx-telegraf.yaml 并部署 EMQX 集群

```
kubectl apply -f emqx-telegraf.yaml 
```

- 检查 telegraf sidecar 是否注入到 EMQX pod 中

```
kubectl get pods  -l  apps.emqx.io/instance=emqx-ee
```

输出类似于： 

```
NAME        READY   STATUS    RESTARTS   AGE
emqx-ee-0   3/3     Running   0          48m
emqx-ee-1   3/3     Running   0          48m
emqx-ee-2   3/3     Running   0          48m
```
备注：当 telegraf sidecar 注入到 EMQX  pod 中后，EQMX pod 中的容器数量会达到3个