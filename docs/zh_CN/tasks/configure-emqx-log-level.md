# 修改 EMQX 日志等级

## 任务目标

如何修改 EMQX 集群日志等级。

## 配置 EMQX 集群

下面是 EMQX Custom Resource 的相关配置，你可以根据希望部署的 EMQX 的版本来选择对应的 APIVersion，具体的兼容性关系，请参考 [EMQX Operator 兼容性](../index.md):

:::: tabs type:card
::: tab apps.emqx.io/v1beta4

`apps.emqx.io/v1beta4 EmqxEnterprise` 支持通过 `.spec.template.spec.emqxContainer.emqxConfig` 字段配置 EMQX 集群日志等级。emqxConfig 字段的具体描述可以参考：[emqxConfig](../reference/v1beta4-reference.md#emqxtemplatespec)。

+ 将下面的内容保存成 YAML 文件，并通过 `kubectl apply` 命令部署它

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
        type: LoadBalancer
  ```

  > `.spec.template.spec.emqxContainer.emqxConfig` 字段配置 EMQX 集群日志等级为 `debug`。

+ 等待 EMQX 集群就绪，可以通过 `kubectl get` 命令查看 EMQX 集群的状态，请确保 `STATUS` 为 `Running`，这个可能需要一些时间

  ```bash
  $ kubectl get emqxenterprises
  NAME      STATUS   AGE
  emqx-ee   Running  8m33s
  ```

+ 获取 EMQX 集群的 External IP，访问 EMQX 控制台

  ```bash
  $ kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip'

  192.168.1.200
  ```

  通过浏览器访问 `http://192.168.1.200:18083`，使用默认的用户名和密码 `admin/public` 登录 EMQX 控制台。

:::
::: tab apps.emqx.io/v2alpha2

`apps.emqx.io/v2alpha2 EMQX` 支持通过 `.spec.config.data` 来配置 EMQX 集群日志等级，EMQX 配置可以参考文档：[配置手册](https://www.emqx.io/docs/zh/v5.1/configuration/configuration-manual.html#%E8%8A%82%E7%82%B9%E8%AE%BE%E7%BD%AE)。

> 这个字段只允许在创建 EMQX 集群的时候配置，不支持更新。如果在创建 EMQX 之后需要修改集群日志等级，请通过 EMQX Dashboard 进行修改。

+ 将下面的内容保存成 YAML 文件，并通过 `kubectl apply` 命令部署它

  ```yaml
  apiVersion: apps.emqx.io/v2alpha2
  kind: EMQX
  metadata:
    name: emqx
  spec:
    image: emqx5.0
    config:
      data: |
        log.console.level = debug
    dashboardServiceTemplate:
      spec:
        type: LoadBalancer
    listenersServiceTemplate:
      spec:
        type: LoadBalancer
  ```

  > `.spec.config.data` 字段配置 EMQX 集群日志等级为 `debug`。

+ 等待 EMQX 集群就绪，可以通过 `kubectl get` 命令查看 EMQX 集群的状态，请确保 `STATUS` 为 `Running`，这个可能需要一些时间

  ```bash
  $ kubectl get emqx
  NAME   IMAGE      STATUS    AGE
  emqx   emqx:5.1   Running   10m
  ```

+ 获取 EMQX 集群的 Dashboard External IP，访问 EMQX 控制台

  ```bash
  $ kubectl get svc emqx-dashboard -o json | jq '.status.loadBalancer.ingress[0].ip'

  192.168.1.200
  ```

  通过浏览器访问 `http://192.168.1.200:18083` ，使用默认的用户名和密码 `admin/public` 登录 EMQX 控制台。

:::
::::

## 验证日志等级

[MQTT X CLI](https://mqttx.app/zh/cli) 是一款开源的 MQTT 5.0 命令行客户端工具，旨在帮助开发者在不需要使用图形化界面的基础上，也能更快的开发和调试 MQTT 服务与应用。

+ 获取 EMQX 集群的 External IP

  :::: tabs type:card
  ::: tab apps.emqx.io/v1beta4

  ```bash
  external_ip=$(kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip')
  ```
  :::
  ::: tab apps.emqx.io/v2alpha2

  ```bash
  external_ip=$(kubectl get svc emqx-listeners -o json | jq '.status.loadBalancer.ingress[0].ip')
  ```
  :::
  ::::

+ 使用 MQTT X CLI 连接 EMQX 集群

  ```bash
  $ mqttx conn -h ${external_ip} -p 1883

  [4/17/2023] [5:17:31 PM] › …  Connecting...
  [4/17/2023] [5:17:31 PM] › ✔  Connected
  ```

+ 使用命令行查看 EMQX 集群日志信息

  ```bash
  $ kubectl logs emqx-core-0 -c emqx
  ```

  可以获取到类似如下的打印，这意味着 EMQX 收到了一个来自客户端的 CONNECT 报文，并向客户端回复了 CONNACK 报文：

  ```bash
  2023-04-17T09:11:35.993031+00:00 [debug] msg: mqtt_packet_received, mfa: emqx_channel:handle_in/2, line: 360, peername: 218.190.230.144:59457, clientid: mqttx_322680d9, packet: CONNECT(Q0, R0, D0, ClientId=mqttx_322680d9, ProtoName=MQTT, ProtoVsn=5, CleanStart=true, KeepAlive=30, Username=undefined, Password=), tag: MQTT
  2023-04-17T09:11:35.997066+00:00 [debug] msg: mqtt_packet_sent, mfa: emqx_connection:serialize_and_inc_stats_fun/1, line: 872, peername: 218.190.230.144:59457, clientid: mqttx_322680d9, packet: CONNACK(Q0, R0, D0, AckFlags=0, ReasonCode=0), tag: MQTT
  ```

