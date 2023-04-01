# 在华为云上部署 EMQX 集群

华为云容器引擎（Cloud Container Engine，简称CCE）提供高度可扩展的、高性能的企业级Kubernetes集群，支持运行Docker容器。借助云容器引擎，您可以在华为云上轻松部署、管理和扩展容器化应用程序。

云容器引擎深度整合高性能的计算（ECS/BMS）、网络（VPC/EIP/ELB）、存储（EVS/OBS/SFS）等服务，并支持GPU、NPU、ARM等异构计算架构，支持多可用区（Available Zone，简称AZ）、多区域（Region）容灾等技术构建高可用Kubernetes集群。

华为云是全球首批Kubernetes认证服务提供商（Kubernetes Certified Service Provider，KCSP），是国内最早投入Kubernetes社区的厂商，是容器开源社区主要贡献者和容器生态领导者。华为云也是 CNCF 云原生计算基金会的创始成员及白金会员，云容器引擎是全球首批通过 CNCF 基金会 Kubernetes 一致性认证的容器服务。

关于更多云容器引擎 CCE产品介绍，请查看 [什么是云容器引擎](https://support.huaweicloud.com/productdesc-cce/cce_productdesc_0001.html?utm_source=cce_Growth_map&utm_medium=display&utm_campaign=help_center&utm_content=Growth_map)

## 前提条件

本文假设您已开通了 CCE 服务，并成功创建了一个可以访问的 Kubernetes 集群，如果您还没有准备好，请查看[入门指引](https://support.huaweicloud.com/qs-cce/cce_qs_0001.html)

> Kubernetes 集群节点必须可以访问外网（可以通过加NAT网关解决）

> Kubernetes 集群节点的操作系统建议是 Ubuntu，否则有可能会缺少必要的库（socat）

## 为 EMQX 集群配置持久化存储

EMQX Custom Resource 使用 StoreClass 来保存 EMQX 运行时的状态。在开始之前，您需要准备 StoreClass。目前CCE默认提供csi-disk、csi-nas、csi-obs等StorageClass，执行如下命令即可查询CCE提供的默认StorageClass。您可以使用CCE提供的CSI插件自定义创建StorageClass，但从功能角度与CCE提供的默认StorageClass并无区别，这里不做过多描述。更多详情请参考[存储类StorageClass](https://support.huaweicloud.com/usermanual-cce/cce_10_0380.html)

```
# kubectl get sc
NAME                PROVISIONER                     AGE
csi-disk            everest-csi-provisioner         17d          # 云硬盘 StorageClass
csi-nas             everest-csi-provisioner         17d          # 文件存储 1.0 StorageClass
csi-sfs             everest-csi-provisioner         17d          # 文件存储 3.0 StorageClass
csi-obs             everest-csi-provisioner         17d          # 对象存储 StorageClass
csi-sfsturbo        everest-csi-provisioner         17d          # 极速文件存储 StorageClass
csi-local-topology  everest-csi-provisioner         17d          # 本地持久卷
```

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
        storageClassName:  csi-disk
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
    spec:
      storageClassName: csi-disk
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

## 通过 LoadBalancer 访问 EMQX 集群

负载均衡( LoadBalancer )可以通过弹性负载均衡从公网访问到工作负载，与弹性IP方式相比提供了高可靠的保障，一般用于系统中需要暴露到公网的服务。负载均衡访问方式由公网弹性负载均衡服务地址以及设置的访问端口组成，例如“10.117.117.117:80”。关于更多负载均衡的内容，请查看[负载均衡(LoadBalancer)](https://support.huaweicloud.com/usermanual-cce/cce_10_0014.html)

在公有云中，一般通过配置资源的 Annotation 来配置负载均衡器的相关属性，

修改 EMQX Custom Resource 的配置，添加相应的 Annotation，并将 Service Type 设置为 LoadBalancer，如下所示:

:::: tabs type:card 
::: tab v2alpha1

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
  name: emqx
spec:
  image: emqx:5.0
  listenersServiceTemplate:
    metadata:
      annotations:
        kubernetes.io/elb.pass-through: "true"
        kubernetes.io/elb.class: union
        kubernetes.io/elb.autocreate: |
            '{
              "type": "public",
              "name": "emqx",
              "bandwidth_name": "cce-emqx",
              "bandwidth_chargemode": "bandwidth",
              "bandwidth_size": 5,
              "bandwidth_sharetype": "PER",
              "eip_type": "5_bgp"
            }'
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
spec:
  template:
    spec:
      emqxContainer:
        image: 
          repository: emqx/emqx-ee
          version: 4.4.14
  serviceTemplate:
    metadata:
      annotations:
        kubernetes.io/elb.pass-through: "true"
        kubernetes.io/elb.class: union
        kubernetes.io/elb.autocreate: |
            '{
              "type": "public",
              "name": "emqx-ee",
              "bandwidth_name": "cce-emqx",
              "bandwidth_chargemode": "bandwidth",
              "bandwidth_size": 5,
              "bandwidth_sharetype": "PER",
              "eip_type": "5_bgp"
            }'
    spec:
      type: LoadBalancer
```

::: 
::::

