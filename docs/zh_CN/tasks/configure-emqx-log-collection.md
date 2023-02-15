# 配置 EMQX 日志采集

## 任务目标

- 如何通过 Telegraf 收集 EMQX 集群日志，并且输出到容器的标准输出

## 部署 telegraf-operator

Telegraf 是 InfluxData 开发的一个开源数据采集代理，可以收集、处理、聚合和写入指标。 它支持四类插件，包括输入，输出，聚合器，处理器。 更多关于 Telegraf 的文章可以参考：[telegraf](https://docs.influxdata.com/telegraf/v1.24/) ，telegraf-operator 的文档可以参考： [telegraf-operator](https://github.com/influxdata/telegraf-operator)

执行如下命令部署 telegraf-operator

```shell
helm repo add influxdata https://helm.influxdata.com/
helm upgrade --install telegraf-operator influxdata/telegraf-operator
```

## 全局配置 - classes

全局配置通过 secret 挂载，指定 class 名称为 logs

`agent` 用来配置 telegraf agent，详细定义参考：[telegraf agent](https://github.com/influxdata/telegraf/blob/master/docs/CONFIGURATION.md#agent)

`inputs.tail` 用来配置输入的 tail 插件，详细定义参考：[tail](https://github.com/influxdata/telegraf/blob/master/plugins/inputs/tail/README.md)

`outputs.file` 用来配置输出的 file 插件，详细定义参考：[file](https://github.com/influxdata/telegraf/blob/master/plugins/inputs/tail/README.md)

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: telegraf-operator-classes
  namespace: default
stringData:
  logs: |+
    [agent]
      interval = "60s"
      flush_jitter = "5s"
      flush_interval = "15s"
      debug = true
      quiet = false
      metric_batch_size = 128
      metric_buffer_limit = 256

    [[inputs.tail]]
      files = ["/opt/emqx/log/emqx.log.[1-9]"]
      from_beginning = false
      max_undelivered_lines = 64
      character_encoding = "utf-8"
      data_format = "grok"
      grok_patterns = ['^%{TIMESTAMP_ISO8601:timestamp:ts-"2006-01-02T15:04:05.999999999-07:00"} \[%{LOGLEVEL:level}\] (?m)%{GREEDYDATA:messages}$']
      [inputs.tail.tags]
        collection = "log"

    [[outputs.file]]
      files = ["stdout"]
```

将上述内容保存为：classes.yaml

- 创建 secret

```shell
kubectl apply -f classes.yaml
```

- 检查创建的 secret

```shell
kubectl get secret telegraf-operator-classes
```

输出类似于：

```shell
NAME                        TYPE     DATA   AGE
telegraf-operator-classes   Opaque   1      11h
```

## 部署 EMQX 集群

Telegraf 使用 annotations 的方式为 Pod 注入日志采集的 sidecar ，详细的 annotations 定义参考文档：[telegraf annotations](https://github.com/influxdata/telegraf-operator#pod-level-annotations)

:::: tabs type:card
::: tab v2alpha1

`telegraf.influxdata.com/internal` 设置为 false， 表示不收集 telegraf agent 自己的指标

`telegraf.influxdata.com/volume-mounts` 设置日志的挂载路径

`telegraf.influxdata.com/class` logs 引用上面指定的 class 的名称

`spec.bootstrapConfig` 配置输出日志到文件，并且日志级别为 debug

`spec.coreTemplate.spec.extraVolumes` 和 `spec.coreTemplate.spec.extraVolumeMounts` 配置日志挂载


```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
  name: emqx
spec:
  image: "emqx/emqx-enterprise:5.0.0"
  bootstrapConfig: |
    log {
      file_handlers {
        my_debug_log {
          enable = true
          level = debug
          file = "log/emqx.log"
          rotation {
            enable = true
            count = 10
          }
        }
      }
    }
  coreTemplate:
    metadata:
      name: emqx-core
      labels:
        apps.emqx.io/instance: emqx
        apps.emqx.io/db-role: core
      annotations:
        telegraf.influxdata.com/class: "logs"
        telegraf.influxdata.com/internal: "false"
        telegraf.influxdata.com/volume-mounts: "{\"log-volume\":\"/opt/emqx/log\"}"
    spec:
      replicas: 1
      extraVolumes:
        - name: log-volume
          emptyDir: {}
      extraVolumeMounts:
        - name: log-volume
          mountPath: /opt/emqx/log
```

:::
::: tab v1beta4

`telegraf.influxdata.com/internal` 设置为 false， 表示不收集 telegraf agent 自己的指标

`telegraf.influxdata.com/volume-mounts` 设置日志的挂载路径

`telegraf.influxdata.com/class` logs 引用上面指定的 class 的名称

`spec.template.spec.emqxContainer.emqxConfig` 配置输出日志到文件，并且日志级别为 debug

`spec.template.spec.volumes` 和 `spec.template.spec.emqxContainer.volumeMounts` 配置日志挂载

```yaml
apiVersion: apps.emqx.io/v1beta4
kind: EmqxEnterprise
metadata:
  name: emqx-ee
  annotations:
    telegraf.influxdata.com/internal: "false"
    telegraf.influxdata.com/volume-mounts: "{\"log-volume\":\"/opt/emqx/log\"}"
    telegraf.influxdata.com/class: "logs"
spec:
  template:
    spec:
      emqxContainer:
        image:
          repository: emqx/emqx-ee
          version: 4.4.14
        emqxConfig:
          log.level: debug
          log.to: file
        volumeMounts:
        - name: log-volume
          mountPath: /opt/emqx/log
      volumes:
        - name: log-volume
          emptyDir: {}
```

:::
::::

将上述内容保存为：emqx-telegraf.yaml

- 部署 EMQX 集群

```shell
kubectl apply -f emqx-telegraf.yaml
```

- 检查 EMQX 集群状态

:::: tabs type:card
::: tab v2alpha1


```shell
kubectl get pods  -l  apps.emqx.io/instance=emqx
```

输出类似于：

```shell
NAME                             READY   STATUS    RESTARTS   AGE
emqx-core-0                      2/2     Running   0          54s
emqx-replicant-c868c79cd-9m5rw   1/1     Running   0          41s
emqx-replicant-c868c79cd-qv8mk   1/1     Running   0          41s
emqx-replicant-c868c79cd-z8bvj   1/1     Running   0          41s
```

**备注：** 当 telegraf sidecar 注入到 EMQX core pod 中后，EQMX core pod 中的容器数量会达到2个


:::
::: tab v1beta4

```shell
kubectl get pods  -l  apps.emqx.io/instance=emqx-ee
```

输出类似于：

```shell
NAME        READY   STATUS    RESTARTS   AGE
emqx-ee-0   3/3     Running   0          8m37s
emqx-ee-1   3/3     Running   0          8m37s
emqx-ee-2   3/3     Running   0          8m37s
```

**备注：** 当 telegraf sidecar 注入到 EMQX  pod 中后，EQMX pod 中的容器数量会达到3个

:::
::::

- 查看收集到的日志

:::: tabs type:card
::: tab v2alpha1

```shell
kubectl logs -f emqx-core-0 -c telegraf
```

:::
::: tab v1beta4

```shell
kubectl logs -f emqx-ee-0 -c telegraf
```


:::
::::

采集到的 EMQX log 都输出到了标准输出
