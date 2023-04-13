# 在 Amazon EKS 中部署 EMQX

Amazon EKS（Elastic Kubernetes Service）是一种托管的 Kubernetes 服务，可让您轻松部署、管理和扩展容器化应用程序。EKS 提供了 Kubernetes 控制平面和节点组，自动处理节点替换、升级和修补。它支持 AWS 服务，如 Load Balancers、RDS 和 IAM，并与其他 Kubernetes 生态系统工具无缝集成。

## 前提条件

在开始之前，您需要准备以下内容：

- Amazon EKS 集群，详情请参见：[开始使用 Amazon EKS](https://docs.aws.amazon.com/zh_cn/eks/latest/userguide/getting-started.html)

- 负载均衡器，详情请参见：[负载均衡器介绍](https://docs.aws.amazon.com/zh_cn/eks/latest/userguide/network-load-balancing.html)

- 存储类，详情请参见：[存储类](https://docs.aws.amazon.com/zh_cn/eks/latest/userguide/storage-classes.html)

## 配置持久化存储

EMQX 自定义资源使用 StoreClass 来保存 EMQX 运行时的状态。在开始之前，需要准备好 StoreClass。以下是使用 `ebs-sc` 配置 EMQX 自定义资源的示例。

:::tip
根据您想要部署的 EMQX 版本，可以选择相应的 APIVersion。有关具体的兼容性关系，请参阅 [EMQX 兼容性](../README.md)
:::

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
        storageClassName: ebs-sc
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
    spec:
      storageClassName: ebs-sc
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

## 访问 EMQX 集群

在公共云提供商中，您可以使用 LoadBalancer 来访问 EMQX 集群。详情请参考：[Service Annotations](https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/guide/service/annotations/)

修改 EMQX 自定义资源的配置，添加相应的注释，并将 Service 类型设置为负载均衡器，如下所示：

:::: tabs type:card
::: tab v2alpha1

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
  name: emqx
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: "external"
    service.beta.kubernetes.io/aws-load-balancer-nlb-target-type: "ip"
    service.beta.kubernetes.io/aws-load-balancer-scheme: internet-facing
    service.beta.kubernetes.io/aws-load-balancer-attributes: load_balancing.cross_zone.enabled=true
    service.beta.kubernetes.io/aws-load-balancer-target-group-attributes: preserve_client_ip.enabled=true
    service.beta.kubernetes.io/aws-load-balancer-attributes: deletion_protection.enabled=true
#   service.beta.kubernetes.io/aws-load-balancer-subnets: subnet-xxx1,subnet-xxx2
spec:
  image: emqx:5.0
  imagePullPolicy: IfNotPresent
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
    service.beta.kubernetes.io/aws-load-balancer-type: "external"
    service.beta.kubernetes.io/aws-load-balancer-nlb-target-type: "ip"
    service.beta.kubernetes.io/aws-load-balancer-scheme: internet-facing
    service.beta.kubernetes.io/aws-load-balancer-attributes: load_balancing.cross_zone.enabled=true
    service.beta.kubernetes.io/aws-load-balancer-target-group-attributes: preserve_client_ip.enabled=true
    service.beta.kubernetes.io/aws-load-balancer-attributes: deletion_protection.enabled=true
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

## TLS 终结

在 Amazon EKS 中，您可以使用 NLB 进行 TLS 终结，可以按照以下步骤进行操作：

1. 在 [AWS Certificate Manager](https://aws.amazon.com/cn/certificate-manager/?nc1=h_ls) 控制台中导入相关证书，然后通过单击证书 ID 进入详细信息页面，然后复制 ARN，如下图所示：

:::tip
证书和密钥的导入格式，请参考[官方文档](https://docs.aws.amazon.com/zh_cn/acm/latest/userguide/import-certificate-format.html)
:::

![](./assets/cert.png)

2. 在 EMQX 自定义资源的元数据中添加如下注释：

    ```yaml
    service.beta.kubernetes.io/aws-load-balancer-ssl-cert: arn:aws:acm:us-east-2:609217282285:certificate/2519bd2b-f523-43de-9593-ec92132791e7
    service.beta.kubernetes.io/aws-load-balancer-backend-protocol: tcp
    service.beta.kubernetes.io/aws-load-balancer-ssl-ports: 1883,mqtt-tls
    ```

    > `service.beta.kubernetes.io/aws-load-balancer-ssl-cert` 的值是我们在第一步复制的 ARN 信息
