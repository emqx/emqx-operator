# 在阿里云上部署 EMQX 集群

EMQX Operator 支持在阿里云容器服务 Kubernetes 版部署 EMQX。阿里云容器服务 Kubernetes 版（Alibaba Cloud Container Service for Kubernetes，简称容器服务 ACK）是全球首批通过 Kubernetes 一致性认证的服务平台，提供高性能的容器应用管理服务，支持企业级 Kubernetes 容器化应用的生命周期管理，让您轻松高效地在云端运行 Kubernetes 容器化应用。详情请查看 [什么是容器服务 Kubernetes 版](https://www.alibabacloud.com/help/zh/container-service-for-kubernetes/latest/what-is-container-service-for-kubernetes)

## 前提条件

在开始之前，你需要准备以下内容：

- 开通阿里云容器服务，并创建一个 ACK 集群。具体请参考：[首次使用容器服务 Kubernetes 版](https://www.alibabacloud.com/help/zh/container-service-for-kubernetes/latest/quick-start-for-first-time-users)

- 通过 kubectl 命令连接 ACK 集群，你可以在本地安装 kubectl 工具，并获取集群的 KubeConfig 来连接集群，或是在容器服务 ACK 控制台上利用 CloudShell 通过 kubectl 管理集群。

  - 通过本地安装 kubectl 工具连接 ACK 集群：具体请参考：[获取集群 KubeConfig 并通过 kubectl 工具连接集群](https://www.alibabacloud.com/help/zh/container-service-for-kubernetes/latest/connect-to-ack-clusters-by-using-kubectl)
  - 通过 CloudShell 连接 ACK 集群：具体请参考：[在 CloudShell 上通过 kubectl 管理 Kubernetes 集群](https://www.alibabacloud.com/help/zh/container-service-for-kubernetes/latest/use-kubectl-on-cloud-shell-to-manage-ack-clusters)

- 安装 EMQX Operator：具体请参考：[安装 EMQX Operator](../getting-started/getting-started.md)

## 快速部署一个 EMQX 集群

下面是 EMQX 自定义资源的相关配置。你可以根据你想部署的 EMQX 版本选择相应的 APIVersion。关于具体的兼容性关系，请参考[EMQX 与 EMQX Operator 的兼容性列表](../README.md)：

:::: tabs type:card
::: tab apps.emqx.io/v1beta4

+ 将下面的内容保存成 YAML 文件，并通过 `kubectl apply` 命令部署它。

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
        ## 更多内容：https://help.aliyun.com/document_detail/134722.html
        storageClassName: alibabacloud-cnfs-nas
        resources:
          requests:
            storage: 20Mi
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
          ## NLB 支持的地域及可用区可以登录 NLB 控制台查看，至少需要两个可用区。多个可用区间用逗号分隔，如 cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321 。
          service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}"
      spec:
        type: LoadBalancer
        ## 更多内容：https://www.alibabacloud.com/help/zh/container-service-for-kubernetes/latest/configurenlbthroughannotation
        loadBalancerClass: "alibabacloud.com/nlb"
  ```

+ 等待 EMQX 集群就绪，可以通过 `kubectl get` 命令查看 EMQX 集群的状态，请确保 `STATUS` 为 `Running`，这个可能需要一些时间

  ```bash
  $ kubectl get emqxenterprises
  NAME      STATUS   AGE
  emqx-ee   Running  8m33s
  ```

+ 获取 EMQX 集群的 External IP, 访问 EMQX 控制台

  ```bash
  $ external_ip=$(kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip')
  $ echo $external_ip

  198.18.3.10
  ```

  通过浏览器访问 `http://${external_ip}:18083` ，使用默认的用户名和密码 `admin/public` 登录 EMQX 控制台。

:::

::: tab apps.emqx.io/v2alpha1

+ 将下面的内容保存成 YAML 文件，并通过 `kubectl apply` 命令部署它。

  ```yaml
  apiVersion: apps.emqx.io/v2alpha1
  kind: EMQX
  metadata:
    name: emqx
  spec:
    image: "emqx:5.0"
    coreTemplate:
      spec:
        ## EMQX 自定义资源不支持在运行时更新这个字段
        volumeClaimTemplates:
          ## 更多内容：https://help.aliyun.com/document_detail/134722.html
          storageClassName: alibabacloud-cnfs-nas
          resources:
            requests:
              storage: 20Mi
          accessModes:
            - ReadWriteOnce
    dashboardServiceTemplate:
      metadata:
        annotations:
          ## NLB 支持的地域及可用区可以登录 NLB 控制台查看，至少需要两个可用区。多个可用区间用逗号分隔，如 cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321 。
          service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}"
      spec:
        type: LoadBalancer
        ## 更多内容：https://www.alibabacloud.com/help/zh/container-service-for-kubernetes/latest/configurenlbthroughannotation
        loadBalancerClass: "alibabacloud.com/nlb"
    listenersServiceTemplate:
      metadata:
        annotations:
          ## NLB 支持的地域及可用区可以登录 NLB 控制台查看，至少需要两个可用区。多个可用区间用逗号分隔，如 cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321 。
          service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}"
      spec:
        type: LoadBalancer
        ## 更多内容：https://www.alibabacloud.com/help/zh/container-service-for-kubernetes/latest/configurenlbthroughannotation
        loadBalancerClass: "alibabacloud.com/nlb"
  ```

+ 等待 EMQX 集群就绪，可以通过 `kubectl get` 命令查看 EMQX 集群的状态，请确保 `STATUS` 为 `Running`，这个可能需要一些时间

  ```bash
  $ kubectl get emqx
  NAME   IMAGE      STATUS    AGE
  emqx   emqx:5.0   Running   2m55s
  ```

+ 获取 EMQX 集群 Dashboard External IP, 访问 EMQX 控制台

  EMQX Operator 会创建两个 EMQX Service 资源，一个是 `emqx-dashboard`，一个是 `emqx-listeners`，分别对应 EMQX 控制台和 EMQX 监听端口。

  ```bash
  $ external_ip=$(kubectl get svc emqx-dashboard -o json | jq '.status.loadBalancer.ingress[0].ip')
  $ echo $external_ip

  198.18.3.10
  ```

  通过浏览器访问 `http://${external_ip}:18083` ，使用默认的用户名和密码 `admin/public` 登录 EMQX 控制台。

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

## 使用 LoadBalancer 终结 TLS 加密

在使用 NLB 终结 TLS 流量之前，需要先创建一个 TLS 证书，阿里云的[数字证书管理服务](https://us-east-2.console.aws.amazon.com/acm/home)控制台，可以导入自签名证书或者购买证书, 证书导入后点击证书详情，获取证书 ID。

> 由于每次重新创建 NLB 时，其关联的 DNS 域名会发生变化，如果采用自签名证书，为方便测试，这里建议将证书绑定的域名设置为`*.cn-shanghai.nlb.aliyuncs.com`

修改 EMQX Custom Resource 的配置，将相关的 annotations 添加到 EMQX Custom Resource 中，并更新 Service Template 中监听的端口。


:::: tabs type:card
::: tab apps.emqx.io/v1beta4

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
      storageClassName: alibabacloud-cnfs-nas
      resources:
        requests:
          storage: 20Mi
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
        ## NLB 支持的地域及可用区可以登录 NLB 控制台查看，至少需要两个可用区。多个可用区间用逗号分隔，如 cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321 。
        service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}"
        ## 如集群为中国内地 Region 时，组合后的证书 ID 为 ${your-cert-id}-cn-hangzhou。如集群为的其他 Region 时，组合后的证书 ID 为 ${your-cert-id}-ap-southeast-1。
        service.beta.kubernetes.io/alibaba-cloud-loadbalancer-cert-id: "${组合后的证书 ID}"
        ## LoadBalancer 监听的 SSL 端口
        service.beta.kubernetes.io/alibaba-cloud-loadbalancer-protocol-port: "tcpssl:30883"
    spec:
      type: LoadBalancer
      ## 更多内容：https://www.alibabacloud.com/help/zh/container-service-for-kubernetes/latest/configurenlbthroughannotation
      loadBalancerClass: "alibabacloud.com/nlb"
      ports:
        - name: tcpssl
          ## LoadBalancer 监听的 SSL 端口
          port: 30883
          protocol: TCP
          ## MQTT TCP 端口
          targetPort: 1883
```
:::

::: tab apps.emqx.io/v2alpha1

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
  name: emqx
spec:
  image: "emqx:5.0"
  coreTemplate:
    spec:
      ## EMQX 自定义资源不支持在运行时更新这个字段
      volumeClaimTemplates:
        storageClassName: alibabacloud-cnfs-nas
        resources:
          requests:
            storage: 20Mi
        accessModes:
          - ReadWriteOnce
  dashboardServiceTemplate:
    metadata:
      annotations:
        ## NLB 支持的地域及可用区可以登录 NLB 控制台查看，至少需要两个可用区。多个可用区间用逗号分隔，如 cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321 。
        service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}"
    spec:
      type: LoadBalancer
      ## 更多内容：https://www.alibabacloud.com/help/zh/container-service-for-kubernetes/latest/configurenlbthroughannotation
      loadBalancerClass: "alibabacloud.com/nlb"
  listenersServiceTemplate:
    metadata:
      annotations:
        ## NLB 支持的地域及可用区可以登录 NLB 控制台查看，至少需要两个可用区。多个可用区间用逗号分隔，如 cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321 。
        service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}"
        ## 如集群为中国内地 Region 时，组合后的证书 ID 为 ${your-cert-id}-cn-hangzhou。如集群为的其他 Region 时，组合后的证书 ID 为 ${your-cert-id}-ap-southeast-1。
        service.beta.kubernetes.io/alibaba-cloud-loadbalancer-cert-id: "${组合后的证书 ID}"
        ## LoadBalancer 监听的 SSL 端口
        service.beta.kubernetes.io/alibaba-cloud-loadbalancer-protocol-port: "tcpssl:30883"
    spec:
      type: LoadBalancer
      ## 更多内容：https://www.alibabacloud.com/help/zh/container-service-for-kubernetes/latest/configurenlbthroughannotation
      loadBalancerClass: "alibabacloud.com/nlb"
      ports:
        - name: tcpssl
          ## LoadBalancer 监听的 SSL 端口
          port: 30883
          protocol: TCP
          ## MQTT TCP 端口
          targetPort: 1883
```

:::
::::

<!-- TODO -->
<!-- LoadBalancer 直连 Pod -->
