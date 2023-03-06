# 使用 EMQX Operator 在 Azure AKS 上部署 EMQX 集群

## 名词解释

EMQX: The most scalable open-source MQTT broker for IoT. 详见：[EMQX文档](https://github.com/emqx/emqx)

EMQX Operator: A Kubernetes Operator for EMQX. 详见：[EMQX Operator文档](https://github.com/emqx/emqx-operator)

AKS: Azure Kubernetes 服务 (AKS)。详见：[Azure 文档](https://docs.microsoft.com/en-us/azure/aks/learn/quick-kubernetes-deploy-portal?tabs=azure-cli)

## 创建AKS集群

登录 Azure AKS 控制台，进入 Kubernetes 服务，创建 Kubernetes 集群。具体创建步骤参考：[云厂商文档](https://docs.microsoft.com/en-us/azure/aks/learn/quick-kubernetes-deploy-portal?tabs=azure-cli)

## 访问 Kubernetes 集群

建议通过 Azure 提供的 Cloud Shell 连接。具体创建步骤参考：[云厂商文档](https://docs.microsoft.com/en-us/azure/cloud-shell/overview)

## StorageClass 配置

这里采用 NSF 文件存储。其他 StorageClass [可参考](https://docs.microsoft.com/en-us/azure/aks/azure-files-csi)

创建 StroageClass

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: azurefile-csi-nfs
provisioner: file.csi.azure.com
allowVolumeExpansion: true
parameters:
  protocol: nfs
mountOptions:
  - nconnect=8
```

查看该 StroageClass 是否创建成功

```shell
kubectl get sc
```
可以看到 `azurefile-csi-nfs` 已经成功创建

## 使用EMQX Operator 部署EMQX集群

Operator 安装[参考](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/getting-started/getting-started.md)

Operator 安装完成后，使用以下yaml 在 azure 上进行部署 EMQX 集群

:::: tabs type:card
::: tab v2alpha1

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
  name: emqx
spec:
  image: "emqx/emqx-enterprise:5.0.0"
  imagePullPolicy: IfNotPresent
  coreTemplate:
    spec:
      volumeClaimTemplates:
        storageClassName: azurefile-csi-nfs
        resources:
          requests:
            storage: 20Mi
        accessModes:
        - ReadWriteOnce
      replicas: 1
  replicantTemplate:
    spec:
      replicas: 3
  dashboardServiceTemplate:
    metadata:
      name: emqx-dashboard
    spec:
      type: LoadBalancer
      selector:
        apps.emqx.io/db-role: core
      ports:
        - name: "dashboard-listeners-http-bind"
          protocol: TCP
          port: 18083
          targetPort: 18083
  listenersServiceTemplate:
    metadata:
      name: emqx-listeners
    spec:
      type: LoadBalancer
      ports:
        - name: "tcp-default"
          protocol: TCP
          port: 1883
          targetPort: 1883
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
      storageClassName: azurefile-csi-nfs
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
    metadata:
      name: emqx-ee
      namespace: default
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
spec:
  persistent:
    storageClassName: azurefile-csi-nfs
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

这里 service type采用LoadBalancer

## 关于 LoadBalancer 终结 TLS

由于 Azure LoadBalancer 不支持 TCP 证书，所以请参考这篇[文档](https://github.com/emqx/emqx-operator/discussions/312)解决 TCP 证书终结问题
