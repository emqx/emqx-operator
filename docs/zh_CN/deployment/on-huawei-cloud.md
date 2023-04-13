# 在华为云上部署 EMQX 集群

华为云容器引擎（Cloud Container Engine，简称 CCE）提供高度可扩展的、高性能的企业级 Kubernetes 集群，支持运行 Docker 容器。借助云容器引擎，您可以在华为云上轻松部署、管理和扩展容器化应用程序。

云容器引擎深度整合高性能的计算（ECS/BMS）、网络（VPC/EIP/ELB）、存储（EVS/OBS/SFS）等服务，并支持 GPU、NPU、ARM 等异构计算架构，支持多可用区（Available Zone，简称 AZ）、多区域（Region）容灾等技术构建高可用 Kubernetes 集群。

华为云是全球首批 Kubernetes 认证服务提供商（Kubernetes Certified Service Provider，KCSP），是国内最早投入 Kubernetes 社区的厂商，是容器开源社区主要贡献者和容器生态领导者。华为云也是 CNCF 云原生计算基金会的创始成员及白金会员，云容器引擎是全球首批通过 CNCF 基金会 Kubernetes 一致性认证的容器服务。

关于更多云容器引擎 CCE 产品介绍，请查看 [什么是云容器引擎](https://support.huaweicloud.com/productdesc-cce/cce_productdesc_0001.html?utm_source=cce_Growth_map&utm_medium=display&utm_campaign=help_center&utm_content=Growth_map)。

## 前提条件

本文假设您已开通了 CCE 服务，并成功创建了一个可以访问的 Kubernetes 集群，如果您还没有准备好，请查看[入门指引](https://support.huaweicloud.com/qs-cce/cce_qs_0001.html)。

> Kubernetes 集群节点必须可以访问外网（可以通过加 NAT 网关解决）

> Kubernetes 集群节点的操作系统建议是 Ubuntu，否则有可能会缺少必要的库（socat）

## 为 EMQX 集群配置持久化存储

EMQX Custom Resource 使用 StoreClass 来保存 EMQX 运行时的状态。在开始之前，您需要准备 StoreClass。目前 CCE 默认提供 csi-disk、csi-nas、csi-obs 等 StorageClass，执行如下命令即可查询 CCE 提供的默认 StorageClass。您可以使用 CCE 提供的 CSI 插件自定义创建 StorageClass，但从功能角度与 CCE 提供的默认 StorageClass 并无区别，这里不做过多描述。更多详情请参考[存储类 StorageClass](https://support.huaweicloud.com/usermanual-cce/cce_10_0380.html)。

```bash
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
        storageClassName: csi-disk
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

负载均衡( LoadBalancer )可以通过弹性负载均衡从公网访问到工作负载，与弹性 IP 方式相比提供了高可靠的保障，一般用于系统中需要暴露到公网的服务。负载均衡访问方式由公网弹性负载均衡服务地址以及设置的访问端口组成，例如“10.117.117.117:80”。关于更多负载均衡的内容，请查看[负载均衡(LoadBalancer)](https://support.huaweicloud.com/usermanual-cce/cce_10_0014.html)。

在公有云中，一般通过配置资源的 Annotation 来配置负载均衡器的相关属性，

修改 EMQX Custom Resource 的配置，添加相应的 Annotation，并将 Service Type 设置为 `LoadBalancer`，如下所示:

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
        kubernetes.io/elb.class: union
        kubernetes.io/elb.autocreate: |
          {
            "type": "public",
            "name": "emqx",
            "bandwidth_name": "cce-emqx",
            "bandwidth_chargemode": "bandwidth",
            "bandwidth_size": 5,
            "bandwidth_sharetype": "PER",
            "eip_type": "5_bgp"
          }
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
        kubernetes.io/elb.class: union
        kubernetes.io/elb.autocreate: |
          {
            "type": "public",
            "name": "emqx-ee",
            "bandwidth_name": "cce-emqx",
            "bandwidth_chargemode": "bandwidth",
            "bandwidth_size": 5,
            "bandwidth_sharetype": "PER",
            "eip_type": "5_bgp"
          }
    spec:
      type: LoadBalancer
```
:::
::::

将上述文件保存为：emqx.yaml，并执行如下命令部署 EMQX 集群：

```bash
$ kubectl apply -f emqx.yaml
emqx.apps.emqx.io/emqx created
```

等待 EMQX 集群就绪：

:::: tabs type:card
::: tab v2alpha1

```bash
$ kubectl get emqx
NAME   IMAGE      STATUS    AGE
emqx   emqx:5.0   Running   2m55s
```
:::
::: tab v1beta4

```bash
$ kubectl get emqxenterprises
NAME      STATUS   AGE
emqx-ee   Running  8m33s
```
:::
::::

> 确保 `STATUS` 为 `Running`，可能需要一些时间等待 EMQX 集群准备就绪

## 使用 MQTT X CLI 连接 EMQX 集群发布/订阅消息

[MQTT X CLI](https://mqttx.app/zh/cli) 是一款开源的 MQTT 5.0 命令行客户端工具，旨在帮助开发者在不需要使用图形化界面的基础上，也能更快的开发和调试 MQTT 服务与应用。

- 创建一个新的终端窗口并使用 MQTT X CLI 订阅消息

```bash
mqttx sub -h ${loadBalancer_ip} -p 1883 -t "test/topic"
```

- 创建一个新的终端窗口并使用 MQTT X CLI 发布消息

```bash
mqttx pub -h ${loadBalancer_ip} -p 1883 -t "test/topic" -m "hello world"
```

> `${loadBalancer}` 为 EMQX Service 对应的 LoadBalancer IP

## 关于 LoadBalancer 终结 TLS

由于华为 ELB 不支持 TCP 证书，所以请参考文档[终结 TLS](https://github.com/emqx/emqx-operator/discussions/312)解决 TCP 证书终结问题。
