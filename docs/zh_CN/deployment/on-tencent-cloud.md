# 在腾讯云上部署 EMQX 集群

EMQX Operator 支持在腾讯云容器服务（Tencent Kubernetes Engine，TKE）部署 EMQX。腾讯云容器服务基于原生 kubernetes 提供以容器为核心的、高度可扩展的高性能容器管理服务。腾讯云容器服务完全兼容原生 kubernetes API，为容器化的应用提供高效部署、资源调度、服务发现和动态伸缩等一系列完整功能，解决用户开发、测试及运维过程的环境一致性问题，提高了大规模容器集群管理的便捷性，帮助用户降低成本，提高效率。容器服务会对不同规格的托管集群收取相应的集群管理费用。在使用中创建的其他的云产品资源（CVM、CBS、CLB 等），将按照各自云产品的计费方式进行收费

## 前提条件

在开始之前，你需要准备以下内容：

- 开通腾讯云容器服务，并创建一个 TKE 集群。具体请参考：[创建 TKE 集群](https://cloud.tencent.com/document/product/457/32189)

- 通过 kubectl 命令连接 TKE 集群，你可以在本地安装 kubectl 工具，并获取集群的 KubeConfig 来连接集群，或是在容器服务 TKE 控制台上利用 CloudShell 通过 kubectl 管理集群。

  - 通过本地安装 kubectl 工具连接 TKE 集群：具体请参考：[使用 kubectl 连接集群](https://cloud.tencent.com/document/product/457/32191#a334f679-7491-4e40-9981-00ae111a9094)
  - 通过 CloudShell 连接 TKE 集群：具体请参考：[使用 CloudShell 连接集群](https://cloud.tencent.com/document/product/457/32191#f97c271a-1204-44d5-967c-2856c83cc5e3)

- 安装 EMQX Operator：具体请参考：[安装 EMQX Operator](../getting-started/getting-started.md)

## 快速部署一个 EMQX 集群

下面是 EMQX 自定义资源的相关配置。你可以根据你想部署的 EMQX 版本选择相应的 APIVersion。关于具体的兼容性关系，请参考[ EMQX 与 EMQX Operator 的兼容性列表](../index.md)：

:::: tabs type:card
::: tab apps.emqx.io/v1beta4

+ 将下面的内容保存成 YAML 文件，并通过 `kubectl apply` 命令部署它

  ```yaml
  apiVersion: apps.emqx.io/v1beta4
  kind: EmqxEnterprise
  metadata:
    name: emqx-ee
  spec:
     ## EMQX 自定义资源不支持在运行时更新这个字段
    persistent:
      metadata:
        name: emqx-ee
      spec:
        ## 更多内容：https://cloud.tencent.com/document/product/457/44239
        storageClassName: cbs
        resources:
          requests:
            ## 腾讯云 TKE 要求云硬盘大小必须为 10 的倍数，默认提供的 cbs（高性能云盘） 要求硬盘最小为 10GB，更多内容请参考：https://cloud.tencent.com/document/product/457/44239
            storage: 10Gi
        accessModes:
          - ReadWriteOnce
    template:
      spec:
        emqxContainer:
          image:
            repository: emqx/emqx-ee
            version: 4.4.14
    serviceTemplate:
      metadata:
        annotations:
          # 自动创建 tke-service-config，更多内容请参考：https://cloud.tencent.com/document/product/457/45490
          service.cloud.tencent.com/tke-service-config-auto: "true"
      spec:
        type: LoadBalancer
  ```

+ 等待 EMQX 集群就绪，可以通过 `kubectl get` 命令查看 EMQX 集群的状态，请确保 `STATUS` 为 `Running`，这个可能需要一些时间

  ```bash
  $ kubectl get emqxenterprises
  NAME      STATUS   AGE
  emqx-ee   Running  8m33s
  ```

+ 获取 EMQX 集群的 External IP，访问 EMQX 控制台

  ```bash
  $ kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip'
  
  198.18.3.10
  ```

  通过浏览器访问 `http://198.18.3.10:18083` ，使用默认的用户名和密码 `admin/public` 登录 EMQX 控制台。

:::
::: tab apps.emqx.io/v2alpha1

+ 将下面的内容保存成 YAML 文件，并通过 `kubectl apply` 命令部署它

  ```yaml
  apiVersion: apps.emqx.io/v2alpha1
  kind: EMQX
  metadata:
    name: emqx
  spec:
    image: emqx:5.0
    coreTemplate:
      spec:
        ## EMQX 自定义资源不支持在运行时更新这个字段
        volumeClaimTemplates:
          ## 更多内容：https://cloud.tencent.com/document/product/457/44238
          storageClassName: cbs
          resources:
            requests:
              ## 云硬盘大小必须为10的倍数。高性能云硬盘最小为10GB，更多内容请参考：https://cloud.tencent.com/document/product/457/44239
              storage: 10Gi
          accessModes:
            - ReadWriteOnce
    dashboardServiceTemplate:
      metadata:
        annotations:
          # 自动创建 tke-service-config，更多内容请参考：https://cloud.tencent.com/document/product/457/45490
          service.cloud.tencent.com/tke-service-config-auto: "true"
      spec:
        type: LoadBalancer
    listenersServiceTemplate:
      metadata:
        annotations:
          # 自动创建 tke-service-config，更多内容请参考：https://cloud.tencent.com/document/product/457/45490
          service.cloud.tencent.com/tke-service-config-auto: "true"
      spec:
        type: LoadBalancer
  ```

+ 等待 EMQX 集群就绪，可以通过 `kubectl get` 命令查看 EMQX 集群的状态，请确保 `STATUS` 为 `Running`，这个可能需要一些时间

  ```bash
  $ kubectl get emqx
  NAME   IMAGE      STATUS    AGE
  emqx   emqx:5.0   Running   2m55s
  ```

+ 获取 EMQX 集群的 External IP，访问 EMQX 控制台

  EMQX Operator 会创建两个 EMQX Service 资源，一个是 `emqx-dashboard`，一个是 `emqx-listeners`，分别对应 EMQX 控制台和 EMQX 监听端口。

  ```bash
  $ kubectl get svc emqx-dashboard -o json | jq '.status.loadBalancer.ingress[0].ip'
  
  198.18.3.10
  ```

  通过浏览器访问 `http://198.18.3.10:18083` ，使用默认的用户名和密码 `admin/public` 登录 EMQX 控制台。

  :::
  ::::

## 使用 MQTT X CLI 连接 EMQX 集群发布/订阅消息

[MQTT X CLI](https://mqttx.app/zh/cli) 是一款开源的 MQTT 5.0 命令行客户端工具，旨在帮助开发者在不需要使用图形化界面的基础上，也能更快的开发和调试 MQTT 服务与应用。

+ 获取 EMQX 集群的 External IP

  :::: tabs type:card
  ::: tab apps.emqx.io/v1beta4

  ```bash
  external_ip=$(kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip')
  ```
  :::
  ::: tab apps.emqx.io/v2alpha1

  ```bash
  external_ip=$(kubectl get svc emqx-listeners -o json | jq '.status.loadBalancer.ingress[0].ip')
  ```
  :::
  ::::

+ 订阅消息

  ```bash
  $ mqttx sub -t 'hello' -h ${external_ip} -p 1883

  [10:00:25] › …  Connecting...
  [10:00:25] › ✔  Connected
  [10:00:25] › …  Subscribing to hello...
  [10:00:25] › ✔  Subscribed to hello
  ```

+ 创建一个新的终端窗口并发布消息

  ```bash
  $ mqttx pub -t 'hello' -h ${external_ip} -p 1883 -m 'hello world'

  [10:00:58] › …  Connecting...
  [10:00:58] › ✔  Connected
  [10:00:58] › …  Message Publishing...
  [10:00:58] › ✔  Message published
  ```

+ 查看订阅终端窗口收到的消息

  ```bash
  [10:00:58] › payload: hello world
  ```

## 关于 LoadBalancer 终结 TLS

目前腾讯云 CLB 不支持终结 TLS ，如需要使用 LoadBalancer 终结 TLS 请参考[终结 TLS](https://github.com/emqx/emqx-operator/discussions/312)。
