# 使用 EMQX Operator 在阿里云 ACK 上部署 EMQX 集群


## 名词解释

EMQX：The most scalable open-source MQTT broker for IoT，[EMQX 文档](https://github.com/emqx/emqx)

EMQX Operator：A Kubernetes Operator for EMQX，[EMQX Operator 文档](https://github.com/emqx/emqx-operator)

ACK：Alibaba Cloud Container Service for Kubernetes，简称容器服务 ACK， [ACK 文档](https://www.alibabacloud.com/help/zh/container-service-for-kubernetes/latest/ack-cluster-overview)

CLB：传统型负载均衡 CLB（Classic Load Balancer）是将访问流量根据转发策略分发到后端多台云服务器的流量分发控制服务，[CLB 文档](https://help.aliyun.com/document_detail/27539.html)


## 创建 ACK 集群

登录阿里云，选择云产品  -> 容器服务 Kubernets 版，点击创建， 选择标准集群，EMQX Operator 要求Kubernetes 版本>=1.20.0，因此我们在此选择 Kubernetes 选择 1.22.10，网络与其他资源信息根据自身需求来制定。具体创建步骤参考： [创建标准集群](https://www.alibabacloud.com/help/zh/container-service-for-kubernetes/latest/create-an-ack-managed-cluster)


## LoadBalancer 配置

支持在 Terway 网络模式下，通过 annotation 将 Pod 直接挂载到 CLB 后端，提升网络转发性能。：[通过Annotation配置负载均衡](https://www.alibabacloud.com/help/zh/container-service-for-kubernetes/latest/use-annotations-to-configure-load-balancing-1)

```shell
service.beta.kubernetes.io/backend-type："eni"
```


## StorageClass 配置

使用如下命令查看当前集群可用的 storageClass:

```shell
kubectl get sc
```

可以看到集群默认创建了多个可用的 storageClass, 本文档部署 EMQX 时选取的第一个 storageClass: alibabacloud-cnfs-nas, 其他 StorageClass 可参考文档[存储-CSI](https://help.aliyun.com/document_detail/127551.html)


## 使用 EMQX Operator 部署 EQMX 集群

EMQX Operator 安装参考：[EMQX Operator 安装](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/getting-started/getting-started.md)

EMQX Operator 安装完成后，使用以下命令在 ACK 上进行部署 EMQX 集群：

```shell
cat << EOF | kubectl apply -f -
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
  name: emqx-ee
  labels:
    "apps.emqx.io/instance": "emqx-ee"
  annotations:
    service.beta.kubernetes.io/backend-type: "eni"
spec:
  emqxTemplate:
    image: emqx/emqx-ee:4.4.8
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
    storageClassName: alibabacloud-cnfs-nas
EOF
```


## 使用 LB 终结 TCP TLS 方案

由于阿里云 CLB 不支持 TCP 证书终结(NLB发布后，我们会更该该项内容)，所以请参考这篇文档解决 TCP 证书终结问题，[LB 终结 TCP TLS 方案](https://github.com/emqx/emqx-operator/discussions/312)


**备注**： 此文档详细解释了使用 EMQX Operator 在阿里云 ACK 上部署 EMQX 集群的步骤，另外还支持配置 LB 直连 Pod, 进一步提升转发性能。





