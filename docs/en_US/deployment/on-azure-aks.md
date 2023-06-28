# Deploy EMQX on Azure Kubernetes Service

EMQX Operator supports deploying EMQX on Azure Kubernetes Service(AKS). AKS simplifies deploying a managed Kubernetes cluster in Azure by offloading the operational overhead to Azure. As a hosted Kubernetes service, Azure handles critical tasks, like health monitoring and maintenance. When you create an AKS cluster, a control plane is automatically created and configured. This control plane is provided at no cost as a managed Azure resource abstracted from the user. You only pay for and manage the nodes attached to the AKS cluster.

## Before You Begin
Before you begin, you must have the following:

- To create an AKS cluster on Azure, you first need to activate the AKS service in your Azure subscription. Refer to the [Azure Kubernetes Service](https://learn.microsoft.com/en-us/azure/aks/) documentation for more information.

- To connect to an AKS cluster using kubectl commands, you can install the kubectl tool locally and obtain the cluster's KubeConfig to connect to the cluster. Alternatively, you can use Cloud Shell through the Azure portal to manage the cluster with kubectl.
  - To connect to an AKS cluster using kubectl, you need to install and configure the kubectl tool on your local machine. Refer to the [Connect to an AKS cluster](https://learn.microsoft.com/en-us/azure/aks/learn/quick-kubernetes-deploy-cli) documentation for detailed instructions on how to do this.
  - To connect to an AKS cluster using CloudShell, use Azure CloudShell to connect to the AKS cluster and manage the cluster using kubectl. Refer to the [Manage an AKS cluster in Azure CloudShell](https://learn.microsoft.com/en-us/azure/aks/learn/quick-kubernetes-deploy-portal?tabs=azure-cli) documentation for detailed instructions on how to connect to Azure CloudShell and use kubectl.


- To install EMQX Operator, please refer to [Install EMQX Operator](../getting-started/getting-started.md)


## Quickly deploying an EMQX cluster

Here are the relevant configurations for EMQX Custom Resource. You can choose the corresponding APIVersion based on the version of EMQX you wish to deploy. For specific compatibility relationships, please refer to [EMQX Operator Compatibility](../index.md):

:::: tabs type:card
::: tab apps.emqx.io/v1beta4

Save the following content as a YAML file and deploy it using the `kubectl apply` command.

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
      ## more information about storage classes: https://learn.microsoft.com/en-us/azure/aks/concepts-storage#storage-classes
      storageClassName: default
      resources:
        requests:
          storage: 10Gi
      accessModes:
        - ReadWriteOnce
  template:
    spec:
      emqxContainer:
        image:
          repository: emqx/emqx-ee
          version: 4.4.15
  serviceTemplate:
    spec:
      ## more information about load balancer: https://learn.microsoft.com/en-us/azure/aks/load-balancer-standard
      type: LoadBalancer

```

Wait for the EMQX cluster to be ready. You can check the status of the EMQX cluster using the `kubectl get` command. Please ensure that the STATUS is `Running` which may take some time.

```shell
$ kubectl get emqxenterprises
NAME      STATUS   AGE
emqx-ee   Running  8m33s
```

Get the External IP of the EMQX cluster and access the EMQX console.

```shell
$ kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip'

20.245.123.100
```

Access the EMQX console by opening a web browser and visiting http://20.245.123.100:18083. Login using the default username and password `admin/public`.


:::
::: tab apps.emqx.io/v2alpha1

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
  name: emqx
spec:
  image: "emqx:5.1"
  coreTemplate:
    spec:
      volumeClaimTemplates:
        ## more information about storage classes: https://learn.microsoft.com/en-us/azure/aks/concepts-storage#storage-classes
        storageClassName: default
        resources:
          requests:
            storage: 10Gi
        accessModes:
        - ReadWriteOnce
  dashboardServiceTemplate:
    spec:
      ## more information about load balancer: https://learn.microsoft.com/en-us/azure/aks/load-balancer-standard
      type: LoadBalancer
  listenersServiceTemplate:
    spec:
      ## more information about load balancer: https://learn.microsoft.com/en-us/azure/aks/load-balancer-standard
      type: LoadBalancer
```

Wait for the EMQX cluster to be ready. You can check the status of the EMQX cluster using the `kubectl get` command. Please ensure that the STATUS is `Running` which may take some time.

```shell
$ kubectl get emqx
NAME   IMAGE      STATUS    AGE
emqx   emqx:5.1   Running   118s
```

Get the External IP of the EMQX cluster and access the EMQX console.

The EMQX Operator will create two EMQX Service resources, one is `emqx-dashboard`, and the other is `emqx-listeners`, corresponding to the EMQX console and EMQX listening port, respectively.

```shell
$ kubectl get svc emqx-dashboard -o json | jq '.status.loadBalancer.ingress[0].ip'

20.245.230.91
```

Access the EMQX console by opening a web browser and visiting http://20.245.230.91:18083. Login using the default username and password `admin/public`.

:::
::::

## Connecting to EMQX cluster to publish/subscribe messages using MQTT X CLI

[MQTT X CLI](https://mqttx.app/cli) is an open-source MQTT 5.0 command-line client tool designed to help developers develop and debug MQTT services and applications faster without the need for a GUI.

- Retrieve External IP of the EMQX cluster

:::: tabs type:card
::: tab apps.emqx.io/v1beta4

```shell
external_ip=$(kubectl get svc emqx-listeners -o json | jq '.status.loadBalancer.ingress[0].ip')
```

:::
::: tab apps.emqx.io/v2alpha1

```shell
external_ip=$(kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip')
```

:::
::::

- Subscribe to messages

```shell
$ mqttx sub -t 'hello' -h ${external_ip} -p 1883

[10:00:25] › …  Connecting...
[10:00:25] › ✔  Connected
[10:00:25] › …  Subscribing to hello...
[10:00:25] › ✔  Subscribed to hello
```

- Create a new terminal window and send a message

```shell
$ mqttx pub -t 'hello' -h ${external_ip} -p 1883 -m 'hello world'

[10:00:58] › …  Connecting...
[10:00:58] › ✔  Connected
[10:00:58] › …  Message Publishing...
[10:00:58] › ✔  Message published
```

- View messages received in the subscription terminal window

```shell
[10:00:58] › payload: hello world
```

## About LoadBalancer Offloading TLS

Since Azure LoadBalancer does not support TCP certificates, please refer to this [document](https://github.com/emqx/emqx-operator/discussions/312) to resolve TCP certificate offloading issues.
