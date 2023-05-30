# 管理 EMQX 集群

本章提供了在 Kubernetes 集群中使用 EMQX 执行常见任务和操作的分步说明。

本章分为几个部分，涵盖:

**配置和设置**

- License 文件和安全性
  - [License 配置 (EMQX 企业版)](./configure-emqx-license.md)
  - [在 EMQX 中开启 TLS](./configure-emqx-tls.md)
- 集群配置
  - [通过 EMQX Operator 修改 EMQX 配置](./configure-emqx-bootstrapConfig.md)
  - [开启 Core + Replicant 集群 (EMQX 5.x)](./configure-emqx-core-replicant.md)
  - [在 EMQX 集群中开启持久化](./configure-emqx-persistence.md)
  - [通过 Kubernetes Service 访问 EMQX 集群](./configure-emqx-service.md)
  - [集群负载重平衡（EMQX 企业版）](./configure-emqx-rebalance.md)

**升级和维护**

- 升级
  - [配置蓝绿发布 (EMQX 企业版)](./configure-emqx-blueGreenUpdate.md)
- 日志管理
  - [在 Kubernetes 中采集 EMQX 的日志](./configure-emqx-log-collection.md)
  - [修改 EMQX 日志等级](./configure-emqx-log-level.md)

**监控和性能**

- [通过 Prometheus 监控 EMQX 集群](./configure-emqx-prometheus.md)

