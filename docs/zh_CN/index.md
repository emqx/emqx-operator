## EMQX Operator 简介

EMQX Broker/Enterprise 是一个云原生的 MQTT 消息中间件。 我们提供了 EMQX Kubernetes Operator 来帮助您在 Kubernetes 的环境上快速创建和管理 EMQX Broker/Enterprise 集群。 它可以大大简化部署和管理 EMQX 集群的流程，对于管理和配置的知识要求也更低。它把部署和管理的工作变成一种低成本的、标准化的、可重复性的能力。

EMQX Operator 包括但不限于以下功能：

* **简化 EMQX 部署**：通过 EMQX 自定义资源声明 EMQX 集群，并快速的部署，更多的内容，请查看[快速开始](./getting-started/getting-started.md)。

* **管理 EMQX 集群**：对 EMQX 进行自动化运维操作，包括集群升级、运行时数据持久化、根据 EMQX 的状态更新 Kubernetes 的资源等，更多的内容，请查看[管理 EMQX 集群](./tasks/overview.md)。

<img src="./introduction/assets/architecture.png" style="zoom:20%;" />

## EMQX 与 EMQX Operator 的兼容性列表

### EMQX 企业版

|  EMQX 企业版 |              EMQX Operator Version              |                          APIVersion                          |      Kind      |
| :------------------------: | :---------------------------------------------: | :----------------------------------------------------------: | :------------: |
|  4.3.x （包含） ～ 4.4   |               1.2.1, 1.2.2, 1.2.3               | [apps.emqx.io/v1beta3](./reference/v1beta3-reference.md) | EmqxEnterprise |
| 4.4.6 （包含） ～ 4.4.8  |                      1.2.5                      | [apps.emqx.io/v1beta3](./reference/v1beta3-reference.md) | EmqxEnterprise |
| 4.4.8 （包含） ～ 4.4.14 | 1.2.6, 1.2.7, 1.2.8, 2.0.0, 2.0.1, 2.0.2, 2.0.3 | [apps.emqx.io/v1beta3](./reference/v1beta3-reference.md) | EmqxEnterprise |
|   4.4.14 （包含） 或更高 4.4.x   |                  2.1.0, 2.1.1                   | [apps.emqx.io/v1beta4](./reference/v1beta4-reference.md) | EmqxEnterprise |
|      5.0.0 （包含） ～ 5.0.23       |    2.0.0, 2.0.1, 2.0.2, 2.0.3, 2.1.0, 2.1.1     | [apps.emqx.io/v2alpha1](./reference/v2alpha1-reference.md) |      EMQX      |
|      5.1.1 或更高       |    2.2.0                                        | [apps.emqx.io/v2alpha2](./reference/v2alpha2-reference.md) |      EMQX      |

### EMQX 开源版

|      EMQX 开源版 |     EMQX Operator Version                            |     APIVersion    |    Kind    |
|------------------------|-------------------|-------------------|-------------------|
| 4.3.x （包含） ～ 4.4 | 1.2.1, 1.2.2, 1.2.3                                 |  [apps.emqx.io/v1beta3](./reference/v1beta3-reference.md)          |  EmqxBroker  |
| 4.4.6 （包含） ～ 4.4.8 | 1.2.5                                                 | [apps.emqx.io/v1beta3](./reference/v1beta3-reference.md)          | EmqxBroker |
| 4.4.8 （包含） ～ 4.4.14 | 1.2.6, 1.2.7, 1.2.8, 2.0.0, 2.0.1, 2.0.2, 2.0.3   |  [apps.emqx.io/v1beta3](./reference/v1beta3-reference.md)          | EmqxBroker |
| 4.4.14 或更高 4.4.x | 2.1.0, 2.1.1                                                 |  [apps.emqx.io/v1beta4](./reference/v1beta4-reference.md)          | EmqxBroker |
| 5.0.6 （包含） ～ 5.0.8 | 2.0.0, 2.0.1, 2.0.3                                |  [apps.emqx.io/v2alpha1](./reference/v2alpha1-reference.md)         |  EMQX     |
| 5.0.8 （包含） ～  5.0.14 | 2.0.2                                            |  [apps.emqx.io/v2alpha1](./reference/v2alpha1-reference.md)         |  EMQX     |
| 5.0.14 （包含） ～ 5.0.23 | 2.1.0, 2.1.1                                                | [apps.emqx.io/v2alpha1](./reference/v2alpha1-reference.md)         | EMQX     |
|      5.1.1 或更高       |    2.2.0                                        | [apps.emqx.io/v2alpha2](./reference/v2alpha2-reference.md) |      EMQX      |

## 如何选择 Kubernetes 版本

EMQX Operator 要求 Kubernetes 集群的版本号  `>=1.24`。

| Kubernetes 版本      | EMQX Operator 兼容性                                         | 注释                                                         |
| -------------------- | ------------------------------------------------------------ | ------------------------------------------------------------ |
| 1.24 更高            | 支持所有功能                                                 |                                                              |
| 1.22 ( 包含) ～ 1.23 | 支持，但是不包含 [MixedProtocolLBService](https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/) | EMQX 集群只能在 LoadBalancer 类型的 Service 中使用一个协议，例如 TCP 或 UDP。 |
| 1.21 ( 包含) ～ 1.22 | 支持，但是不包含 [Pod 删除开销](https://kubernetes.io/zh-cn/docs/concepts/workloads/controllers/replicaset/#pod-deletion-cost) | EMQX Core + Replicant 模式集群时，更新 EMQX 集群无法准确的删除 Pod。|
| 1.20 ( 包含) ～ 1.21 | 支持，但是如果使用 `NodePort`  类型的 Service，需要手动管理  `.spec.ports[].nodePort` | 更多的详情，请查看 [Kubernetes changelog](https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG/CHANGELOG-1.20.md#bug-or-regression-4). |
| 1.16 ( 包含) ～ 1.20 | 支持，但是不推荐，因为缺乏足够的测试                         |                                                              |
| 低于 1.16            | 不支持                                                       | 低于 1.16 版本的 Kubernetes 不支持 `apiextensions/v1` APIVersion。 |
