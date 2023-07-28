# 在 GCP 中部署 EMQX

EMQX 是一款高性能的开源分布式物联网 MQTT 消息服务器，它提供了可靠、高效的消息传递功能。而 Google Kubernetes Engine （GKE）作为一种托管的 Kubernetes 服务，提供了便捷的容器化应用程序部署和管理能力。在本文中，我们将介绍如何利用 EMQX Operator 在 GCP GKE 上部署 EMQX，从而构建强大的物联网 MQTT 通信解决方案。

## 前提条件

在开始之前，您必须具备以下条件：

- 要在 Google Cloud Platform 上创建 GKE 集群，您需要在 GCP 订阅中启用 GKE 服务。您可以在 Google Kubernetes Engine 文档中找到有关如何执行此操作的更多信息。

- 要使用 kubectl 命令连接到 GKE 集群，您可以在本地计算机上安装 kubectl 工具，并获取集群的 KubeConfig 以连接到集群。或者，您可以通过 GCP 控制台使用 Cloud Shell 来使用 kubectl 管理集群。

  - 要使用 kubectl 连接到 GKE 集群，您需要在本地计算机上安装并配置 kubectl 工具。有关如何执行此操作的详细说明，请参阅 [连接到 GKE 集群](https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-access-for-kubectl) 文档。

  - 要使用 Cloud Shell 连接到 GKE 集群，您可以直接使用 GCP 控制台中的 Cloud Shell 来连接到 GKE 集群并使用 kubectl 管理集群。有关如何连接到 Cloud Shell 并使用 kubectl 的详细说明，请参阅 [使用 Cloud Shell 管理 GKE 集群](https://cloud.google.com/code/docs/shell/create-configure-gke-cluster) 文档。

- 要安装 EMQX Operator，请参考 [安装 EMQX Operator](../getting-started/getting-started.md)。

  ::: warning
  要在 Google Kubernetes Engine 上安装 `cert-manager`，请参阅官方文档：

  - [GKE Autopilot](https://cert-manager.io/docs/installation/compatibility/#gke-autopilot)
  - [Private GKE Cluster](https://cert-manager.io/docs/installation/compatibility/#gke)

  运行 `helm` 命令时，请记得使用 `--set installCRDs=true` 标志安装 CRD。

  更多信息请访问 [cert-manager](https://cert-manager.io)。
  :::

## 快速部署 EMQX 集群

以下是 EMQX 自定义资源的相关配置。您可以根据您希望部署的 EMQX 版本选择相应的 APIVersion。有关具体的兼容关系，请参阅 [EMQX Operator 兼容性](../index.md)：

:::: tabs type:card
::: tab apps.emqx.io/v2beta1

将以下内容保存为 YAML 文件，并使用 kubectl apply 命令进行部署。

```yaml
apiVersion: apps.emqx.io/v2beta1
kind: EMQX
metadata:
  name: emqx
spec:
  image: "emqx:5.1"
  coreTemplate:
    spec:
      volumeClaimTemplates:
      ## 关于存储类的更多信息：https://cloud.google.com/kubernetes-engine/docs/concepts/persistent-volumes#storageclasses
        storageClassName: standard
        resources:
          requests:
            storage: 10Gi
        accessModes:
        - ReadWriteOnce
  dashboardServiceTemplate:
    spec:
      ## 关于负载均衡器的更多信息：https://cloud.google.com/kubernetes-engine/docs/how-to/internal-load-balancing
      type: LoadBalancer
  listenersServiceTemplate:
    spec:
      ## 关于负载均衡器的更多信息：https://cloud.google.com/kubernetes-engine/docs/how-to/internal-load-balancing
      type: LoadBalancer
```

等待 EMQX 集群准备就绪。您可以使用 kubectl get 命令检查 EMQX 集群的状态。请确保状态为 Running，这可能需要一些时间。

```shell
$ kubectl get emqx
NAME   IMAGE      STATUS    AGE
emqx   emqx:5.1   Running   118s
```

获取 EMQX 集群的外部 IP 地址，并访问 EMQX 控制台。

EMQX Operator 将创建两个 EMQX Service 资源，一个是 emqx-dashboard，另一个是 emqx-listeners，分别对应 EMQX 控制台和 EMQX 监听端口。

```shell
$ kubectl get svc emqx-dashboard -o json | jq '.status.loadBalancer.ingress[0].ip'

34.122.174.166
```

通过在 Web 浏览器中打开 http://34.122.174.166:18083，访问 EMQX 控制台。使用默认的用户名和密码 admin/public 进行登录。

:::
::: tab apps.emqx.io/v1beta4

将以下内容保存为 YAML 文件，并使用 `kubectl apply` 命令进行部署。

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
      ## 关于存储类的更多信息：https://cloud.google.com/kubernetes-engine/docs/concepts/persistent-volumes#storageclasses
      storageClassName: standard
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
      ## 关于负载均衡器的更多信息：https://cloud.google.com/kubernetes-engine/docs/how-to/internal-load-balancing
      type: LoadBalancer
```

等待 EMQX 集群准备就绪。您可以使用 kubectl get 命令检查 EMQX 集群的状态。请确保状态为 Running，这可能需要一些时间。

```shell
$ kubectl get emqxenterprises
NAME      STATUS   AGE
emqx-ee   Running  8m33s
```

获取 EMQX 集群的外部 IP 地址，并访问 EMQX 控制台。

```shell
$ kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip'

34.68.80.122
```

通过在 Web 浏览器中打开 http://34.68.80.122:18083，访问 EMQX 控制台。使用默认的用户名和密码 admin/public 进行登录。

:::
::::

## 使用 MQTT X CLI 连接到 EMQX 集群发布/订阅消息

MQTT X CLI 是一个开源的 MQTT 5.0 命令行客户端工具，旨在帮助开发人员在没有 GUI 的情况下更快地开发和调试 MQTT 服务和应用程序。

- 获取 EMQX 集群的外部 IP 地址

    :::: tabs type:card
    ::: tab apps.emqx.io/v2beta1

    ```shell
    external_ip=$(kubectl get svc emqx-listeners -o json | jq '.status.loadBalancer.ingress[0].ip')
    ```

    :::
    ::: tab apps.emqx.io/v1beta4

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

- 在新的终端窗口中发送消息

    ```shell
    $ mqttx pub -t 'hello' -h ${external_ip} -p 1883 -m 'hello world'

    [10:00:58] › …  Connecting...
    [10:00:58] › ✔  Connected
    [10:00:58] › …  Message Publishing...
    [10:00:58] › ✔  Message published
    ```

- 在订阅的终端窗口中查看接收到的消息

    ```shell
    [10:00:58] › payload: hello world
    ```

## 使用 LoadBalancer 进行 TLS 终结

由于 Google LoadBalancer 不支持 TCP 证书，请参阅这个[文档](https://github.com/emqx/emqx-operator/discussions/312)解决 TCP 证书卸载问题。
