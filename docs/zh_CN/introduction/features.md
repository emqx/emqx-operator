# 核心功能

- 快速部署 EMQX Broker/Enterprise 集群，无繁琐配置
- 根据定义的变动滚动更新 EMQX Broker/Enterprise 集群
- EMQX Broker/Enterprise 的自动化伸缩
- 无需中断服务，即可更新 EMQX Broker/Enterprise 版本
- 集群和节点的监控，可以与 Prometheus 集成
- 集成了官方 EMQX Broker/Enterprise 容器镜像
- 自动发现 EMQX Listeners，并将监听器绑定到相应的 Kubernetes Service 资源
- 更新EMQX 配置(如: port、plugins等)， Pod 不重启
- 新的无状态节点支持,EMQX Replicant 使用 Deployment 进行部署(EMQX Operator >= 2.0)