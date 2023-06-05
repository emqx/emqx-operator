# 在 Azure 中部署 EMQX

EMQX 是一款高性能的开源分布式物联网 MQTT 消息服务器，它提供了可靠、高效的消息传递功能。而 Azure Kubernetes Service（AKS）作为一种托管的 Kubernetes 服务，提供了便捷的容器化应用程序部署和管理能力。在本文中，我们将介绍如何利用 EMQX Operator 在 Azure AKS 上部署 EMQX，从而构建强大的物联网 MQTT 通信解决方案。


## 前提条件

在开始之前，您必须具备以下条件：

- 要在 Azure 上创建一个 AKS 集群，您首先需要在您的 Azure 订阅中激活 AKS 服务。请参考 [Azure Kubernetes 服务](https://learn.microsoft.com/zh-cn/azure/aks/) 文档以获取更多信息。

- 要使用 kubectl 命令连接到一个 AKS 集群，您可以在本地安装 kubectl 工具并获取集群的 KubeConfig 来连接到集群。或者，您可以通过 Azure 门户使用 Cloud Shell 来管理集群。
  - 要使用 kubectl 连接到一个 AKS 集群，您需要在您的本地机器上安装并配置 kubectl 工具。请参考 [连接到一个 AKS 集群](https://learn.microsoft.com/zh-cn/azure/aks/learn/quick-kubernetes-deploy-cli) 文档。
  - 要使用 CloudShell 连接到一个 AKS 集群，使用 Azure CloudShell 连接到 AKS 集群并使用 kubectl 管理集群。请参考 [在 Azure CloudShell 中管理一个 AKS 集群](https://learn.microsoft.com/zh-cn/azure/aks/learn/quick-kubernetes-deploy-portal?tabs=azure-cli) 文档，了解如何连接到 Azure CloudShell 和使用 kubectl 的详细说明。

- 要安装 EMQX Operator，请参考 [安装 EMQX Operator](../getting-started/getting-started.md)。

## 快速部署一个 EMQX 集群

以下是 EMQX Custom Resource 的相关配置。您可以根据您想要部署的 EMQX 版本选择相应的 APIVersion。具体的兼容关系，请参考 [EMQX Operator 兼容性](../README.md)：

:::: tabs type:card
::: tab apps.emqx.io/v1beta4

将以下内容保存为一个 YAML 文件，并使用 `kubectl apply` 命令进行部署。

```yaml
apiVersion: apps.emqx.io/v1beta4
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  persistent:
    metadata:
      name: emqx-ee
    spec:
      ## 关于存储类的更多信息：https://learn.microsoft.com/zh-cn/azure/aks/concepts-storage#storage-classes
      storageClassName: default
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
          version: 4.4.15
  serviceTemplate:
    spec:
      ## 关于负载均衡器的更多信息：https://learn.microsoft.com/zh-cn/azure/aks/load-balancer-standard
      type: LoadBalancer
```

等待 EMQX 集群准备就绪。您可以使用 kubectl get 命令检查 EMQX 集群的状态。请确保状态为 Running，这可能需要一些时间。


```shell
$ kubectl get emqxenterprises
NAME      STATUS   AGE
emqx-ee   Running  8m33s
```

获取 EMQX 集群的外部 IP，并访问 EMQX 控制台。

```shell
$ kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip'

20.245.123.100
```

通过打开一个网络浏览器并访问 http://20.245.123.100:18083 来访问 EMQX 控制台。使用默认的用户名和密码 admin/public 登录。

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
      volumeClaimTemplates:
        ## 关于存储类的更多信息：https://learn.microsoft.com/zh-cn/azure/aks/concepts-storage#storage-classes
        storageClassName: default
        resources:
          requests:
            storage: 10Gi
        accessModes:
        - ReadWriteOnce
  dashboardServiceTemplate:
    spec:
      ## 关于负载均衡器的更多信息：https://learn.microsoft.com/zh-cn/azure/aks/load-balancer-standard
      type: LoadBalancer
  listenersServiceTemplate:
    spec:
      ## 关于负载均衡器的更多信息：https://learn.microsoft.com/zh-cn/azure/aks/load-balancer-standard
      type: LoadBalancer
```

等待 EMQX 集群准备就绪。您可以使用 kubectl get 命令检查 EMQX 集群的状态。请确保状态为 Running，这可能需要一些时间。

```shell
$ kubectl get emqx
NAME   IMAGE      STATUS    AGE
emqx   emqx:5.0   Running   118s
```

获取 EMQX 集群的外部 IP，并访问 EMQX 控制台。

EMQX Operator 将创建两个 EMQX 服务资源，一个是 emqx-dashboard，另一个是 emqx-listeners，分别对应 EMQX 控制台和 EMQX 监听端口。

```shell
$ kubectl get svc emqx-dashboard -o json | jq '.status.loadBalancer.ingress[0].ip'

52.132.12.100
```

通过打开一个网络浏览器并访问 http://52.132.12.100:18083 来访问 EMQX 控制台。使用默认的用户名和密码 admin/public 登录。

:::
::::

## 使用 MQTT X CLI 连接到 EMQX 集群以发布/订阅消息

MQTT X CLI 是一个开源的 MQTT 5.0 命令行客户端工具，旨在帮助开发者无需 GUI 即可更快地开发和调试 MQTT 服务和应用。

- 获取 EMQX 集群的外部 IP

    :::: tabs type:card
    ::: tab apps.emqx.io/v1beta4

    ```shell
    external_ip=$(kubectl get svc emqx-listeners -o json | jq '.status.loadBalancer.ingress[0].ip')
    ```

    :::
    ::: tab apps.emqx.io/v2alpha1

    ```shell
    external_ip=$(kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip')
    ```

    :::
    ::::

- 订阅消息

  ```shell
  $ mqttx sub -t 'hello' -h ${external_ip} -p 1883

  [10:00:25] › …  Connecting...
  [10:00:25] › ✔  Connected
  [10:00:25] › …  Subscribing to hello...
  [10:00:25] › ✔  Subscribed to hello
  ```

- 创建一个新的终端窗口并发送消息

    ```shell
    $ mqttx pub -t 'hello' -h ${external_ip} -p 1883 -m 'hello world'

    [10:00:58] › …  Connecting...
    [10:00:58] › ✔  Connected
    [10:00:58] › …  Message Publishing...
    [10:00:58] › ✔  Message published
    ```

- 在订阅终端窗口中查看接收到的消息

  ```shell
  [10:00:58] › payload: hello world
  ```

## 使用 LoadBalancer 进行 TLS 终结

由于 Azure LoadBalancer 不支持 TCP 证书，请参阅这个[文档](https://github.com/emqx/emqx-operator/discussions/312)解决 TCP 证书卸载问题。
