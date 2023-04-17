# 在腾讯云上部署 EMQX 集群

腾讯云容器服务（Tencent Kubernetes Engine，TKE）基于原生 kubernetes 提供以容器为核心的、高度可扩展的高性能容器管理服务。腾讯云容器服务完全兼容原生 kubernetes API，为容器化的应用提供高效部署、资源调度、服务发现和动态伸缩等一系列完整功能，解决用户开发、测试及运维过程的环境一致性问题，提高了大规模容器集群管理的便捷性，帮助用户降低成本，提高效率。容器服务会对不同规格的托管集群收取相应的集群管理费用。在使用中创建的其他的云产品资源（CVM、CBS、CLB 等），将按照各自云产品的计费方式进行收费。

## 前提条件

在开始之前，我们需要开通腾讯云 TKE 及相关的服务，具体请参考[创建集群](https://cloud.tencent.com/document/product/457/32189)https://www.alibabacloud.com/help/zh/container-service-for-kubernetes/latest/quick-start-for-first-time-users)，本文假设您已经成功部署了一个可以访问的 Kubernetes 集群。

## 部署 EMQX Operator

EMQX Operator 安装参考：[部署 EMQX Operator](../getting-started/getting-started.md)

## 为 EMQX 集群配置持久化存储

EMQX Custom Resource 使用 StoreClass 来保存 EMQX 运行时的状态。在开始之前，您需要准备 StoreClass。集群管理员可使用 StorageClass 为容器服务集群定义不同的存储类型。您可通过 StorageClass 配合 PersistentVolumeClaim 动态创建需要的存储资源。使用 `kubectl get storeClass` 可以查看当前 Kubernetes 集群中的 StoreClass。关于更多存储的相关信息，请查看[存储管理](https://cloud.tencent.com/document/product/457/46962)

腾讯云容器服务已默认提供了多种类型的 StorageClass，本文以 `cbs` 为例，如果你想创建自己的 storeClass，请参考[StorageClass 管理云硬盘模板](https://cloud.tencent.com/document/product/457/44239)

下面是 EMQX Custom Resource 的相关配置，你可以根据希望部署的 EMQX 的版本来选择对应的 APIVersion，具体的兼容性关系，请参考[EMQX Operator 兼容性](../README.md):

:::: tabs type:card
::: tab v2alpha1

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
  name: emqx
spec:
  image: emqx:5.0
  coreTemplate:
    spec:
      volumeClaimTemplates:
        storageClassName: cbs
        resources:
          requests:
            storage: 20Mi
        accessModes:
        - ReadWriteOnce
```

:::
::: tab v1beta4

```yaml
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
      storageClassName: cbs
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
```

:::
::::

## 使用 LoadBalancer 访问 EMQX 集群

TkeServiceConfig 是腾讯云容器服务提供的自定义资源 CRD， 通过 TkeServiceConfig 能够帮助您更灵活的配置 LoadBalancer 类型的 Service ，及管理其中负载均衡的各种配置。负载均衡 CLB 的相关配置可参见 [TkeServiceConfig 介绍](https://cloud.tencent.com/document/product/457/41895)。

使用 TkeServiceConfig 能够帮您快速进行负载均衡器的配置。通过 Service 注解 `service.cloud.tencent.com/tke-service-config:<config-name>`，您可以指定目标配置并应用到 Service 中。

TkeServiceConfig 并不会帮您直接配置并修改协议和端口，您需要在配置中描述协议和端口以便指定配置下发的监听器。在一个 TkeServiceConfig 中可以声明多组监听器配置，目前主要针对负载均衡的健康检查以及对后端访问提供配置。 通过指定协议和端口，配置能够被准确的下发到对应监听器：

`spec.loadBalancer.l4Listeners.protocol`：四层协议

`spec.loadBalancer.l4Listeners.port`：监听端口

创建 Loadbalancer 模式 Service 时，设置注解 `service.cloud.tencent.com/tke-service-config-auto: "true"`，将自动创建 \<ServiceName>-auto-service-config。您也可以通过 **service.cloud.tencent.com/tke-service-config:\<config-name>** 直接指定您自行创建的 TkeServiceConfig。两个注解不可同时使用。

除了 TkeServiceConfig，您可以通过其他 Annotation 注解配置 Service，以实现更丰富的负载均衡的能力。详情请查看[Service Annotation 说明](https://cloud.tencent.com/document/product/457/51258)

:::: tabs type:card
::: tab v2alpha1

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
  name: emqx
  annotations:
    service.cloud.tencent.com/tke-service-config-auto: "true"
    # 自动创建 tke-service-config
    # service.cloud.tencent.com/tke-service-config: emqx-service-config
    # 指定已有的 tke-service-config
spec:
  image: emqx:5.0
  listenersServiceTemplate:
    spec:
      type: LoadBalancer
```

:::
::: tab v1beta4

```yaml
apiVersion: apps.emqx.io/v1beta4
kind: EmqxEnterprise
metadata:
  name: emqx-ee
  annotations:
    service.cloud.tencent.com/tke-service-config-auto: "true"
    # 自动创建 tke-service-config
    # service.cloud.tencent.com/tke-service-config: emqx-ee-service-config
    # 指定已有的 tke-service-config
spec:
  template:
    spec:
      emqxContainer:
        image:
          repository: emqx/emqx-ee
          version: 4.4.14
  serviceTemplate:
    spec:
      type: LoadBalancer
```

:::
::::

## 使用 LoadBalancer 直连 Pod 模式 Service

原生 LoadBalancer 模式 Service 可自动创建负载均衡 CLB，并通过集群的 Nodeport 转发至集群内，再通过 iptable 或 ipvs 进行二次转发。该模式下的 Service 能满足大部分使用场景 ，但在以下场景中更推荐使用**直连 Pod 模式 Service**：

+ 有获取来源 IP 需求时（非直连模式必须另外开启 Local 转发）。

+ 要求具备更高转发性能时（非直连模式下 CLB 和 Service 本身存在两层 CLB，性能有一定损失）。

+ 需使用完整的健康检查和会话保持到 Pod 层级时（非直连模式下 CLB 和 Service 本身存在两层 CLB，健康检查及会话保持功能较难配置）。

直连 Pod 模式 Service 的 YAML 配置与普通 Service YAML 配置相同，示例中的 annotation 即代表是否开启直连 Pod 模式。

::: tip
需要在 `kube-system/tke-service-controller-config` ConfigMap 中新增 `GlobalRouteDirectAccess: "true"` 以开启 GlobalRoute 直连能力。
:::

:::: tabs type:card
::: tab v2alpha1

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
  name: emqx
  annotations:
    service.cloud.tencent.com/tke-service-config-auto: "true"
   	service.cloud.tencent.com/tke-service-config: emqx-service-config
spec:
  image: emqx:5.0
  listenersServiceTemplate:
    spec:
      type: LoadBalancer
```

:::
::: tab v1beta4

```yaml
apiVersion: apps.emqx.io/v1beta4
kind: EmqxEnterprise
metadata:
  name: emqx-ee
  annotations:
    service.cloud.tencent.com/direct-access: "true" ##开启直连 Pod 模式
   	service.cloud.tencent.com/tke-service-config: emqx-ee-service-config
spec:
  template:
    spec:
      emqxContainer:
        image:
          repository: emqx/emqx-ee
          version: 4.4.14
  serviceTemplate:
    spec:
      type: LoadBalancer
```

:::
::::

## 使用 LB 终结 TCP TLS 方案

目前腾讯云 CLB 不支持终结 TCP TLS ，如需要使用 LB 终结 TCP TLS 请参考[LB 终结 TCP TLS 方案](https://github.com/emqx/emqx-operator/discussions/312)





