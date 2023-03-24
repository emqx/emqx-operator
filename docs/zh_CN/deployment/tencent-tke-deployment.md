# 使用 EMQX Operator 在腾讯云 TKE 上部署 EMQX 集群

## 名词解释

EMQX：The most scalable open-source MQTT broker for IoT，[EMQX 文档](https://github.com/emqx/emqx)

EMQX Operator：A Kubernetes Operator for EMQX，[EMQX Operator 文档](https://github.com/emqx/emqx-operator)

TKE：腾讯云容器服务（Tencent Kubernetes Engine，TKE）基于原生 kubernetes 提供以容器为核心的、高度可扩展的高性能容器管理服务。[TKE 文档](https://cloud.tencent.com/document/product/457)

CLB：负载均衡（Cloud Load Balancer， CLB），[CLB 文档](https://cloud.tencent.com/document/product/214/8975)

## 创建 TEK 集群

登录腾讯云, 选择云产品  -> 容器服务，点击创建， 选择标准集群，具体创建步骤参考：[云厂商文档](https://cloud.tencent.com/document/product/457/32189)

## LoadBalancer 配置

非直连模式下 CLB 和 Service 本身存在两层 CLB，性能有一定损失，开启直连配置可以提升转发性能。只需要在服务的 Service 的 annotations 里面添加以下注解： [直连模式配置说明](https://cloud.tencent.com/document/product/457/41897)

```yaml
service.cloud.tencent.com/direct-access: "true" 
```

**备注**: 开启直连模式需要在 kube-system/tke-service-controller-config ConfigMap 中新增 GlobalRouteDirectAccess: "true" 以开启 GlobalRoute 直连能力。

## 创建 StorageClass

点击集群名称进入集群详情页面，点击存储 -> StorageClass 创建需要的 StorageClass, 具体步骤参考：[创建StorageClass](https://console.cloud.tencent.com/tke2/cluster/sub/create/storage/sc?rid=16&clusterId=cls-mm0it4nz)

## 使用 EMQX Operator 部署 EQMX 集群

EMQX Operator 安装参考：[EMQX Operator 安装](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/getting-started/getting-started.md)

EMQX Operator 安装完成后，使用以下命令在 TKE 上进行部署 EMQX 集群：

:::: tabs type:card 
::: tab v2alpha1

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
  name: emqx
  annotations:
    ##开启LB 直连 Pod 模式
    service.cloud.tencent.com/direct-access: "true"
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
      type: LoadBalancer
      ports:
        - name: "dashboard-listeners-http-bind"
          protocol: TCP
          port: 18083
          targetPort: 18083
```
:::
::: tab v1beta4

```yaml
apiVersion: apps.emqx.io/v1beta4
kind: EmqxEnterprise
metadata:
  name: emqx-ee
  annotations:
    ##开启LB 直连 Pod 模式
    service.cloud.tencent.com/direct-access: "true"
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
      type: LoadBalancer
      ports:
        - name: "http-dashboard-18083"
          protocol: "TCP"
          port: 18083
          targetPort: 18083
```
:::
::: tab v1beta3

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
  name: emqx-ee
  annotations:
    ##开启LB 直连 Pod 模式
    service.cloud.tencent.com/direct-access: "true"
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
         type: LoadBalancer
         ports:
           - name: "http-dashboard-18083"
             protocol: "TCP"
             port: 18083
             targetPort: 18083
```
::: 
::::

## 使用 LB 终结 TCP TLS 方案

目前腾讯云 CLB 支持 终结 TCP TLS ，如需要使用 LB 终结 TCP TLS 请参考这篇文档，[LB 终结 TCP TLS 方案](https://github.com/emqx/emqx-operator/discussions/312)





