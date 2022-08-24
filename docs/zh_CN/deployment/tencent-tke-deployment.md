# 使用 EMQX Operator 在腾讯云 TKE 上部署EMQX 集群

## 名词解释

EMQX：The most scalable open-source MQTT broker for IoT，[EMQX 详见](https://github.com/emqx/emqx) 

EMQX Operator：A Kubernetes Operator for EMQX，[EMQX Operator 详见](https://github.com/emqx/emqx-operator)

TKE：腾讯云容器服务（Tencent Kubernetes Engine，TKE）基于原生 kubernetes 提供以容器为核心的、高度可扩展的高性能容器管理服务，[TKE 概述](https://cloud.tencent.com/document/product/457)

CLB：负载均衡（Cloud Load Balancer， CLB），[CLB 概述](https://cloud.tencent.com/document/product/214/8975)

## 创建 TEK 集群

登录腾讯云, 选择云产品 -> 容器服务，点击创建， 选择标准集群，EMQX Operator 要求Kubernetes 版本>=1.20.0，因此我们在此选择 Kubernetes 选择 1.22.5 ,网络与其他资源信息根据自身需求来制定。具体创建步骤参考： [创建标准集群](https://cloud.tencent.com/document/product/457/32189)  

## LoadBalancer 配置

非直连模式下 CLB 和 Service 存在两层 CLB，性能有一定损失，开启 CLB 直连 pod 可以提升请求转发性能，我们推荐使用直连模式。需要在Service 的 annotations 里面添加以下注解： [直连模式配置说明](https://cloud.tencent.com/document/product/457/41897)
```yaml
service.cloud.tencent.com/direct-access: "true" 
```

备注: 开启直连模式需要在 kube-system/tke-service-controller-config ConfigMap 中新增 GlobalRouteDirectAccess: "true" 以开启 GlobalRoute 直连能力。

## 创建 StorageClass

点击集群名称进入集群详情页面，点击存储 -> StorageClass 创建需要的StorageClass, 具体步骤参考[创建StorageClass](https://cloud.tencent.com/document/product/457/44232)

## 使用 EMQX Operator 部署 EQMX 集群

EMQX Operator 安装参考：[EMQX Operator 安装](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/getting-started/getting-started.md)

EMQX Operator 安装完成后，使用以下命令在 TKE 上进行部署 EMQX 集群：

```yaml
cat << EOF | kubectl apply -f -
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
  name: emqx-ee
  labels:
    "apps.emqx.io/instance": "emqx-ee"
  annotations:
    service.cloud.tencent.com/direct-access: "true" ##开启 CLB 直连 Pod
spec:
  emqxTemplate:
    image: emqx/emqx-ee:4.4.6
    serviceTemplate:
      metadata:
        name: emqx-ee
        namespace: default
        labels:
          "apps.emqx.io/instance": "emqx-ee"
      spec:
        type: LoadBalancer
        selector:
          "apps.emqx.io/instance": "emqx-ee"
  persistent:
    accessModes: 
      - ReadWriteOnce
    resources:
        requests:
          storage: 10Gi 
    storageClassName: emqx-test
EOF
```

使用 LB 终结 TCP mTLS 方案

目前腾讯云 CLB 支持 终结 TCP TLS ，如需要使用 LB 终结 TCP mTLS 请参考这篇文档：[LB 终结 TCP mTLS 方案](https://github.com/emqx/emqx-operator/discussions/312)
