# Deploy EMQX clusters on Azure AKS using EMQX Operator

## Terminology explanation

EMQX: The most scalable open-source MQTT broker for IoT. For details：[EMQX docs](https://github.com/emqx/emqx)

EMQX Operator: A Kubernetes Operator for EMQX. For details：[EMQX Operator docs](https://github.com/emqx/emqx-operator)

AKS: Azure Kubernetes Service (AKS) simplifies deploying a managed Kubernetes cluster in Azure by offloading the operational overhead to Azure. As a hosted Kubernetes service, Azure handles critical tasks, like health monitoring and maintenance. Since Kubernetes masters are managed by Azure, you only manage and maintain the agent nodes. For details：[Azure docs](https://docs.microsoft.com/en-us/azure/aks/learn/quick-kubernetes-deploy-portal?tabs=azure-cli)

## Create AKS Cluster

Log in to the Azure AKS dashboard, go to the Kubernetes service, and create a Kubernetes cluster. [For details](https://docs.microsoft.com/en-us/azure/aks/learn/quick-kubernetes-deploy-portal?tabs=azure-cli)

## Access Kubernetes cluster

It is recommended to connect through the Cloud Shell provided by Azure. [For details](https://docs.microsoft.com/en-us/azure/cloud-shell/overview)

## StorageClass configurations

NSF file storage is used here. Other StorageClass, please refer to [storage class](https://docs.microsoft.com/en-us/azure/aks/azure-files-csi)

Create StorageClass

```yaml
cat << "EOF" | kubectl apply -f -
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
EOF
```

Check if the StroageClass was created successfully

```shell
kubectl get sc
```
You can see that `azurefile-csi-nfs` has been successfully created


## Deploy EMQX cluster with EMQX Operator

Operator installation refer to [Operator docs](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/getting-started/getting-started.md)

After the Operator installation is complete, deploy the EMQX cluster on Azure using the following yaml


:::: tabs type:card
::: tab v2alpha1

```shell
cat << "EOF" | kubectl apply -f -
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
EOF
```

:::
::: tab v1beta4

```shell
cat << "EOF" | kubectl apply -f -
apiVersion: apps.emqx.io/v1beta4
kind: EmqxEnterprise
metadata:
  name: emqx-ee
  labels:
    "apps.emqx.io/instance": "emqx-ee"
spec:
  persistent:
    metadata:
      name: emqx-ee
      labels:
        "apps.emqx.io/instance": "emqx-ee"
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
      labels:
        "apps.emqx.io/instance": "emqx-ee"
    spec:
      type: LoadBalancer
      selector:
        "apps.emqx.io/instance": "emqx-ee"
EOF
```

:::
::::

Here the service type is LoadBalancer


## About LoadBalancer offloading TLS

Since Azure LoadBalancer does not support TCP certificates, please refer to this [document](https://github.com/emqx/emqx-operator/discussions/312) to resolve TCP certificate offloading issues.
