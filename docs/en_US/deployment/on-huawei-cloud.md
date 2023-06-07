# Deploy EMQX On Huawei Cloud Container Engine

EMQX Operator supports deploying EMQX on Huawei Cloud Container Engine (CCE). Cloud Container Engine provides highly scalable, high-performance enterprise-grade Kubernetes clusters that support running Docker containers. With Cloud Container Engine, you can easily deploy, manage, and expand containerized applications on HUAWEI CLOUD.

The cloud container engine deeply integrates services such as high-performance computing (ECS/BMS), network (VPC/EIP/ELB), storage (EVS/OBS/SFS), and supports heterogeneous computing architectures such as GPU, NPU, and ARM. Availability zone (Available Zone, referred to as AZ), multi-region (Region) disaster recovery and other technologies to build a high-availability Kubernetes cluster. For more product introduction of Cloud Container Engine CCE, please see [What is Cloud Container Engine](https://support.huaweicloud.com/intl/en-us/productdesc-cce/cce_productdesc_0001.html).

## Prerequisites

Before starting, you need to prepare the following:

- Activate HUAWEI CLOUD Container Service and create a CCE cluster. For details, please refer to: [Create a CCE cluster](https://support.huaweicloud.com/intl/en-us/qs-cce/cce_qs_0008.html)

    ::: tip
    Kubernetes cluster nodes must be able to access the external network (can be solved by adding a NAT gateway), otherwise third-party images other than the container image service (SoftWare Repository) cannot be pulled
    :::

    :::tip
    The operating system of Kubernetes cluster nodes is recommended to choose Ubuntu, otherwise the necessary library (socat) may be missing
    :::

- Connect to the CCE cluster through the kubectl command. You can install the kubectl tool locally and obtain the KubeConfig of the cluster to connect to the cluster, or use CloudShell on the container service CCE console to manage the cluster through kubectl.

  - Connect to the CCE cluster by installing the kubectl tool locally: For details, please refer to: [Using kubectl to connect to the cluster](https://support.huaweicloud.com/intl/en-us/usermanual-cce/cce_10_0107.html#section2)
  - Connect to the CCE cluster through CloudShell: For details, please refer to: [Using CloudShell to connect to the cluster](https://support.huaweicloud.com/usermanual-cce/cce_10_0671.html#section2)

- Install EMQX Operator: For details, please refer to: [Install EMQX Operator](../getting-started/getting-started.md)

## Quickly Deploy An EMQX Cluster

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
        ## More content: https://support.huaweicloud.com/intl/en-us/usermanual-cce/cce_10_0380.html#section1
        storageClassName: csi-disk
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
            version: 4.4.14
    serviceTemplate:
      metadata:
        annotations:
          ## Automatically create the associated ELB. For detailed field descriptions, please refer to: https://support.huaweicloud.com/intl/en-us/usermanual-cce/cce_10_0014.html#cce_10_0014__table939522754617
          kubernetes.io/elb.autocreate: |
            {
              "type": "public",
              "bandwidth_name": "cce-emqx",
              "bandwidth_size": 5,
              "bandwidth_sharetype": "PER",
              "eip_type": "5_bgp"
            }
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
          ## More content: https://support.huaweicloud.com/intl/en-us/usermanual-cce/cce_10_0380.html#section1
          storageClassName: csi-disk
          resources:
            requests:
              storage: 10Gi
          accessModes:
            - ReadWriteOnce
    dashboardServiceTemplate:
      metadata:
        annotations:
          ## Automatically create the associated ELB. For detailed field descriptions, please refer to: https://support.huaweicloud.com/intl/en-us/usermanual-cce/cce_10_0014.html#cce_10_0014__table939522754617
          kubernetes.io/elb.autocreate: |
            {
              "type": "public",
              "bandwidth_name": "cce-emqx",
              "bandwidth_size": 5,
              "bandwidth_sharetype": "PER",
              "eip_type": "5_bgp"
            }
      spec:
        type: LoadBalancer
    listenersServiceTemplate:
      metadata:
        annotations:
          ## Automatically create the associated ELB. For detailed field descriptions, please refer to: https://support.huaweicloud.com/intl/en-us/usermanual-cce/cce_10_0014.html#cce_10_0014__table939522754617
          kubernetes.io/elb.autocreate: |
            {
              "type": "public",
              "bandwidth_name": "cce-emqx",
              "bandwidth_size": 5,
              "bandwidth_sharetype": "PER",
              "eip_type": "5_bgp"
            }
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

  Access `http://198.18.3.10:18083` through a browser, and use the default username and password `admin/public` to log in to the EMQX console.

  ```
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

+ Create a new terminal window and post a message

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

Since Huawei ELB does not support TCP certificates, please refer to the document [Termination TLS](https://github.com/emqx/emqx-operator/discussions/312) to solve the problem of TCP certificate termination.
