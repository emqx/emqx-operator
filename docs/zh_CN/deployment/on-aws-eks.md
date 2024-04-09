# 在 Amazon EKS 中部署 EMQX

EMQX Operator 支持在 Amazon 容器服务 EKS（Elastic Kubernetes Service）上部署 EMQX。Amazon EKS 是一种托管的 Kubernetes 服务，可让您轻松部署、管理和扩展容器化应用程序。EKS 提供了 Kubernetes 控制平面和节点组，自动处理节点替换、升级和修补。它支持 AWS 服务，如 Load Balancers、RDS 和 IAM，并与其他 Kubernetes 生态系统工具无缝集成。详情请查看 [什么是 Amazon EKS](https://docs.aws.amazon.com/zh_cn/eks/latest/userguide/what-is-eks.html)

## 前提条件

在开始之前，您需要准备以下内容：

- 开通 Amazon 容器服务，并创建一个 EKS 集群，具体请参考：[创建 Amazon EKS 集群](https://docs.aws.amazon.com/zh_cn/eks/latest/userguide/getting-started.html)

- 通过本地安装 kubectl 工具连接 EKS 集群：具体请参考：[使用 kubectl 连接集群](https://docs.aws.amazon.com/zh_cn/eks/latest/userguide/getting-started-console.html#eks-configure-kubectl)

- 在集群上部署 AWS Load Balancer Controller，具体请参考：[创建网络负载均衡器](https://docs.aws.amazon.com/zh_cn/eks/latest/userguide/network-load-balancing.html)

- 安装 EMQX Operator：具体请参考：[安装 EMQX Operator](../getting-started/getting-started.md)

## 快速部署一个 EMQX 集群

下面是 EMQX 自定义资源的相关配置。你可以根据你想部署的 EMQX 版本选择相应的 APIVersion。关于具体的兼容性关系，请参考 [EMQX 与 EMQX Operator 的兼容性列表](../index.md)

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
        ## 若开启了持久化，您需要配置 podSecurityContext，
        ## 详情请参考 discussion: https://github.com/emqx/emqx-operator/discussions/716
        podSecurityContext:
          runAsUser: 1000
          runAsGroup: 1000
          fsGroup: 1000
          fsGroupChangePolicy: Always
          supplementalGroups:
            - 1000
        ## EMQX 自定义资源不支持在运行时更新这个字段
        volumeClaimTemplates:
          ## 更多内容：https://docs.aws.amazon.com/zh_cn/eks/latest/userguide/storage-classes.html
          ## 请将 Amazon EBS CSI 驱动程序作为 Amazon EKS 附加组件管理，
          ## 更多文档请参考：https://docs.aws.amazon.com/zh_cn/eks/latest/userguide/managing-ebs-csi.html
          storageClassName: gp2
          resources:
            requests:
              storage: 10Gi
          accessModes:
            - ReadWriteOnce
    dashboardServiceTemplate:
      metadata:
        ## 更多内容：https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/guide/service/annotations/
        annotations:
          ## 指定 NLB 是面向 Internet 的还是内部的。如果未指定，则默认为内部。
          service.beta.kubernetes.io/aws-load-balancer-scheme: internet-facing
          ## 指定 NLB 将流量路由到的可用区。指定至少一个子网，subnetID 或 subnetName（子网名称标签）都可以使用。
          service.beta.kubernetes.io/aws-load-balancer-subnets: subnet-xxx1,subnet-xxx2
      spec:
        type: LoadBalancer
        ## 更多内容：https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/guide/service/nlb/
        loadBalancerClass: service.k8s.aws/nlb
    listenersServiceTemplate:
      metadata:
        ## 更多内容：https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/guide/service/annotations/
        annotations:
          ## 指定 NLB 是面向 Internet 的还是内部的。如果未指定，则默认为内部。
          service.beta.kubernetes.io/aws-load-balancer-scheme: internet-facing
          ## 指定 NLB 将流量路由到的可用区。指定至少一个子网，subnetID 或 subnetName（子网名称标签）都可以使用。
          service.beta.kubernetes.io/aws-load-balancer-subnets: subnet-xxx1,subnet-xxx2
      spec:
        type: LoadBalancer
        ## 更多内容：https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/guide/service/nlb/
        loadBalancerClass: service.k8s.aws/nlb
  ```

+ 等待 EMQX 集群就绪，可以通过 `kubectl get` 命令查看 EMQX 集群的状态，请确保 `STATUS` 为 `Running`，这个可能需要一些时间

  ```bash
  $ kubectl get emqx
  NAME   IMAGE      STATUS    AGE
  emqx   emqx:5.1   Running   18m
  ```

+ 获取 EMQX 集群的 Dashboard External IP, 访问 EMQX 控制台

  EMQX Operator 会创建两个 EMQX Service 资源，一个是 emqx-dashboard，一个是 emqx-listeners，分别对应 EMQX 控制台和 EMQX 监听端口。

  ```bash
  $ kubectl get svc emqx-dashboard -o json | jq '.status.loadBalancer.ingress[0].ip'

  192.168.1.200
  ```

  通过浏览器访问 `http://192.168.1.200:18083` ，使用默认的用户名和密码 `admin/public` 登录 EMQX 控制台。

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
        ## 更多内容：https://docs.aws.amazon.com/zh_cn/eks/latest/userguide/storage-classes.html
        ## 请将 Amazon EBS CSI 驱动程序作为 Amazon EKS 附加组件管理，更多文档请参考：https://docs.aws.amazon.com/zh_cn/eks/latest/userguide/managing-ebs-csi.html
        storageClassName: gp2
        resources:
          requests:
            storage: 10Gi
        accessModes:
          - ReadWriteOnce
    template:
      spec:
        ## 若开启了持久化，您需要配置 podSecurityContext，
        ## 详情请参考 discussion: https://github.com/emqx/emqx-operator/discussions/716
        podSecurityContext:
          runAsUser: 1000
          runAsGroup: 1000
          fsGroup: 1000
          fsGroupChangePolicy: Always
          supplementalGroups:
            - 1000
        emqxContainer:
          image:
            repository: emqx/emqx-ee
            version: 4.4.14
    serviceTemplate:
      metadata:
        ## 更多内容：https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/guide/service/annotations/
        annotations:
          ## 指定 NLB 是面向 Internet 的还是内部的。如果未指定，则默认为内部。
          service.beta.kubernetes.io/aws-load-balancer-scheme: internet-facing
          ## 指定 NLB 将流量路由到的可用区。指定至少一个子网，subnetID 或 subnetName（子网名称标签）都可以使用。
          service.beta.kubernetes.io/aws-load-balancer-subnets: subnet-xxx1, subnet-xxx2
      spec:
        type: LoadBalancer
        ## 更多内容：https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/guide/service/nlb/
        loadBalancerClass: service.k8s.aws/nlb
  ```

+ 等待 EMQX 集群就绪，可以通过 `kubectl get` 命令查看 EMQX 集群的状态，请确保 `STATUS` 为 `Running`，这个可能需要一些时间

  ```bash
  $ kubectl get emqxenterprises
  NAME      STATUS   AGE
  emqx-ee   Running  26m
  ```

+ 获取 EMQX 集群的 External IP, 访问 EMQX 控制台

  ```bash
  $ kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip'

  192.168.1.200
  ```

  通过浏览器访问 `http://192.168.1.200:18083` ，使用默认的用户名和密码 `admin/public` 登录 EMQX 控制台。

:::
::::

## 使用 MQTT X CLI 发布/订阅消息

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

## 使用 LoadBalancer 终结 TLS 加密

在 Amazon EKS 中，您可以使用 NLB 进行 TLS 终结，可以按照以下步骤进行操作：

1. 在 [AWS Certificate Manager](https://aws.amazon.com/cn/certificate-manager/?nc1=h_ls) 控制台中导入相关证书，然后通过单击证书 ID 进入详细信息页面，然后记录 ARN 信息

    :::tip

    证书和密钥的导入格式，请参考 [import certificate](https://docs.aws.amazon.com/zh_cn/acm/latest/userguide/import-certificate-format.html)

    :::

2. 在 EMQX 自定义资源的 Annotations 中添加如下注释：

    ```yaml
    ## 指定由 AWS Certificate Manager 管理的一个或多个证书的 ARN。
    service.beta.kubernetes.io/aws-load-balancer-ssl-cert: arn:aws:acm:us-west-2:xxxxx:certificate/xxxxxxx
    ## 指定是否对负载均衡器和 kubernetes pod 之间的后端流量使用 TLS。
    service.beta.kubernetes.io/aws-load-balancer-backend-protocol: tcp
    ## 指定带有 TLS 侦听器的前端端口。这意味着通过 AWS NLB 服务访问 1883 端口需要通过 TLS 认证，
    ## 但是直接访问 K8S service port 不需要 TLS 认证
    service.beta.kubernetes.io/aws-load-balancer-ssl-ports: "1883"
    ```

    > `service.beta.kubernetes.io/aws-load-balancer-ssl-cert` 的值是我们在第一步记录的 ARN 信息
