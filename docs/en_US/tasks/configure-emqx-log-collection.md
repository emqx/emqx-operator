# Collect EMQX Logs in Kubernetes

## Task target

How to collect EMQX cluster logs through Telegraf and export them to the standard output of the container.

## Deploy Telegraf Operator

Telegraf is a server-based agent for collecting and sending all metrics and events from databases, systems, and IoT sensors. It supports four types of plugins, including input, output, aggregator and processor. More articles about Telegraf can be found at [telegraf](https://docs.influxdata.com/telegraf/v1.24/), The documentation for telegraf-operator can be found in [telegraf-operator](https://github.com/influxdata/telegraf-operator).

Execute the following command to deploy telegraf-operator

```shell
helm repo add influxdata https://helm.influxdata.com/
helm upgrade --install telegraf-operator influxdata/telegraf-operator
```

## Global Configuration - `classes`

The global configuration is mounted via secret, specifying the class name as logs, where

- `agent` is to configure telegraf agent, refer to the detailed definition: [telegraf agent](https://github.com/influxdata/telegraf/blob/master/docs/CONFIGURATION.md#agent)
- `inputs.tail` is the tail plug-in used to configure the input and is defined in detail in [tail](https://github.com/influxdata/telegraf/blob/master/plugins/inputs/tail/README.md)
- `outputs.file` is a file plug-in used to configure the output is defined in detail in [file](https://github.com/influxdata/telegraf/blob/master/plugins/inputs/tail/README.md)

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

Save the above as `classes.yaml`

- Create secret

```shell
kubectl apply -f classes.yaml
```

- Check the created secret

```shell
kubectl get secret telegraf-operator-classes
```

The output is similar to:

```shell
NAME                        TYPE     DATA   AGE
telegraf-operator-classes   Opaque   1      11h
```

## Deploy EMQX Cluster

Telegraf uses annotations to inject sidecar for Pod log collection, for a detailed definition of annotations refer to the documentation: [telegraf annotations](https://github.com/influxdata/telegraf-operator#pod-level-annotations)

Here are the relevant configurations for EMQX Custom Resource. You can choose the corresponding APIVersion based on the version of EMQX you wish to deploy. For specific compatibility relationships, please refer to [EMQX Operator Compatibility](../README.md):

:::: tabs type:card
::: tab v2alpha1

`telegraf.influxdata.com/internal` Set to false to not collect the telegraf agent's own metrics

`telegraf.influxdata.com/volume-mounts` Set the mount path of the log

`telegraf.influxdata.com/class` logs reference the name of the class specified above

`spec.bootstrapConfig` Configure the output log to a file and the log level to debug

`spec.coreTemplate.spec.extraVolumes` and `spec.coreTemplate.spec.extraVolumeMounts` Configuring Log Mounts

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
  name: emqx
  annotations:
    telegraf.influxdata.com/class: "logs"
    telegraf.influxdata.com/internal: "false"
    telegraf.influxdata.com/volume-mounts: "{\"log-volume\":\"/opt/emqx/log\"}"
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
    spec:
      extraVolumes:
        - name: log-volume
          emptyDir: {}
      extraVolumeMounts:
        - name: log-volume
          mountPath: /opt/emqx/log
  replicantTemplate:
    spec:
      extraVolumes:
        - name: log-volume
          emptyDir: {}
      extraVolumeMounts:
        - name: log-volume
          mountPath: /opt/emqx/log
```

Save the above content as `emqx.yaml` and execute the following command to deploy the EMQX cluster:

```bash
$ kubectl apply -f emqx.yaml

emqx.apps.emqx.io/emqx created
```

Check the status of the EMQX cluster and make sure that `STATUS` is `Running`, which may take some time to wait for the EMQX cluster to be ready.

```bash
$ kubectl get emqx emqx

NAME   IMAGE      STATUS    AGE
emqx   emqx:5.0   Running   10m
```

:::
::: tab v1beta4

`telegraf.influxdata.com/internal` Set to false to not collect the telegraf agent's own metrics

`telegraf.influxdata.com/volume-mounts` Set the mount path of the log

`telegraf.influxdata.com/class` logs references the name of the class specified above

`spec.template.spec.emqxContainer.emqxConfig` Configure the output log to a file and the log level to debug

`spec.template.spec.volumes` and `spec.template.spec.emqxContainer.volumeMounts` Configure Log volume

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

Save the above content as `emqx.yaml` and execute the following command to deploy the EMQX cluster:

```bash
$ kubectl apply -f emqx.yaml

emqxenterprise.apps.emqx.io/emqx-ee created
```

Check the status of the EMQX cluster and make sure that `STATUS` is `Running`, which may take some time to wait for the EMQX cluster to be ready.

```bash
$ kubectl get emqxenterprises

NAME      STATUS   AGE
emqx-ee   Running  8m33s
```

:::
::::

## Check the Telegraf Logs

```
kubectl logs -f $pod_name -c telegraf
```

