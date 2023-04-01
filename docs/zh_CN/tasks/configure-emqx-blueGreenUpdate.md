# 配置蓝绿发布（EMQX 企业版）

## 任务目标

如何通过蓝绿发布优雅的升级 EMQX 集群

## 为什么需要蓝绿发布

EMQX 提供的是长连接服务，在 Kubernets 中，现有升级策略除了热升级外，都需要重启 EMQX 服务，这种升级策略会导致设备出现断连，如果设备有重连机制，就会出现大量设备同时请求连接的情况，从而引发雪崩，最终导致大量客户端暂时得不到服务。因此 EMQX Operator 基于 EMQX 企业版的节点疏散（Node Evacuation）功能实现了蓝绿升级来解决上述问题。EMQX Operator 进行蓝绿升级的流程如下图所示：

![](./assets/configure-emqx-blueGreenUpdate/blueGreenUpdate.png)

EMQX 节点疏散功能用于疏散节点中的所有连接，手动/自动的将客户端连接和会话移动到集群中的其他节点或者其他集群。关于 EMQX 节点疏散的详细介绍可以参考文档：[Node Evacuation](https://docs.emqx.com/zh/enterprise/v4.4/advanced/rebalancing.html#%E8%8A%82%E7%82%B9%E7%96%8F%E6%95%A3) 。

:::tip 

节点疏散功能仅在 EMQX 企业版 4.4.12 版本才开放。

:::


## 如何使用蓝绿发布

EMQX 企业版在 EMQX Operator 里面对应的 CRD 为 EmqxEnterprise，EmqxEnterprise 支持通过 `.spec.blueGreenUpdate` 字段来配置 EMQX 企业版蓝绿升级，blueGreenUpdate 字段的具体描述可以参考：[blueGreenUpdate](https://github.com/emqx/emqx-operator/blob/main-2.1/docs/en_US/reference/v1beta4-reference.md#evacuationstrategy)。

```yaml
apiVersion: apps.emqx.io/v1beta4
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  blueGreenUpdate:
    initialDelaySeconds: 5
    evacuationStrategy:
      waitTakeover: 5
      connEvictRate: 10
      sessEvictRate: 10
  template:
    spec:
      emqxContainer:
        image: 
          repository: emqx/emqx-ee
          version: 4.4.14
```

> `waitTakeover` 表示当前节点开始 session 疏散之前等待的时间（单位为 second）。`connEvictRate` 表示当前节点客户端断开速率（单位为：count/second）。`sessEvictRate` 表示当前节点客户端 session 疏散速率（单位为：count/second）。`.spec.license.stringData` 字段填充的是 License 证书内容，在本文该字段的内容被省略，请用自己证书的内容进行填充。

将上述内容保存为：emqx-update.yaml，执行如下命令部署 EMQX 企业版集群：

```bash
$ kubectl apply -f emqx-update.yaml

emqxenterprise.apps.emqx.io/emqx-ee created
```

检查 EMQX 集群状态，请确保 `STATUS` 为 `Running`，这可能需要一些时间等待 EMQX 集群准备就绪。

   ```bash
$ kubectl get emqxenterprises

NAME      STATUS   AGE
emqx-ee   Running  8m33s
   ```

### 使用 MQTT X CLI 连接 EMQX 集群

MQTT X CLI 是开源一个的，支持自动重连的 MQTT 5.0 CLI Client，也是一个纯命令行模式的 MQTT X。旨在帮助更快地开发和调试 MQTT 服务和应用程序，而无需使用图形界面。关于 MQTT X CLI 的文档可以参考：[MQTTX CLI](https://mqttx.app/docs/cli)。

执行如下命令连接 EMQX 集群：

```bash
mqttx bench  conn -h 47.103.65.17  -p 32010   -c 3000
```

> `-h` 表示 EMQX Pod 所在宿主机 IP。`-p` 表示 nodePort 端口。`-c` 表示创建的连接数。本文在部署 EMQX 集群的时候采用的是 NodePort 模式暴露服务。如果采用 LoadBalancer 的方式暴露服务则 `-h` 应为 LoadBalancer 的 IP，`-p` 应为 EMQX MQTT 服务端口。

输出类似于：

```bash
[10:05:21 AM] › ℹ  Start the connect benchmarking, connections: 3000, req interval: 10ms
✔  success   [3000/3000] - Connected
[10:06:13 AM] › ℹ  Done, total time: 31.113s
```

### 触发 EMQX Operator 进行蓝绿升级

修改 EmqxEnterprise 对象 `.spec.template` 字段的任意内容都会触发 EMQX Operator 进行蓝绿升级。在本文中通过我们修改 EMQX Container Name 来触发升级，用户可根据实际需求自行修改。

```bash
$ kubectl patch EmqxEnterprise emqx-ee --type='merge' -p '{"spec": {"template": {"spec": {"emqxContainer": {"emqxConfig": {"image": {"version": "4.4.15"}}}}}}}'

emqxenterprise.apps.emqx.io/emqx-ee patched
```

检查蓝绿升级的状态

```bash
$ kubectl get emqxEnterprise emqx-ee -o json | jq ".status.blueGreenUpdateStatus.evacuationsStatus"

[
  {
    "connection_eviction_rate": 10,
    "node": "emqx-ee@emqx-ee-54fc496fb4-2.emqx-ee-headless.default.svc.cluster.local",
    "session_eviction_rate": 10,
    "session_goal": 0,
    "connection_goal": 22,
    "session_recipients": [
      "emqx-ee@emqx-ee-5d87d4c6bd-2.emqx-ee-headless.default.svc.cluster.local",
      "emqx-ee@emqx-ee-5d87d4c6bd-1.emqx-ee-headless.default.svc.cluster.local",
      "emqx-ee@emqx-ee-5d87d4c6bd-0.emqx-ee-headless.default.svc.cluster.local"
    ],
    "state": "waiting_takeover",
    "stats": {
      "current_connected": 0,
      "current_sessions": 0,
      "initial_connected": 33,
      "initial_sessions": 0
    }
  }
]
```

> `connection_eviction_rate` 表示节点疏散速率（单位：count/second）。`node` 表示当前正在进行疏散的节点。`session_eviction_rate` 表示节点 session 疏散速率(单位：count/second)。`session_recipients` 表示 session 疏散的接受者列表。`state` 表示节点疏散阶段。`stats` 表示疏散节点的统计指标，包括当前连接数（current_connected），当前 session 数（current_sessions），初始连接数（initial_connected），初始 session 数（initial_sessions）。

### 在升级期间使用 Prometheus 监控客户端连接

使用浏览器访问 Prometheus Web 服务，点击 Graph，在收索框输入 `emqx_connections_count`，并点击 Execute，显示如下图所示：

![](./assets/configure-emqx-blueGreenUpdate/prometheus.png)

从图中可以看出存在新旧两个 EMQX 集群，每个集群都有三个 EMQX 节点。在开始进行蓝绿升级后，旧集群每个节点的连接按照配置的速率断开并迁移到新集群的节点上，最终旧集群中的所有连接完全迁移到新集群中，则代表蓝绿升级完成。

