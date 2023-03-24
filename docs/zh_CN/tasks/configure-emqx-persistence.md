# 配置 EMQX 持久化

## 任务目标

- 如何通过 persistent 字段 配置 EMQX 4.x 集群持久化。
- 如何通过 volumeClaimTemplates 字段配置 EMQX 5.x 集群 Core 节点持久化。

## EMQX 集群持久化配置

- 配置 EMQX 集群

:::: tabs type:card
::: tab v2alpha1

在 EMQX 5.0 中，EMQX 集群中的节点可以分成两个角色：核心（Core）节点和 复制（Replicant）节点。Core 节点负责集群中所有的写操作，作为 EMQX 数据库 [Mria](https://github.com/emqx/mria) 的真实数据源来存储路由表、会话、配置、报警以及Dashboard 用户信息等数据。而 Replicant 节点被设计成无状态的，不参与数据的写入，添加或者删除 Replicant 节点不会改变集群数据的冗余。因此在 EMQX CRD 里面，我们仅支持 Core 节点的持久化。

EMQX CRD 支持通过 `.spec.coreTemplate.spec.volumeClaimTemplates` 字段配置 EMQX 集群 Core 节点持久化。`.spec.coreTemplate.spec.volumeClaimTemplates` 字段的语义及配置与 Kubernetes 的 `PersistentVolumeClaimSpec` 一致，其配置可以参考文档：[PersistentVolumeClaimSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#persistentvolumeclaimspec-v1-core) 。

当用户配置了 `.spec.coreTemplate.spec.volumeClaimTemplates` 字段时，EMQX Operator 会为 EMQX 集群中的 每一个 Core 节点创建一个固定的 PVC（PersistentVolumeClaim）来表示用户对持久化的请求。当 Pod 被删除时，其对应的 PVC 不会自动清除。当 Pod 被重建时，会自动和已存在的 PVC 进行匹配。如果不想再使用旧集群的数据，需要手动清理 PVC。

PVC 表达的是用户对持久化的请求，而负责存储的则是持久卷（[PersistentVolume](https://kubernetes.io/zh-cn/docs/concepts/storage/persistent-volumes/)，PV），PVC 和 PV 通过 PVC Name 一对一绑定。PV 是集群中的一块存储，可以根据需求手动制备，也可以使用存储类（[StorageClass](https://kubernetes.io/zh-cn/docs/concepts/storage/storage-classes/))来动态制备。当用户不再使用 PV 资源时，可以手动删除 PVC 对象，从而允许该 PV 资源被回收再利用。目前，PV 的回收策略有两种：Retained（保留）和 Deleted（删除），其回收策略细节可以参考文档：[Reclaiming](https://kubernetes.io/zh-cn/docs/concepts/storage/persistent-volumes/#reclaiming)。

EMQX Operator 使用 PV 持久化 EMQX 集群 Core 节点 `/opt/emqx/data` 目录下的数据。EMQX Core 节点 `/opt/emqx/data` 目录存放的数据主要包含：路由表、会话、配置、报警以及Dashboard 用户信息等数据。

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
  name: emqx
spec:
  image: emqx/emqx:5.0.14
  imagePullPolicy: IfNotPresent
  coreTemplate:
    spec:
      volumeClaimTemplates:
        storageClassName: standard
        resources:
          requests:
            storage: 20Mi
        accessModes:
        - ReadWriteOnce
      replicas: 3
  replicantTemplate:
    spec:
      replicas: 0
  dashboardServiceTemplate:
    spec:
      type: NodePort
      ports:
        - name: "dashboard-listeners-http-bind"
          protocol: TCP
          port: 18083
          targetPort: 18083
          nodePort: 32016
```

> `storageClassName` 字段表示 StorageClass 的名称，可以使用命令 `kubectl get storageclass` 获取 Kubernetes 集群已经存在的 StorageClass，也可以根据自己需求自行创建 StorageClass。accessModes 字段表示 PV 的访问模式，默认使用 `ReadWriteOnce` 模式，更多访问模式可以参考文档：[AccessModes](https://kubernetes.io/zh-cn/docs/concepts/storage/persistent-volumes/#access-modes)。`.spec.dashboardServiceTemplate` 字段配置了 EMQX 集群对外暴露服务的方式为：NodePort，并指定了 EMQX Dashboard 服务 18083 端口对应的 nodePort 为 32016（nodePort 取值范围为：30000-32767)。

:::
::: tab v1beta4

EMQX CRD 支持通过 `.spec.persistent` 字段配置 EMQX 集群持久化。`.spec.persistent` 字段的语义及配置与 Kubernetes 的 `PersistentVolumeClaimSpec` 一致，其配置可以参考文档：[PersistentVolumeClaimSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#persistentvolumeclaimspec-v1-core)。

当用户配置了 `.spec.persistent` 字段时，EMQX Operator 会为 EMQX 集群中的 每一个 Pod 创建一个固定的 PVC（PersistentVolumeClaim）来表示用户对持久化的请求。当 Pod 被删除时，其对应的 PVC 不会自动清除。当 Pod 被重建时，会自动和已存在的 PVC 进行匹配。如果不想再使用旧集群的数据，需要手动清理 PVC。

PVC 表达的是用户对持久化的请求，而负责存储的则是持久卷（[PersistentVolume](https://kubernetes.io/zh-cn/docs/concepts/storage/persistent-volumes/)，PV），PVC 和 PV 通过 PVC Name 一对一绑定。PV 是集群中的一块存储，可以根据需求手动制备，也可以使用存储类（[StorageClass](https://kubernetes.io/zh-cn/docs/concepts/storage/storage-classes/))来动态制备。当用户不再使用 PV 资源时，可以手动删除 PVC 对象，从而允许该 PV 资源被回收再利用。目前，PV 的回收策略有两种：Retained（保留）和 Deleted（删除），其回收策略细节可以参考文档：[Reclaiming](https://kubernetes.io/zh-cn/docs/concepts/storage/persistent-volumes/#reclaiming)。

EMQX Operator 使用 PV 持久化 EMQX 节点 `/opt/emqx/data` 目录下的数据。EMQX 节点 `/opt/emqx/data` 目录存放的数据主要包含：loaded_plugins（已加载的插件信息），loaded_modules（已加载的模块信息），mnesia 数据库数据（存储 EMQX 自身运行数据，例如告警记录、规则引擎已创建的资源和规则、Dashboard 用户信息等数据）。

``` yaml
apiVersion: apps.emqx.io/v1beta4
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  persistent:
    metadata:
      name: emqx-ee
      labels:
        "apps.emqx.io/instance": "emqx-ee"
    spec:
      storageClassName: standard
      resources:
        requests:
          storage: 20Mi
      accessModes:
        - ReadWriteOnce
  template:
    spec:
      emqxContainer:
        image: 
          repository: emqx/emqx-ee
          version: 4.4.14
  serviceTemplate:
    spec:
      type: NodePort
      ports:
        - name: "http-dashboard-18083"
          protocol: "TCP"
          port: 18083
          targetPort: 18083
          nodePort: 32016
```

> `storageClassName` 字段表示 StorageClass 的名称，可以使用命令 `kubectl get storageclass` 获取 Kubernetes 集群已经存在的 StorageClass，也可以根据自己需求自行创建 StorageClass。accessModes 字段表示 PV 的访问模式，默认使用 `ReadWriteOnce` 模式，更多访问模式可以参考文档：[AccessModes](https://kubernetes.io/zh-cn/docs/concepts/storage/persistent-volumes/#access-modes)。`.spec.serviceTemplate` 字段配置了 EMQX 集群对外暴露服务的方式为：NodePort，并指定了 EMQX Dashboard 服务 18083 端口对应的 nodePort 为 32016（nodePort 取值范围为：30000-32767)。

:::
::: tab v1beta3

EMQX CRD 支持通过 `.spec.persistent` 字段配置 EMQX 集群持久化。`.spec.persistent` 字段的语义及配置与 Kubernetes 的 `PersistentVolumeClaimSpec` 一致，其配置可以参考文档：[PersistentVolumeClaimSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#persistentvolumeclaimspec-v1-core)。

当用户配置了 `.spec.persistent` 字段时，EMQX Operator 会为 EMQX 集群中的 每一个 Pod 创建一个固定的 PVC（PersistentVolumeClaim）来表示用户对持久化的请求。当 Pod 被删除时，其对应的 PVC 不会自动清除。当 Pod 被重建时，会自动和已存在的 PVC 进行匹配。如果不想再使用旧集群的数据，需要手动清理 PVC。

PVC 表达的是用户对持久化的请求，而负责存储的则是持久卷（[PersistentVolume](https://kubernetes.io/zh-cn/docs/concepts/storage/persistent-volumes/)，PV），PVC 和 PV 通过 PVC Name 一对一绑定。PV 是集群中的一块存储，可以根据需求手动制备，也可以使用存储类（[StorageClass](https://kubernetes.io/zh-cn/docs/concepts/storage/storage-classes/))来动态制备。当用户不再使用 PV 资源时，可以手动删除 PVC 对象，从而允许该 PV 资源被回收再利用。目前，PV 的回收策略有两种：Retained（保留）和 Deleted（删除），其回收策略细节可以参考文档：[Reclaiming](https://kubernetes.io/zh-cn/docs/concepts/storage/persistent-volumes/#reclaiming)。

EMQX Operator 使用 PV 持久化 EMQX 节点 `/opt/emqx/data` 目录下的数据。EMQX 节点 `/opt/emqx/data` 目录存放的数据主要包含：loaded_plugins（已加载的插件信息），loaded_modules（已加载的模块信息），mnesia 数据库数据（存储 EMQX 自身运行数据，例如告警记录、规则引擎已创建的资源和规则、Dashboard 用户信息等数据）。

``` yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  persistent:
    storageClassName: standard
    resources:
      requests:
        storage: 20Mi
    accessModes:
    - ReadWriteOnce
  emqxTemplate:
    image: emqx/emqx-ee:4.4.14
    serviceTemplate:
      spec:
        type: NodePort
        ports:
          - name: "http-dashboard-18083"
            protocol: "TCP"
            port: 18083
            targetPort: 18083
            nodePort: 32016
```
> `storageClassName` 字段表示 StorageClass 的名称，可以使用命令 `kubectl get storageclass` 获取 Kubernetes 集群已经存在的 StorageClass，也可以根据自己需求自行创建 StorageClass。accessModes 字段表示 PV 的访问模式，默认使用 `ReadWriteOnce` 模式，更多访问模式可以参考文档：[AccessModes](https://kubernetes.io/zh-cn/docs/concepts/storage/persistent-volumes/#access-modes)。`.spec.emqxTemplate.serviceTemplate` 字段配置了 EMQX 集群对外暴露服务的方式为：NodePort，并指定了 EMQX Dashboard 服务 18083 端口对应的 nodePort 为 32016（nodePort 取值范围为：30000-32767)。

:::
::::

将上述内容保存为：emqx-persistent.yaml，执行如下命令部署 EMQX 集群：

```bash
kubectl apply -f emqx-persistent.yaml
```

输出类似于：

```
emqx.apps.emqx.io/emqx created
```

- 检查 EMQX 集群是否就绪

:::: tabs type:card
::: tab v2alpha1

```bash
 kubectl get emqx emqx -o json | jq '.status.conditions[] | select( .type == "Running" and .status == "True")'
```

输出类似于：

```bash
{
  "lastTransitionTime": "2023-03-01T02:17:03Z",
  "lastUpdateTime": "2023-03-01T02:17:03Z",
  "message": "Cluster is running",
  "reason": "ClusterRunning",
  "status": "True",
  "type": "Running"
}
```

:::
::: tab v1beta4

```
kubectl get emqxEnterprise emqx-ee -o json | jq '.status.conditions[] | select( .type == "Running" and .status == "True")'
```

输出类似于：

```bash
{
  "lastTransitionTime": "2023-03-01T02:49:22Z",
  "lastUpdateTime": "2023-03-01T02:49:23Z",
  "message": "All resources are ready",
  "reason": "ClusterReady",
  "status": "True",
  "type": "Running"
}
```

:::
::: tab v1beta3

```bash
kubectl get emqxEnterprise emqx-ee -o json | jq '.status.conditions[] | select( .type == "Running" and .status == "True")'
```

输出类似于：

```bash
{
  "lastTransitionTime": "2023-03-01T02:49:22Z",
  "lastUpdateTime": "2023-03-01T02:49:23Z",
  "message": "All resources are ready",
  "reason": "ClusterReady",
  "status": "True",
  "type": "Running"
}
```

:::
::::

## 验证 EMQX 集群持久化是否生效

验证方案： 1）在旧 EMQX 集群中通过 Dashboard 创建一条测试规则；2）删除旧集群；3） 重新创建 EMQX 集群，通过 Dashboard 查看之前创建的规则是否存在。

- 通过 Dashboard 创建测试规则

打开浏览器，输入 EMQX Pod 所在宿主机 `IP` 和 端口 `32016` 来登录 EMQX 集群 Dashboard（Dashboard 默认用户名为：admin ，默认密码为：public），进入 Dashboard 点击 数据集成 → 规则 进入创建规则的页面，我们先点击添加动作的按钮为这条规则添加响应动作，然后点击创建生成规则，如下图所示：

![](./assets/configure-emqx-persistent/emqx-core-action.png)

当我们的规则创建成功之后，在页面会出现一条规则记录，规则 ID 为：emqx-persistent-test，如下图所示：

![](./assets/configure-emqx-persistent/emqx-core-rule-old.png)

- 删除旧 EMQX 集群

执行如下命令删除 EMQX 集群：

```bash
kubectl delete -f  emqx-persistent.yaml
```

> emqx-persistent.yaml 是本文中第一次部署 EMQX 集群所使用的 YAML 文件，这个文件不需要做任何的改动。

输出类似于：

```
emqx.apps.emqx.io "emqx" deleted
```

执行如下命令查看 EMQX 集群是否被删除：

```bash
kubectl get emqx emqx -o json | jq ".status.emqxNodes"
```

输出类似于：

```
Error from server (NotFound): emqxes.apps.emqx.io "emqx" not found
```

- 重新创建 EMQX 集群

执行如下命令重新创建 EMQX 集群：

```bash
kubectl apply -f  emqx-persistent.yaml
```

输出类似于：

```
emqx.apps.emqx.io/emqx created
```

接下来执行如下命令查看 EMQX 集群是否就绪：

```bash
kubectl get emqx emqx -o json | jq '.status.conditions[] | select( .type == "Running" and .status == "True")
```

输出类似于：

```bash
{
  "lastTransitionTime": "2023-03-01T02:17:03Z",
  "lastUpdateTime": "2023-03-01T02:17:03Z",
  "message": "Cluster is running",
  "reason": "ClusterRunning",
  "status": "True",
  "type": "Running"
}
```

最后通过浏览器访问 EMQX Dashboard 查看之前创建的规则是否存在，如下如图所示：

![](./assets/configure-emqx-persistent/emqx-core-rule-new.png)

从图中可以看出：在旧集群中创建的规则 emqx-persistent-test 在新的集群中依旧存在，则说明我们配置的持久化是生效的。