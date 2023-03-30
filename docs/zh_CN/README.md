## EMQX Operator简介

EMQX Broker/Enterprise 是一个云原生的 MQTT 消息中间件。 我们提供了 EMQX Kubernetes Operator 来帮助您在 Kubernetes 的环境上快速创建和管理 EMQX Broker/Enterprise 集群。 它可以大大简化部署和管理 EMQX 集群的流程，对于管理和配置的知识要求也更低。它把部署和管理的工作变成一种低成本的、标准化的、可重复性的能力。

**注意：** 每个 Kubernetes 集群中只能部署一个 EMQX Operator。

EMQX Operator 与 EMQX 版本的对应关系如下：

|      EMQX 版本         |      EMQX Operator 版本                           |     APIVersion    | 
|:----------------------:|:------------------------------------------------:|:-----------------:| 
| 4.3.x <= EMQX < 4.4    | 1.2.0（停止维护）                                  |  v1beta2          |
| 4.3.x <= EMQX < 4.4    | 1.2.1，1.2.2，1.2.3 （推荐）                       |  v1beta3          |
| 4.4.6 <= EMQX < 4.4.8  | 1.2.5                                            |  v1beta3          | 
| 4.4.8 <= EMQX < 4.4.14 | 1.2.6，1.2.7，1.2.8，2.0.0，2.0.1，2.0.2 （推荐）   |  v1beta3          |
| 4.4.14 <= EMQX         | 2.1.0                                            |  v1beta4          |
| 5.0.6 <= EMQX < 5.0.8  | 2.0.0，2.0.1（推荐）                               |  v2alpha1         |
| 5.0.8 <= EMQX < 5.0.14 | 2.0.2                                            |  v2alpha1         |
| 5.0.14 <= EMQX         | 2.1.0                                            |  v2alpha1         |

## Operator架构
![](./introduction/assets/architecture.png)
