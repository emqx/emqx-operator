# Deploy EMQX on Google Kubernetes Engine

The EMQX Operator allows for the deployment of EMQX on Google Kubernetes Engine (GKE), which simplifies the process of deploying a managed Kubernetes cluster in GCP. With GKE, you can offload the operational overhead to GCP, enabling you to focus on your application deployment and management. By deploying EMQX on GKE, you can take advantage of the scalability and flexibility of Kubernetes, while benefiting from the simplicity and convenience of a managed service. With EMQX Operator and GKE, you can easily deploy and manage your MQTT broker on the cloud, allowing you to focus on your business goals.

## Before You Begin

Before you begin, you need to have the following:

- A GKE cluster on Google Cloud Platform.
  - You need to enable the GKE service in your GCP subscription. Refer to the [Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine/) documentation for more information.

- Working `kubectl` configuration to connect to the GKE cluster.
  - To connect to the GKE cluster using `kubectl`, you need to install and configure the `kubectl` tool on your local machine. Refer to the [Connect to a GKE cluster](https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-access-for-kubectl) documentation for detailed instructions.
  - Alternatively, you can use the Cloud Shell directly from the GCP Console to connect to the GKE cluster and manage the cluster using `kubectl`. Refer to the [Manage a GKE cluster with Cloud Shell](https://cloud.google.com/code/docs/shell/create-configure-gke-cluster) documentation for further details.

- Install EMQX Operator.
  - Please refer to [Install EMQX Operator](../getting-started/getting-started.md) for further details.

## Deploy EMQX Cluster Quickly

Here is the basic configuration for an EMQX Custom Resource (CR).

+ Save the following document as a YAML file and deploy it with `kubectl apply`.

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

  ::: warning
  If you want to limit the CPU and memory resources, ensure that the CPU is at least 250m and the memory is at least 512M.

  - [Resource requests in Autopilot](https://cloud.google.com/kubernetes-engine/docs/concepts/autopilot-resource-requests)
  :::

+ Wait for the EMQX cluster to become ready.

  Check the status of the EMQX cluster using `kubectl get`, make sure that the `STATUS` is `Ready`. This may take some time.

  ```shell
  $ kubectl get emqx
  NAME   STATUS    AGE
  emqx   Ready     1m2s
  ```

+ Look up the external IP of the EMQX Dashboard to access it.

  EMQX Operator will create a Service resource for the EMQX Dashboard according to the `dashboardServiceTemplate` configuration.

  ```shell
  $ kubectl get svc emqx-dashboard -o json | jq -r '.status.loadBalancer.ingress[0].ip'
  34.122.174.166
  ```

+ Access `http://34.122.174.166:18083` through the browser.

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

## Use LoadBalancer for TLS offloading

Since Google LoadBalancer doesn't support TCP certificates, please check [discussion](https://github.com/emqx/emqx-operator/discussions/312) to address TCP certificate offloading issues.
