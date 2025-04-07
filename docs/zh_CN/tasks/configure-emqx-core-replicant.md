# 配置 Core + Replicant 集群 (EMQX 5.x)

## 任务目标

- 通过 `coreTemplate` 字段配置 EMQX 集群 Core 节点。
- 通过 `replicantTemplate` 字段配置 EMQX 集群 Replicant 节点。

## Core 节点与 Replicant 节点

::: tip
仅有 EMQX 企业版支持 Core + Replicant 节点集群。
:::

在 EMQX 5.0 中，EMQX 集群中的节点可以分成两个角色：核心（Core）节点和 复制（Replicant）节点。Core 节点负责集群中所有的写操作，与 EMQX 4.x 集群中的节点行为一致，作为 EMQX 数据库 [Mria](https://github.com/emqx/mria) 的真实数据源来存储路由表、会话、配置、报警以及 Dashboard 用户信息等数据。而 Replicant 节点被设计成无状态的，不参与数据的写入，添加或者删除 Replicant 节点不会改变集群数据的冗余。更多关于 EMQX 5.0 架构的信息请参考文档：[EMQX 5.0 架构](https://docs.emqx.com/zh/enterprise/v5.0/deploy/cluster/mria-introduction.html#mria-%E6%9E%B6%E6%9E%84%E4%BB%8B%E7%BB%8D)，Core 节点与 Replicant 节点的拓扑结构如下图所示：

  <div style="text-align:center">
  <img src="./assets/configure-core-replicant/mria-core-repliant.png" style="zoom:30%;" />
  </div>

::: tip
EMQX 集群中至少要有一个 Core 节点，出于高可用的目的，EMQX Operator 建议 EMQX 集群至少有三个 Core 节点。
:::

## 部署 EMQX 集群

`apps.emqx.io/v2beta1 EMQX` 支持通过 `.spec.coreTemplate` 字段来配置 EMQX 集群 Core 节点，使用 `.spec.replicantTemplate` 字段来配置 EMQX 集群 Replicant 节点，更多信息请查看：[API 参考](../reference/v2beta1-reference.md#emqxspec)。

+ 将下面的内容保存成 YAML 文件，并通过 `kubectl apply` 命令部署它

  ```yaml
  apiVersion: apps.emqx.io/v2beta1
  kind: EMQX
  metadata:
    name: emqx
  spec:
    image: emqx/emqx:latest
    coreTemplate:
      spec:
        replicas: 2
        resources:
          requests:
            cpu: 250m
            memory: 512Mi
    replicantTemplate:
      spec:
        replicas: 3
        resources:
          requests:
            cpu: 250m
            memory: 1Gi
    dashboardServiceTemplate:
      spec:
        type: LoadBalancer
  ```

  > 上文的 YAML 中，我们声明了这是一个由二个 Core 节点和三个 Replicant 节点组成的 EMQX 集群。Core 节点最低需要 512Mi 内存 ，Replicant 节点最低需要 1Gi 内存。你可以根据实际的业务负载进行调整。在实际业务中，Replicant 节点会接受全部的客户端请求，所以 Replicant 节点需要的资源会更高一些。

+ 等待 EMQX 集群就绪，可以通过 `kubectl get` 命令查看 EMQX 集群的状态，请确保 `STATUS` 为 `Running`，这个可能需要一些时间

  ```
  $ kubectl get emqx
  NAME   IMAGE              STATUS    AGE
  emqx   emqx/emqx:latest   Running   2m55s
  ```

+ 获取 EMQX 集群的 Dashboard External IP，访问 EMQX 控制台

  EMQX Operator 会创建两个 EMQX Service 资源，一个是 emqx-dashboard，一个是 emqx-listeners，分别对应 EMQX 控制台和 EMQX 监听端口。

  ```bash
  $ kubectl get svc emqx-dashboard -o json | jq '.status.loadBalancer.ingress[0].ip'

  192.168.1.200
  ```

  通过浏览器访问 `http://192.168.1.200:18083` ，使用默认的用户名和密码 `admin/public` 登录 EMQX 控制台。

## 检查 EMQX 集群

  可以通过检查 EMQX 自定义资源的状态来获取所有集群中节点的信息。

  ```bash
  $ kubectl get emqx emqx -o json | jq .status.coreNodes
  [
    {
      "node": "emqx@emqx-core-0.emqx-headless.default.svc.cluster.local",
      "node_status": "running",
      "otp_release": "24.3.4.2-2/12.3.2.2",
      "role": "core",
      "version": "5.0.20"
    },
    {
      "node": "emqx@emqx-core-1.emqx-headless.default.svc.cluster.local",
      "node_status": "running",
      "otp_release": "24.3.4.2-2/12.3.2.2",
      "role": "core",
      "version": "5.0.20"
    },
     {
      "node": "emqx@emqx-core-2.emqx-headless.default.svc.cluster.local",
      "node_status": "running",
      "otp_release": "24.3.4.2-2/12.3.2.2",
      "role": "core",
      "version": "5.0.20"
    }
  ]
  ```


  ```bash
  $ kubectl get emqx emqx -o json | jq .status.replicantNodes
  [
    {
      "node": "emqx@10.244.4.56",
      "node_status": "running",
      "otp_release": "24.3.4.2-2/12.3.2.2",
      "role": "replicant",
      "version": "5.0.20"
    },
    {
      "node": "emqx@10.244.4.57",
      "node_status": "running",
      "otp_release": "24.3.4.2-2/12.3.2.2",
      "role": "replicant",
      "version": "5.0.20"
    },
    {
      "node": "emqx@10.244.4.58",
      "node_status": "running",
      "otp_release": "24.3.4.2-2/12.3.2.2",
      "role": "replicant",
      "version": "5.0.20"
    }
  ]
  ```
