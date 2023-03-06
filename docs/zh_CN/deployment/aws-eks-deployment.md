# 本文主要介绍在AWS EKS上通过EMQX Operator 部署 EMQX集群，主要内容包括:

## 名词解释

EMQX: The most scalable open-source MQTT broker for IoT, [详见](https://github.com/emqx/emqx) 

EMQX Operator: A Kubernetes Operator for EMQX, [详见](https://github.com/emqx/emqx-operator) 

EKS:  Amazon Elastic Kubernetes Service , [详见](https://docs.aws.amazon.com/eks/latest/userguide/what-is-eks.html) 

NLB：AWS 提供的LoadBalancer

## 创建 EKS 集群

登录 AWS EKS 控制台，进入创建eks 集群页面，具体创建步骤参考：[云厂商文档](https://docs.aws.amazon.com/eks/latest/userguide/create-cluster.html)

## 访问 EKS 集群

参考: [AWS 手册](https://docs.aws.amazon.com/eks/latest/userguide/create-kubeconfig.html)  

## LoadBalancer 配置 

Load Balancer 介绍: [AWS 手册](https://docs.aws.amazon.com/eks/latest/userguide/network-load-balancing.html) 

Load Balancer Controller安装 : [AWS 手册](https://docs.aws.amazon.com/eks/latest/userguide/aws-load-balancer-controller.html) 

Annotations: [AWS 手册](https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/guide/service/annotations/) 

## StorageClass 配置

[点击查看权限设置](https://docs.aws.amazon.com/eks/latest/userguide/csi-iam-role.html) 
storageclass  yaml 示例，此处使用ebs 
[查看ebs插件安装](https://docs.aws.amazon.com/eks/latest/userguide/managing-ebs-csi.html) 

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: ebs-sc
provisioner: ebs.csi.aws.com
volumeBindingMode: Immediate
parameters:
  csi.storage.k8s.io/fstype: xfs
  type: io1
  iopsPerGB: "500"
  encrypted: "true"
allowedTopologies:
- matchLabelExpressions:
  - key: topology.ebs.csi.aws.com/zone
    values:
    - us-east-2c
```

执行以下命令

```bash
kubectl apply -f storageclass.yaml
```

## 使用 EMQX Operator 进行集群创建 

[查看 Operator 安装](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/getting-started/getting-started.md) 
Operator 安装完成后，使用以下 yaml 在 AWS EKS 上进行部署 EMQX 集群

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
  image: emqx/emqx:5.0.14
  imagePullPolicy: IfNotPresent
  coreTemplate:
    spec:
      volumeClaimTemplates:
        storageClassName: ebs-sc
        resources:
          requests:
            storage: 20Mi
        accessModes:
        - ReadWriteOnce
      replicas: 3
  replicantTemplate:
    spec:
      replicas: 0
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
#   service.beta.kubernetes.io/aws-load-balancer-subnets: subnet-xxx1,subnet-xxx2
spec:
  persistent:
    metadata:
      name: emqx-ee
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
  serviceTemplate:
    spec:
      type: LoadBalancer
```
::: 
::: tab v1beta3

```yaml
apiVersion: apps.emqx.io/v1beta3
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
#   service.beta.kubernetes.io/aws-load-balancer-subnets: subnet-xxx1,subnet-xxx2
spec:
  persistent:
    storageClassName: ebs-sc
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
```
::: 
::::

__默认情况下，AWS 创建 NLB 时会自动将实例所在的 VPC 下的所有区域加入网络映射，这意味客户端的流量会转发至所有区域，即便该区域未部署任何 K8S Node，因此会网络不通的问题。因此我们需要根据实际情况通过注解`service.beta.kubernetes.io/aws-load-balancer-subnets`指定 NLB 的可用区域__

## 使用 NLB 进行 TLS 终结

我们推荐在 NLB 上做 TLS 终结,如需在 NLB 上实现 TLS 终于，你可以通过以下几个步骤实现

### 证书导入

在 AWS [控制台](https://us-east-2.console.aws.amazon.com/acm/home)，导入相关证书, 证书导入后点击证书 ID，进入详情页面，复制ARN信息，如下图:
![](./assets/cert.png)

### 修改部署yaml

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
    service.beta.kubernetes.io/aws-load-balancer-ssl-cert: arn:aws:acm:us-west-2:arn:arn:aws:acm:us-east-1:609217282285:certificate/326649a0-f3b3-4bdb-a478-5691b4ba0ef3
    service.beta.kubernetes.io/aws-load-balancer-backend-protocol: tcp
    service.beta.kubernetes.io/aws-load-balancer-ssl-ports: 1883,mqtt-tls
    service.beta.kubernetes.io/aws-load-balancer-target-group-attributes: preserve_client_ip.enabled=true
    service.beta.kubernetes.io/aws-load-balancer-attributes: deletion_protection.enabled=true
#   service.beta.kubernetes.io/aws-load-balancer-subnets: subnet-xxx1,subnet-xxx2
spec:
  image: emqx/emqx:5.0.14
  imagePullPolicy: IfNotPresent
  coreTemplate:
    spec:
      volumeClaimTemplates:
        storageClassName: ebs-sc
        resources:
          requests:
            storage: 20Mi
        accessModes:
        - ReadWriteOnce
      replicas: 3
  replicantTemplate:
    spec:
      replicas: 0
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
    service.beta.kubernetes.io/aws-load-balancer-ssl-cert: arn:aws:acm:us-west-2:arn:arn:aws:acm:us-east-1:609217282285:certificate/326649a0-f3b3-4bdb-a478-5691b4ba0ef3
    service.beta.kubernetes.io/aws-load-balancer-backend-protocol: tcp
    service.beta.kubernetes.io/aws-load-balancer-ssl-ports: 1883,mqtt-tls
    service.beta.kubernetes.io/aws-load-balancer-target-group-attributes: preserve_client_ip.enabled=true
    service.beta.kubernetes.io/aws-load-balancer-attributes: deletion_protection.enabled=true
#   service.beta.kubernetes.io/aws-load-balancer-subnets: subnet-xxx1,subnet-xxx2
spec:
  persistent:
    metadata:
      name: emqx-ee
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
  serviceTemplate:
    spec:
      type: LoadBalancer
```
::: 
::: tab v1beta3

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
  name: emqx-ee
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: "external"
    service.beta.kubernetes.io/aws-load-balancer-nlb-target-type: "ip"
    service.beta.kubernetes.io/aws-load-balancer-scheme: internet-facing
    service.beta.kubernetes.io/aws-load-balancer-attributes: load_balancing.cross_zone.enabled=true
    service.beta.kubernetes.io/aws-load-balancer-ssl-cert: arn:aws:acm:us-west-2:arn:arn:aws:acm:us-east-1:609217282285:certificate/326649a0-f3b3-4bdb-a478-5691b4ba0ef3
    service.beta.kubernetes.io/aws-load-balancer-backend-protocol: tcp
    service.beta.kubernetes.io/aws-load-balancer-ssl-ports: 1883,mqtt-tls
    service.beta.kubernetes.io/aws-load-balancer-target-group-attributes: preserve_client_ip.enabled=true
    service.beta.kubernetes.io/aws-load-balancer-attributes: deletion_protection.enabled=true
#   service.beta.kubernetes.io/aws-load-balancer-subnets: subnet-xxx1,subnet-xxx2
spec:
  persistent:
    storageClassName: ebs-sc
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
```
::: 
::::

相比不使用 TLS 证书，我们在 annotations 里增加了下面三项内容，其中 `service.beta.kubernetes.io/aws-load-balancer-ssl-cert` 的值为我们第一步中复制的 ARN 信息。

```yaml
service.beta.kubernetes.io/aws-load-balancer-ssl-cert: arn:aws:acm:us-west-2:arn:arn:aws:acm:us-east-1:609217282285:certificate/326649a0-f3b3-4bdb-a478-5691b4ba0ef3
service.beta.kubernetes.io/aws-load-balancer-backend-protocol: tcp
service.beta.kubernetes.io/aws-load-balancer-ssl-ports: 1883,mqtt-tls
```