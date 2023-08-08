# Deploy EMQX on Google Kubernetes Engine

The EMQX Operator allows for the deployment of EMQX on Google Kubernetes Engine (GKE), which simplifies the process of deploying a managed Kubernetes cluster in GCP. With GKE, you can offload the operational overhead to GCP, enabling you to focus on your application deployment and management. By deploying EMQX on GKE, you can take advantage of the scalability and flexibility of Kubernetes, while benefiting from the simplicity and convenience of a managed service. With EMQX Operator and GKE, you can easily deploy and manage your MQTT broker on the cloud, allowing you to focus on your business goals and objectives.


## Before You Begin
Before you begin, you must have the following:

- To create a GKE cluster on Google Cloud Platform, you will need to enable the GKE service in your GCP subscription. You can find more information on how to do this in the Google Kubernetes Engine documentation.


- To connect to a GKE cluster using kubectl commands, you can install the kubectl tool on your local machine and obtain the cluster's KubeConfig to connect to the cluster. Alternatively, you can use Cloud Shell through the GCP Console to manage the cluster with kubectl.

  - To connect to a GKE cluster using kubectl, you will need to install and configure the kubectl tool on your local machine. Refer to the [Connect to a GKE cluster](https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-access-for-kubectl) documentation for detailed instructions on how to do this.

  - To connect to a GKE cluster using Cloud Shell, you can use the Cloud Shell directly from the GCP Console to connect to the GKE cluster and manage the cluster using kubectl. Refer to the [Manage a GKE cluster with Cloud Shell](https://cloud.google.com/code/docs/shell/create-configure-gke-cluster) documentation for detailed instructions on how to connect to Cloud Shell and use kubectl.

- To install EMQX Operator, please refer to [Install EMQX Operator](../getting-started/getting-started.md)


## Install cert-manager

```yaml
$ helm repo add jetstack https://charts.jetstack.io
$ helm repo update
$ helm install \
  cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --set installCRDs=true \
  --set global.leaderElection.namespace=cert-manager
```

::: warning
The default configuration of installing cert-manager may cause bootstrapping issues. Therefore, by using the configuration of `global.leaderElection.namespacer`, `cert-manager` is configured to use a different namespace in leader election.
:::

## Quickly deploying an EMQX cluster

Here are the relevant configurations for EMQX Custom Resource. You can choose the corresponding APIVersion based on the version of EMQX you wish to deploy. For specific compatibility relationships, please refer to [EMQX Operator Compatibility](../index.md):

  ::: warning
  If you want to request CPU and memory resources, you need to ensure that the CPU is greater than or equal to 250m and the memory is greater than or equal to 512M.

  - [Resource requests in Autopilot](https://cloud.google.com/kubernetes-engine/docs/concepts/autopilot-resource-requests)
  :::

:::: tabs type:card
::: tab apps.emqx.io/v2beta1
  * [ ]
Save the following content as a YAML file and deploy it using the `kubectl apply` command.


```yaml
apiVersion: apps.emqx.io/v2beta1
kind: EMQX
metadata:
  name: emqx
spec:
  image: "emqx:5.1"
  coreTemplate:
    spec:
      volumeClaimTemplates:
      ## more information about storage classes: https://cloud.google.com/kubernetes-engine/docs/concepts/persistent-volumes#storageclasses
        storageClassName: standard
        resources:
          requests:
            storage: 10Gi
        accessModes:
        - ReadWriteOnce
  dashboardServiceTemplate:
    spec:
      ## more information about load balancer: https://cloud.google.com/kubernetes-engine/docs/how-to/internal-load-balancing
      type: LoadBalancer
  listenersServiceTemplate:
    spec:
      ## more information about load balancer: https://cloud.google.com/kubernetes-engine/docs/how-to/internal-load-balancing
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

34.122.174.166
```

Access the EMQX console by opening a web browser and visiting http://34.122.174.166:18083. Login using the default username and password `admin/public`.

:::
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
      ## more information about storage classes: https://cloud.google.com/kubernetes-engine/docs/concepts/persistent-volumes#storageclasses
      storageClassName: standard
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
      ## more information about load balancer: https://cloud.google.com/kubernetes-engine/docs/how-to/internal-load-balancing
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

34.68.80.122
```

Access the EMQX console by opening a web browser and visiting http://34.68.80.122:18083. Login using the default username and password `admin/public`.

:::
::::

## Connecting to EMQX cluster to publish/subscribe messages using MQTT X CLI

[MQTT X CLI](https://mqttx.app/cli) is an open-source MQTT 5.0 command-line client tool designed to help developers develop and debug MQTT services and applications faster without the need for a GUI.

- Retrieve External IP of the EMQX cluster

:::: tabs type:card
::: tab apps.emqx.io/v1beta4

```shell
external_ip=$(kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip')
```

:::
::: tab apps.emqx.io/v2beta1

```shell
external_ip=$(kubectl get svc emqx-listeners -o json | jq '.status.loadBalancer.ingress[0].ip')
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

## Use LoadBalancer for TLS offloading

Since Google LoadBalancer doesn't support TCP certificates, please check [discussion](https://github.com/emqx/emqx-operator/discussions/312) to address TCP certificate offloading issues.
