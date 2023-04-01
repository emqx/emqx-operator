# Deploy EMQX on Azure Kubernetes Service

Azure Kubernetes Service (AKS) simplifies deploying a managed Kubernetes cluster in Azure by offloading the operational overhead to Azure. As a hosted Kubernetes service, Azure handles critical tasks, like health monitoring and maintenance. When you create an AKS cluster, a control plane is automatically created and configured. This control plane is provided at no cost as a managed Azure resource abstracted from the user. You only pay for and manage the nodes attached to the AKS cluster.

## Before You Begin
Before you begin, you must have the following:

+ An Azure Kubernetes Service (AKS) cluster, for details: [Create an Azure Kubernetes Service (AKS) cluster](https://docs.microsoft.com/en-us/azure/aks/kubernetes-walkthrough-portal)

+ A StorageClass, for details: [Use Azure Files Container Storage Interface (CSI) driver in Azure Kubernetes Service (AKS)](https://learn.microsoft.com/en-us/azure/aks/azure-files-csie)

+ A LoadBalancer, for details: [Network concepts for applications in Azure Kubernetes Service (AKS)](https://docs.microsoft.com/en-us/azure/aks/concepts-network#load-balancer)

## Enable EMQX Cluster Persistence

NSF file storage is used here. Other StorageClass, please refer to [storage class](https://docs.microsoft.com/en-us/azure/aks/azure-files-csi)

Create StorageClass like this file:

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

Check if the StorageClass was created successfully

```shell
kubectl get sc
```
You can see that `azurefile-csi-nfs` has been successfully created

Here are the relevant configurations for EMQX Custom Resource. You can choose the corresponding APIVersion based on the version of EMQX you wish to deploy. For specific compatibility relationships, please refer to [EMQX Operator Compatibility](../README.md):

:::: tabs type:card
::: tab v2alpha1

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
  name: emqx
spec:
  image: "emqx:5.0"
  coreTemplate:
    spec:
      volumeClaimTemplates:
        storageClassName: azurefile-csi-nfs
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
```
:::
::::

## About LoadBalancer Offloading TLS

Since Azure LoadBalancer does not support TCP certificates, please refer to this [document](https://github.com/emqx/emqx-operator/discussions/312) to resolve TCP certificate offloading issues.
