# 在阿里云上部署 EMQX 集群

阿里云容器服务Kubernetes版（Alibaba Cloud Container Service for Kubernetes，简称容器服务ACK）是面向多种业务场景提供多样化的Kubernetes集群：

+ ACK集群：适合大多数业务场景，是一种最通用的Kubernetes集群。
+ ASK集群：适合快速伸缩的敏捷业务场景以及单个或多个任务处理的场景。更多信息，请参见ASK概述。
+ 边缘集群：是IoT、CDN等边缘业务的必选。更多信息，请参见ACK@Edge概述。

此外，容器服务ACK在基因计算、AI大数据等领域提供了高度集成的解决方案，结合IaaS高性能计算、网络能力，发挥容器的优秀性能。在多云混合云领域，容器服务ACK提供了多集群统一管理能力，您可在容器服务控制台，统一管理来自线下IDC，或者其他云上的Kubernetes集群。

## 前提条件

在开始之前，我们需要开通阿里云 AKC 及相关的服务，具体请参考：[首次使用容器服务Kubernetes版](https://www.alibabacloud.com/help/zh/container-service-for-kubernetes/latest/quick-start-for-first-time-users)，本文假设您已经开通了 AKC 服务，并且已经创建了一个 ACK 集群。

## 部署 EMQX Operator

EMQX Operator 安装参考：[部署 EMQX Operator](../getting-started/getting-started.md)

## 为 EMQX 集群配置持久化存储

EMQX Custom Resource 使用 StoreClass 来保存 EMQX 运行时的状态。在开始之前，您需要准备 StoreClass。使用 `kubectl get storeClass` 可以查看当前 Kubernetes 集群中的 StoreClass，阿里云 ACK 服务默认创建了多个可用的 StorageClass，本文以 `alibabacloud-cnfs-nas` 为例, 其他 StorageClass 可参考文档[存储-CSI](https://help.aliyun.com/document_detail/127551.html)

下面是 EMQX Custom Resource 的相关配置，你可以根据希望部署的 EMQX 的版本来选择对应的 APIVersion，具体的兼容性关系，请参考[EMQX Operator 兼容性](../README.md):

:::: tabs type:card
::: tab v2alpha1

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
  name: emqx
  annotations:
   service.beta.kubernetes.io/backend-type: "eni"
spec:
  image: emqx:5.0
  coreTemplate:
    spec:
      volumeClaimTemplates:
        storageClassName: alibabacloud-cnfs-nas
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
  annotations:
    service.beta.kubernetes.io/backend-type: "eni"
spec:
  persistent:
    metadata:
      name: emqx-ee
    spec:
      storageClassName: alibabacloud-cnfs-nas
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

## 使用 LoadBalancer 访问 EMQX 集群，并终结 TLS 加密

在公有云中，可以通过云厂商提供的 LoadBalancer 服务来访问 EMQX 集群，网络型负载均衡 NLB（ Network Load Balancer ）是阿里云面向万物互联时代推出的新一代四层负载均衡，支持超高性能和自动弹性能力，单实例可以达到1亿并发连接，轻松应对高并发业务。NLB 可以在 LoadBalancer 实现 TLS 终结，将 TCP 流量转发给 EMQX，降低了 EMQX 的负载。

> 使用 NLB 要求 k8s 版本不低于v1.24且 CCM 版本不低于v2.5.0。有关 CCM 的版本升级说明请查看[官方文档](https://help.aliyun.com/document_detail/198792.html)

在使用 NLB 终结 TLS 流量之前，需要先创建一个 TLS 证书，阿里云的[数字证书管理服务](https://us-east-2.console.aws.amazon.com/acm/home)控制台，可以导入自签名证书或者购买证书, 证书导入后点击证书详情，获取证书ID。如下图:
![](./assets/aliyun-cert.png)

> 由于每次重新创建 NLB 时，其关联的 DNS 域名会发生变化，如果采用自签名证书，为方便测试，这里建议将证书绑定的域名设置为`*.cn-shanghai.nlb.aliyuncs.com`

在公有云中，一般通过资源的 Annotation 来配置负载均衡器的相关属性，详情请参考[通过Annotation配置负载均衡](https://www.alibabacloud.com/help/zh/container-service-for-kubernetes/latest/use-annotations-to-configure-load-balancing-1)

修改 EMQX Custom Resource 的配置，添加相应的 Annotation，并将 Service Type 设置为 LoadBalancer，如下所示:

:::: tabs type:card
::: tab v2alpha1

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
  name: emqx
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-name: "nlb"
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-force-override-listeners: "true"
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-protocol-port: "tcpssl:8883"
    # 如集群为中国内地 Region 时，组合后的证书ID为${your-cert-id}-cn-hangzhou。
    # 如集群为除中国内地以外的其他Region时，组合后的证书ID为${your-cert-id}-ap-southeast-1，例如：6134-ap-southeast-1。
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-cert-id: "${组合后的证书ID}"
    # NLB 支持的地域及可用区可以登录 NLB 控制台查看，至少需要两个可用区。
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321
spec:
    image: emqx:5.0
    listenersServiceTemplate:
     spec:
        type: LoadBalancer
        ports:
        - name: tcpssl
          port: 8883
          protocol: TCP
          targetPort: 1883
```
:::
::: tab v1beta4

```yaml
apiVersion: apps.emqx.io/v1beta4
kind: EmqxEnterprise
metadata:
  name: emqx-ee
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-name: "nlb"
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-force-override-listeners: "true"
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-protocol-port: "tcpssl:8883"
    # 如集群为中国内地 Region 时，组合后的证书ID为${your-cert-id}-cn-hangzhou。
    # 如集群为除中国内地以外的其他Region时，组合后的证书ID为${your-cert-id}-ap-southeast-1，例如：6134-ap-southeast-1。
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-cert-id: "${组合后的证书ID}"
    # NLB 支持的地域及可用区可以登录 NLB 控制台查看，至少需要两个可用区。
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321
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
      ports:
      - name: tcpssl
        port: 8883
        protocol: TCP
        targetPort: 1883
```
:::
::::

可用区域的组成规则为服务器所在区域+专有网络中[交换机](https://vpc.console.aliyun.com/vpc/cn-shanghai/switches)的实例ID
![](./assets/aliyun-vsw.png)

> 查看[官方文档](https://help.aliyun.com/document_detail/456461.html)以了解更多的参数说明

部署成功后，可在[网络型负载均衡 NLB](https://slb.console.aliyun.com/nlb)中查看自动创建的 NLB 实例
