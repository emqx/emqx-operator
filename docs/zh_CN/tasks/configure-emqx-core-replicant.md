# 开启 Core + Replicant 集群 (EMQX 5.x)

## 任务目标

- 如何通过 coreTemplate 字段配置 EMQX 集群 Core 节点。
- 如何通过 replicantTemplate 字段配置 EMQX 集群 Replicant 节点。

## Core 节点与 Replicant 节点

在 EMQX 5.0 中，为了实现集群横向扩展能力，可以将集群中的 EMQX 节点分成两个角色：核心（Core）节点和 复制（Replicant）节点。其拓扑结构如下图所示：

 <img src="./assets/configure-core-replicant/mria-core-repliant.png" style="zoom:50%;" />

Core 节点的行为与 EMQX 4.x 中节点一致：Core 节点使用全连接的方式组成集群，每个节点都可以发起事务、持有锁等。因此，EMQX 5.0 仍然要求 Core 节点在部署上要尽量的可靠。

Replicant 节点不再直接参与事务的处理。但它们会连接到 Core 节点，并被动地复制来自 Core 节点的数据更新。Replicant 节点不允许执行任何的写操作。而是将其转交给 Core 节点代为执行。另外，由于 Replicant 会复制来自 Core 节点的数据，所以它们有一份完整的本地数据副本，以达到最高的读操作的效率，这样有助于降低 EMQX 路由的时延。另外，Replicant 节点被设计成是无状态的，添加或删除它们不会导致集群数据的丢失、也不会影响其他节点的服务状态，所以 Replicant 节点可以被放在一个自动扩展组中。


::: tip
EMQX 集群中至少要有一个 Core 节点，出于高可用的目的，EMQX Operator 要求 EMQX 集群至少有两个 Core 节点和两个 Replicant 节点。
:::

## 部署 EMQX 集群

EMQX CRD 支持使用 `.spec.coreTemplate` 字段来配置 EMQX 集群 Core 节点，coreTemplate 字段的具体描述可以参考：[coreTemplate](https://github.com/emqx/emqx-operator/blob/2.0.2/docs/en_US/reference/v2alpha1-reference.md#emqxcoretemplate)。使用 `.spec.replicantTemplate` 字段来配置 EMQX 集群 Replicant 节点，replicantTemplate 字段的具体描述可以参考：[emqxreplicanttemplate](https://github.com/emqx/emqx-operator/blob/2.0.2/docs/en_US/reference/v2alpha1-reference.md#emqxreplicanttemplate)。

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
  name: emqx
spec:
  image: emqx:5.0
  coreTemplate:
    spec:
      replicas: 2
  replicantTemplate:
    spec:
      replicas: 3
```

将上述内容保存为：emqx-core.yaml，并执行如下命令部署 EMQX 集群：

```bash
$ kubectl apply -f emqx.yaml

emqx.apps.emqx.io/emqx created
```

检查 EMQX 集群状态，请确保 `STATUS` 为 `Running`，这可能需要一些时间等待 EMQX 集群准备就绪。

```bash
$ kubectl get emqx emqx

NAME   IMAGE      STATUS    AGE
emqx   emqx:5.0   Running   10m
```

## 检查 EMQX 集群

可以通过检查 EMQX 自定义资源的状态来获取所有集群中节点的信息，节点的 `role` 字段表示它们在集群中的角色，在上文中部署了一个由两个 Core 节点与三个 Replicant 节点组成的集群。

```bash
$ kubectl get emqx emqx -o json | jq .status.emqxNodes
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
  },
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
  }
]
```
