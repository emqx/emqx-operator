# Deploy EMQX On Tencent Kubernetes Engine

EMQX Operator supports deploying EMQX on Tencent Kubernetes Engine (TKE). Tencent Cloud Container Service provides container-centric, highly scalable, high-performance container management services based on native kubernetes. Tencent Cloud Container Service is fully compatible with the native kubernetes API, providing containerized applications with a series of complete functions such as efficient deployment, resource scheduling, service discovery, and dynamic scaling, which solves the environmental consistency problems of user development, testing, and operation and maintenance processes, and improves The convenience of large-scale container cluster management helps users reduce costs and improve efficiency. Container Service charges corresponding cluster management fees for managed clusters of different specifications. Other cloud product resources (CVM, CBS, CLB, etc.) created during use will be charged according to the billing methods of their respective cloud products

## Prerequisites

Before starting, you need to prepare the following:

- Activate Tencent Cloud Container Service and create a TKE cluster. For details, please refer to: [Create a TKE cluster](https://www.tencentcloud.com/document/product/457/30637)

- Connect to the TKE cluster through the kubectl command. You can install the kubectl tool locally and obtain the KubeConfig of the cluster to connect to the cluster, or use CloudShell on the container service TKE console to manage the cluster through kubectl.

  - Connect to the TKE cluster by installing the kubectl tool locally: For details, please refer to: [Using kubectl to connect to the cluster](https://www.tencentcloud.com/document/product/457/30639#a334f679-7491-4e40-9981-00ae111a9094)
  - Connect to the TKE cluster through CloudShell: For details, please refer to: [Using CloudShell to connect to the cluster](https://www.tencentcloud.com/document/product/457/30639#f97c271a-1204-44d5-967c-2856c83cc5e3)

- Install EMQX Operator: For details, please refer to: [Install EMQX Operator](../getting-started/getting-started.md)

## Quickly deploy an EMQX cluster

The following is the relevant configuration of EMQX custom resources. You can select the corresponding APIVersion according to the EMQX version you want to deploy. For the specific compatibility relationship, please refer to [Compatibility list between EMQX and EMQX Operator](../README.md):

:::: tabs type:card
::: tab apps.emqx.io/v1beta4

+ Save the following content as a YAML file and deploy it via the `kubectl apply` command

  ```yaml
  apiVersion: apps.emqx.io/v1beta4
  kind: EmqxEnterprise
  metadata:
    name: emqx-ee
  spec:
    ## EMQX custom resources do not support updating this field at runtime
    persistent:
      metadata:
        name: emqx-ee
      spec:
        ## More content: https://www.tencentcloud.com/document/product/457/36158
        storageClassName: cbs
        resources:
          requests:
            ## Tencent Cloud TKE requires that the size of the cloud hard disk must be a multiple of 10. The default cbs (high-performance cloud disk) requires a minimum hard disk size of 10GB. For more information, please refer to: https://www.tencentcloud.com/document/product/457/36159
            storage: 10Gi
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
        annotations:
          # Automatically create tke-service-config, for more information, please refer to: https://www.tencentcloud.com/document/product/457/36834
          service.cloud.tencent.com/tke-service-config-auto: "true"
      spec:
        type: LoadBalancer
    ```

+ Wait for the EMQX cluster to be ready, you can check the status of the EMQX cluster through `kubectl get` command, please make sure `STATUS` is `Running`, this may take some time

  ```bash
  $ kubectl get emqxenterprises
  NAME      STATUS   AGE
  emqx-ee   Running  8m33s
  ```

+ Obtain the External IP of the EMQX cluster and access the EMQX console

  ```bash
  $ kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip'

  198.18.3.10
  ```

  Access `http://198.18.3.10:18083` through a browser, and use the default username and password `admin/public` to log in to the EMQX console.

:::
::: tab apps.emqx.io/v2alpha1

+ Save the following content as a YAML file and deploy it via the `kubectl apply` command

  ```yaml
  apiVersion: apps.emqx.io/v2alpha1
  kind: EMQX
  metadata:
    name: emqx
  spec:
    image: emqx:5.0
    coreTemplate:
      spec:
        ## EMQX custom resources do not support updating this field at runtime
        volumeClaimTemplates:
          ## More content: https://www.tencentcloud.com/document/product/457/36158
          storageClassName: cbs
          resources:
            requests:
              ## The cloud disk size must be a multiple of 10. The minimum high-performance cloud hard disk is 10GB. For more information, please refer to: https://www.tencentcloud.com/document/product/457/36159
              storage: 10Gi
          accessModes:
            - ReadWriteOnce
    dashboardServiceTemplate:
      metadata:
        annotations:
          # Automatically create tke-service-config, for more information, please refer to: https://www.tencentcloud.com/document/product/457/36834
          service.cloud.tencent.com/tke-service-config-auto: "true"
      spec:
        type: LoadBalancer
    listenersServiceTemplate:
      metadata:
        annotations:
          # Automatically create tke-service-config, for more information, please refer to: https://www.tencentcloud.com/document/product/457/36834
          service.cloud.tencent.com/tke-service-config-auto: "true"
      spec:
        type: LoadBalancer
  ```

+ Wait for the EMQX cluster to be ready, you can check the status of the EMQX cluster through `kubectl get` command, please make sure `STATUS` is `Running`, this may take some time

  ```bash
  $ kubectl get emqx
  NAME   IMAGE      STATUS    AGE
  emqx   emqx:5.0   Running   2m55s
  ```

+ Obtain the External IP of the EMQX cluster and access the EMQX console

  EMQX Operator will create two EMQX Service resources, one is `emqx-dashboard` and the other is `emqx-listeners`, corresponding to EMQX console and EMQX listening port respectively.

  ```bash
  $ kubectl get svc emqx-dashboard -o json | jq '.status.loadBalancer.ingress[0].ip'

  198.18.3.10
  ```

  Access `http://198.18.3.10:18083` through a browser, and use the default username and password `admin/public` to log in to the EMQX console.

  :::
  ::::

## Use MQTT X CLI To Publish/Subscribe Messages

[MQTT X CLI](https://mqttx.app/cli) is an open source MQTT 5.0 command line client tool, designed to help developers to more Quickly develop and debug MQTT services and applications.

+ Obtain the External IP of the EMQX cluster

  :::: tabs type:card
  ::: tab apps.emqx.io/v1beta4

  ```bash
  external_ip=$(kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip')
  ```
  :::
  ::: tab apps.emqx.io/v2alpha1

  ```bash
  external_ip=$(kubectl get svc emqx-listeners -o json | jq '.status.loadBalancer.ingress[0].ip')
  ```
  :::
  ::::

+ Subscribe to news

  ```bash
  $ mqttx sub -t 'hello' -h ${external_ip} -p 1883

  [10:00:25] › … Connecting...
  [10:00:25] › ✔ Connected
  [10:00:25] › … Subscribing to hello...
  [10:00:25] › ✔ Subscribed to hello
  ```

+ Create a new terminal window and publishinformation

  ```bash
  $ mqttx pub -t 'hello' -h ${external_ip} -p 1883 -m 'hello world'

  [10:00:58] › … Connecting...
  [10:00:58] › ✔ Connected
  [10:00:58] › … Message Publishing...
  [10:00:58] › ✔ Message published
  ```

+ View messages received in a subscribed terminal window

  ```bash
  [10:00:58] › payload: hello world
  ```

## About LoadBalancer Terminating TLS

Currently, Tencent Cloud CLB does not support TLS termination. If you need to use LoadBalancer to terminate TLS, please refer to [Termination TLS](https://github.com/emqx/emqx-operator/discussions/312).
