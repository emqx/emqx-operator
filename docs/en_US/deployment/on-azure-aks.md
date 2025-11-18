# Deploy EMQX on Azure Kubernetes Service

EMQX Operator supports deploying EMQX on Azure Kubernetes Service (AKS). AKS simplifies deploying a managed Kubernetes cluster in Azure by offloading the operational overhead to Azure. As a hosted Kubernetes service, Azure handles critical tasks, like health monitoring and maintenance. When you create an AKS cluster, a control plane is automatically created and configured. This control plane is provided at no cost as a managed Azure resource abstracted from the user.

## Before You Begin

Before you begin, you need to have the following:

- An AKS cluster on Azure.
  * You need to activate the AKS service in your Azure subscription. Refer to the [Azure Kubernetes Service](https://learn.microsoft.com/en-us/azure/aks/) documentation for more information.

- Working `kubectl` configuration to connect to the AKS cluster.
  - To connect to the AKS cluster using `kubectl`, you need to install and configure the `kubectl` tool on your local machine. Refer to the [Connect to an AKS cluster](https://learn.microsoft.com/en-us/azure/aks/learn/quick-kubernetes-deploy-cli) documentation for detailed instructions.
  - To connect to the AKS cluster using CloudShell, refer to the [Manage an AKS cluster in Azure CloudShell](https://learn.microsoft.com/en-us/azure/aks/learn/quick-kubernetes-deploy-portal?tabs=azure-cli) documentation.


- Install EMQX Operator.
  - Please refer to [Install EMQX Operator](../getting-started/getting-started.md) for further details.


## Deploy EMQX Cluster Quickly

Here is the basic configuration for an EMQX Custom Resource (CR).

+ Save it as a YAML file and deploy with `kubectl apply`.

  ```yaml
  apiVersion: apps.emqx.io/v2
  kind: EMQX
  metadata:
    name: emqx
  spec:
    image: "emqx/emqx:6.0"
    config:
      data: |
        license {
          key = "..."
        }
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

+ Wait for the EMQX cluster to become ready.

  Check the status of the EMQX cluster using `kubectl get`, make sure that the `STATUS` is `Ready`. This may take some time.

  ```shell
  $ kubectl get emqx
  NAME   STATUS    AGE
  emqx   Ready     1m5s
  ```

+ Look up the external IP of the EMQX Dashboard and access it.

  EMQX Operator will create a Service resource for the EMQX Dashboard according to the `dashboardServiceTemplate` configuration.

  ```shell
  $ kubectl get svc emqx-dashboard -o json | jq -r '.status.loadBalancer.ingress[0].ip'
  20.245.230.91
  ```

  Access `http://20.245.230.91:18083` through the browser.

  Use the default username `admin` and password `public` to log into the EMQX Dashboard.

## Use MQTTX to Subscribe and Publish

[MQTTX CLI](https://mqttx.app/cli) is an open source MQTT 5.0 command line client tool, designed to help developers to start using MQTT services and applications quickly.

+ Obtain the external IP of the EMQX TCP listener.

  EMQX Operator will create a respective Service resource for the configured listeners.

  ```shell
  external_ip=$(kubectl get svc emqx-listeners -o json | jq -r '.status.loadBalancer.ingress[0].ip')
  ```

+ Subscribe to messages.

```shell
$ mqttx sub -t 'hello' -h ${external_ip} -p 1883
[10:00:25] › …  Connecting...
[10:00:25] › ✔  Connected
[10:00:25] › …  Subscribing to hello...
[10:00:25] › ✔  Subscribed to hello
```

+ In a separate shell, connect to the EMQX cluster and publish a message.

```shell
$ mqttx pub -t 'hello' -h ${external_ip} -p 1883 -m 'hello world'
[10:00:58] › …  Connecting...
[10:00:58] › ✔  Connected
[10:00:58] › …  Message Publishing...
[10:00:58] › ✔  Message published
```

+ Observe the subscriber client receiving the message.

```shell
[10:00:58] › payload: hello world
```

## About LoadBalancer Offloading TLS

Since Azure LoadBalancer does not support TCP certificates, please refer to this [document](https://github.com/emqx/emqx-operator/discussions/312) to resolve TCP certificate offloading issues.
