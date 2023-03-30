# 配置 EMQX 日志等级

## 任务目标

- 如何配置 EMQX 集群日志等级。

## 配置 EMQX 集群

:::: tabs type:card 
::: tab v2alpha1

EMQX CRD 支持使用 `.spec.bootstrapConfig` 来配置 EMQX 集群日志等级，bootstrapConfig 的配置可以参考文档：[bootstrapConfig](https://www.emqx.io/docs/zh/v5.0/admin/cfg.html)。这个字段只允许在创建 EMQX 集群的时候配置，不支持更新。**注意：** 如果在创建 EMQX 之后需要修改集群日志等级，请通过 EMQX Dashboard 进行修改。

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
  name: emqx
spec:
  image: emqx/emqx:5.0.14
  imagePullPolicy: IfNotPresent
  bootstrapConfig: |
    log {
       console_handler {
          level  =  debug
        } 
    }
  coreTemplate:
    spec:
      replicas: 3
  replicantTemplate:
    spec:
      replicas: 0
  listenersServiceTemplate:
    spec:
      type: NodePort
      ports:
        - name: "tcp-default"
          protocol: TCP
          port: 1883
          targetPort: 1883
          nodePort: 32010
```

> `.spec.bootstrapConfig` 字段配置 EMQX 集群日志等级为 `debug`。

:::
::: tab v1beta4

EMQX 企业版在 EMQX Operator 里面对应的 CRD 为 EmqxEnterprise，EmqxEnterprise 支持通过 `.spec.template.spec.emqxContainer.emqxConfig` 字段配置 EMQX 集群日志等级。emqxConfig 字段的具体描述可以参考：[emqxConfig](https://github.com/emqx/emqx-operator/blob/main-2.1/docs/en_US/reference/v1beta4-reference.md#emqxtemplatespec)。

```yaml
apiVersion: apps.emqx.io/v1beta4
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  template:
    spec:
      emqxContainer:
        image:
          repository: emqx/emqx-ee
          version: 4.4.14
        emqxConfig:
          log.level: debug
  serviceTemplate:
    spec:
      type: NodePort
      ports:
        - name: "mqtt-tcp-1883"
          protocol: "TCP"
          port: 1883
          targetPort: 1883
          nodePort: 32010
```

> `.spec.template.spec.emqxContainer.emqxConfig` 字段配置 EMQX 集群日志等级为 `debug`。

:::
::: tab v1beta3

EMQX 企业版在 EMQX Operator 里面对应的 CRD 为 EmqxEnterprise，EmqxEnterprise 支持通过 `.spec.emqxTemplate.config` 字段配置集群日志等级。config 字段的描述可以参考文档：[config](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/reference/v1beta3-reference.md#emqxenterprisetemplate)

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  emqxTemplate:
    image: emqx/emqx-ee:4.4.14
    config:
      log.level: debug
    serviceTemplate:
      spec:
        type: NodePort
        ports:
          - name: "mqtt-tcp-1883"
            protocol: "TCP"
            port: 1883
            targetPort: 1883
            nodePort: 32010
```

> `.spec.emqxTemplate.config` 字段配置 EMQX 集群日志等级为 `debug`。

:::
::::

将上述内容保存为：emqx-log-level.yaml，并执行如下命令部署 EMQX 集群：

```bash
kubectl apply -f emqx-log-level.yaml
```

输出类似于：

```
emqx.apps.emqx.io/emqx created
```

- 检查 EMQX 集群是否就绪

:::: tabs type:card 
::: tab v2alpha1

```bash
kubectl get emqx emqx -o json | jq '.status.conditions[] | select( .type == "Running" and .status == "True")'
```

输出类似于：

```bash
{
  "lastTransitionTime": "2023-03-01T02:17:03Z",
  "lastUpdateTime": "2023-03-01T02:17:03Z",
  "message": "Cluster is running",
  "reason": "ClusterRunning",
  "status": "True",
  "type": "Running"
}
```

::: 
::: tab v1beta4

```bash
kubectl get emqxEnterprise emqx-ee -o json | jq '.status.conditions[] | select( .type == "Running" and .status == "True")
```
输出类似于：

```bash
{
  "lastTransitionTime": "2023-03-01T02:49:22Z",
  "lastUpdateTime": "2023-03-01T02:49:23Z",
  "message": "All resources are ready",
  "reason": "ClusterReady",
  "status": "True",
  "type": "Running"
}
```

::: 
::: tab v1beta3

```bash
kubectl get emqxEnterprise emqx-ee -o json | jq '.status.conditions[] | select( .type == "Running" and .status == "True")
```

输出类似于：

```bash
{
  "lastTransitionTime": "2023-03-01T02:49:22Z",
  "lastUpdateTime": "2023-03-01T02:49:23Z",
  "message": "All resources are ready",
  "reason": "ClusterReady",
  "status": "True",
  "type": "Running"
}
```

::: 
::::

## 验证 EMQX 集群日志等级配置是否生效

- 使用 MQTT X 连接 EMQX 集群发送消息

在 MQTT X 页面点击创建新连接的按钮，按照如图所示配置 EMQX 集群节点信息，在配置好连接信息之后，点击 connect 按钮连接 EMQX 集群：

![](./assets/configure-log-level/mqtt-connected.png)

然后点击订阅按钮新建订阅，如图所示 MQTT X 已成功连接 EMQX 集群并且已经成功创建订阅：

![](./assets/configure-log-level/mqtt-sub.png)

在成功连接 EMQX 集群并创建订阅之后，我们就可以向 EMQX 集群发送消息，如下图所示：

![](./assets/configure-log-level/mqtt-pub.png)

-  使用命令行查看 EMQX 集群日志信息

```bash
kubectl logs emqx-core-0 -c emqx 
```

输出如下图所示：

![](./assets/configure-log-level/emqx-debug-log.png)

从图中可以看到刚才使用 MQTT 连接 EMQX 集群建立连接以及发送消息的 debug 日志信息。
