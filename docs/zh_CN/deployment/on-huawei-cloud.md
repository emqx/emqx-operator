# 在华为云上部署 EMQX 集群

EMQX Operator 支持在华为云容器引擎（Cloud Container Engine，简称 CCE）部署 EMQX。云容器引擎提供高度可扩展的、高性能的企业级 Kubernetes 集群，支持运行 Docker 容器。借助云容器引擎，您可以在华为云上轻松部署、管理和扩展容器化应用程序。

云容器引擎深度整合高性能的计算（ECS/BMS）、网络（VPC/EIP/ELB）、存储（EVS/OBS/SFS）等服务，并支持 GPU、NPU、ARM 等异构计算架构，支持多可用区（Available Zone，简称 AZ）、多区域（Region）容灾等技术构建高可用 Kubernetes 集群。关于更多云容器引擎 CCE 产品介绍，请查看 [什么是云容器引擎](https://support.huaweicloud.com/productdesc-cce/cce_productdesc_0001.html?utm_source=cce_Growth_map&utm_medium=display&utm_campaign=help_center&utm_content=Growth_map)。

## 前提条件

在开始之前，你需要准备以下内容：

- 开通华为云容器服务，并创建一个 CCE 集群。具体请参考：[创建 CCE 集群](https://support.huaweicloud.com/usermanual-cce/cce_01_0028.html)

    ::: tip
    Kubernetes 集群节点必须可以访问外网（可以通过加 NAT 网关解决），否则无法拉取除容器镜像服务（SoftWare Repository）外的第三方镜像
    :::

    :::tip
    Kubernetes 集群节点的操作系统建议选择 Ubuntu，否则有可能会缺少必要的库（socat）
    :::

- 通过 kubectl 命令连接 CCE 集群，你可以在本地安装 kubectl 工具，并获取集群的 KubeConfig 来连接集群，或是在容器服务 CCE 控制台上利用 CloudShell 通过 kubectl 管理集群。

  - 通过本地安装 kubectl 工具连接 CCE 集群：具体请参考：[使用 kubectl 连接集群](https://support.huaweicloud.com/usermanual-cce/cce_10_0107.html#section3)
  - 通过 CloudShell 连接 CCE 集群：具体请参考：[使用 CloudShell 连接集群](https://support.huaweicloud.com/usermanual-cce/cce_10_0107.html#section2)

- 安装 EMQX Operator：具体请参考：[安装 EMQX Operator](../getting-started/getting-started.md)

## 快速部署一个 EMQX 集群

下面是 EMQX 自定义资源的相关配置。你可以根据你想部署的 EMQX 版本选择相应的 APIVersion。关于具体的兼容性关系，请参考[ EMQX 与 EMQX Operator 的兼容性列表](../index.md)：

:::: tabs type:card
::: tab apps.emqx.io/v2beta1

+ 将下面的内容保存成 YAML 文件，并通过 `kubectl apply` 命令部署它

  ```yaml
  apiVersion: apps.emqx.io/v2beta1
  kind: EMQX
  metadata:
    name: emqx
  spec:
    image: emqx:5
    coreTemplate:
      spec:
        ## EMQX 自定义资源不支持在运行时更新这个字段
        volumeClaimTemplates:
          ## 更多内容：https://support.huaweicloud.com/usermanual-cce/cce_10_0380.html#section1
          storageClassName: csi-disk
          resources:
            requests:
              storage: 10Gi
          accessModes:
            - ReadWriteOnce
    dashboardServiceTemplate:
      metadata:
        annotations:
          ## 自动创建关联的 ELB，详细字段说明请参考：https://support.huaweicloud.com/usermanual-cce/cce_10_0014.html#cce_10_0014__table939522754617
          kubernetes.io/elb.autocreate: |
            {
              "type": "public",
              "bandwidth_name": "cce-emqx",
              "bandwidth_size": 5,
              "bandwidth_sharetype": "PER",
              "eip_type": "5_bgp"
            }
      spec:
        type: LoadBalancer
    listenersServiceTemplate:
      metadata:
        annotations:
          ## 自动创建关联的 ELB，详细字段说明请参考：https://support.huaweicloud.com/usermanual-cce/cce_10_0014.html#cce_10_0014__table939522754617
          kubernetes.io/elb.autocreate: |
            {
              "type": "public",
              "bandwidth_name": "cce-emqx",
              "bandwidth_size": 5,
              "bandwidth_sharetype": "PER",
              "eip_type": "5_bgp"
            }
      spec:
        type: LoadBalancer
  ```

+ 等待 EMQX 集群就绪，可以通过 `kubectl get` 命令查看 EMQX 集群的状态，请确保 `STATUS` 为 `Running`，这个可能需要一些时间

  ```bash
  $ kubectl get emqx
  NAME   IMAGE      STATUS    AGE
  emqx   emqx:5.1   Running   2m55s
  ```

+ 获取 EMQX 集群的 External IP，访问 EMQX 控制台

  EMQX Operator 会创建两个 EMQX Service 资源，一个是 `emqx-dashboard`，一个是 `emqx-listeners`，分别对应 EMQX 控制台和 EMQX 监听端口。

  ```bash
  $ kubectl get svc emqx-dashboard -o json | jq '.status.loadBalancer.ingress[0].ip'

  198.18.3.10

  通过浏览器访问 `http://198.18.3.10:18083` ，使用默认的用户名和密码 `admin/public` 登录 EMQX 控制台。

  ```
:::
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
        ## 更多内容：https://support.huaweicloud.com/usermanual-cce/cce_10_0380.html#section1
        storageClassName: csi-disk
        resources:
          requests:
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
          ## 自动创建关联的 ELB，详细字段说明请参考：https://support.huaweicloud.com/usermanual-cce/cce_10_0014.html#cce_10_0014__table939522754617
          kubernetes.io/elb.autocreate: |
            {
              "type": "public",
              "bandwidth_name": "cce-emqx",
              "bandwidth_size": 5,
              "bandwidth_sharetype": "PER",
              "eip_type": "5_bgp"
            }
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
::::

## 使用 MQTT X CLI 连接 EMQX 集群发布/订阅消息

[MQTT X CLI](https://mqttx.app/zh/cli) 是一款开源的 MQTT 5.0 命令行客户端工具，旨在帮助开发者在不需要使用图形化界面的基础上，也能更快的开发和调试 MQTT 服务与应用。

+ 获取 EMQX 集群的 External IP

  :::: tabs type:card
  ::: tab apps.emqx.io/v2beta1

  ```bash
  external_ip=$(kubectl get svc emqx-listeners -o json | jq '.status.loadBalancer.ingress[0].ip')
  ```
  :::
  ::: tab apps.emqx.io/v1beta4

  ```bash
  external_ip=$(kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip')
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

由于华为 ELB 不支持 TCP 证书，所以请参考文档[终结 TLS](https://github.com/emqx/emqx-operator/discussions/312)解决 TCP 证书终结问题。
