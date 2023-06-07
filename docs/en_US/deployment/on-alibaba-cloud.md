# Deploy EMQX On Alibaba Cloud Container Service

EMQX Operator supports deploying EMQX on Alibaba Cloud Container Service for Kubernetes. Alibaba Cloud Container Service for Kubernetes (Alibaba Cloud Container Service for Kubernetes, ACK for short) is the first batch of service platforms in the world that have passed the Kubernetes conformance certification. It provides high-performance container application management services and supports the life of enterprise-level Kubernetes containerized applications. Cycle management allows you to easily and efficiently run Kubernetes containerized applications on the cloud. For details, please see [What is Container Service for Kubernetes](https://www.alibabacloud.com/help/en/container-service-for-kubernetes/latest/what-is-container-service-for-kubernetes)

## Prerequisites

Before starting, you need to prepare the following:

- Activate Alibaba Cloud Container Service and create an ACK cluster. For details, please refer to: [Using Container Service for Kubernetes for the First Time](https://www.alibabacloud.com/help/en/container-service-for-kubernetes/latest/quick-start-for-first-time-users)

- Connect to the ACK cluster through the kubectl command. You can install the kubectl tool locally and obtain the KubeConfig of the cluster to connect to the cluster, or use CloudShell on the container service ACK console to manage the cluster through kubectl.

  - Connect to the ACK cluster by installing the kubectl tool locally: For details, please refer to: [Get the cluster KubeConfig and connect to the cluster through the kubectl tool](https://www.alibabacloud.com/help/en/container-service-for-kubernetes/latest/connect-to-ack-clusters-by-using-kubectl)
  - Connect to the ACK cluster through CloudShell: For details, please refer to: [Managing Kubernetes clusters through kubectl on CloudShell](https://www.alibabacloud.com/help/en/container-service-for-kubernetes/latest/use-kubectl-on-cloud-shell-to-manage-ack-clusters)

- Install EMQX Operator: For details, please refer to: [Install EMQX Operator](../getting-started/getting-started.md)

## Quickly Deploy An EMQX Cluster

The following is the relevant configuration of EMQX custom resources. You can select the corresponding APIVersion according to the EMQX version you want to deploy. For the specific compatibility relationship, please refer to [Compatibility list between EMQX and EMQX Operator](../README.md):

:::: tabs type:card
::: tab apps.emqx.io/v1beta4

+ Save the following content as a YAML file and deploy it via the `kubectl apply` command.

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
        ## More content: https://help.aliyun.com/document_detail/134722.html
        storageClassName: alibabacloud-cnfs-nas
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
          ## The regions and availability zones supported by NLB can be viewed by logging in to the NLB console, and at least two availability zones are required. Multiple availability ranges are separated by commas, such as cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321 .
          service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}"
      spec:
        type: LoadBalancer
        ## More content: https://www.alibabacloud.com/help/en/container-service-for-kubernetes/latest/configurenlbthroughannotation
        loadBalancerClass: "alibabacloud.com/nlb"
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

+ Save the following content as a YAML file and deploy it via the `kubectl apply` command.

  ```yaml
  apiVersion: apps.emqx.io/v2alpha1
  kind: EMQX
  metadata:
    name: emqx
  spec:
    image: "emqx:5.0"
    coreTemplate:
      spec:
        ## EMQX custom resources do not support updating this field at runtime
        volumeClaimTemplates:
          ## More content: https://help.aliyun.com/document_detail/134722.html
          storageClassName: alibabacloud-cnfs-nas
          resources:
            requests:
              storage: 10Gi
          accessModes:
            - ReadWriteOnce
    dashboardServiceTemplate:
      metadata:
        annotations:
          ## The regions and availability zones supported by NLB can be viewed by logging in to the NLB console, and at least two availability zones are required. Multiple availability ranges are separated by commas, such as cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321 .
          service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}"
      spec:
        type: LoadBalancer
        ## More content: https://www.alibabacloud.com/help/en/container-service-for-kubernetes/latest/configurenlbthroughannotation
        loadBalancerClass: "alibabacloud.com/nlb"
    listenersServiceTemplate:
      metadata:
        annotations:
          ## The regions and availability zones supported by NLB can be viewed by logging in to the NLB console, and at least two availability zones are required. Multiple availability ranges are separated by commas, such as cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321 .
          service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}"
      spec:
        type: LoadBalancer
        ## More content: https://www.alibabacloud.com/help/en/container-service-for-kubernetes/latest/configurenlbthroughannotation
        loadBalancerClass: "alibabacloud.com/nlb"
  ```

+ Wait for the EMQX cluster to be ready, you can check the status of the EMQX cluster through `kubectl get` command, please make sure `STATUS` is `Running`, this may take some time

  ```bash
  $ kubectl get emqx
  NAME   IMAGE      STATUS    AGE
  emqx   emqx:5.0   Running   2m55s
  ```

+ Get EMQX cluster Dashboard External IP, access EMQX console

   EMQX Operator will create two EMQX Service resources, one is `emqx-dashboard` and the other is `emqx-listeners`, corresponding to EMQX console and EMQX listening port respectively.

   ```bash
   $ external_ip=$(kubectl get svc emqx-dashboard -o json | jq '.status.loadBalancer.ingress[0].ip')
   $ echo $external_ip

   198.18.3.10
   ```

   Access via browser`http://198.18.3.10:18083`, use the default username and password `admin/public` to log in to the EMQX console.

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

## Terminate TLS encryption with LoadBalancer

Before using NLB to terminate TLS traffic, you need to create a TLS certificate first, Alibaba Cloud's [Digital Certificate Management Service](https://us-east-2.console.aws.amazon.com/acm/home) console, You can import a self-signed certificate or purchase a certificate. After the certificate is imported, click the certificate details to obtain the certificate ID.

> Since the associated DNS domain name will change every time the NLB is recreated, if a self-signed certificate is used, for the convenience of testing, it is recommended to set the domain name bound to the certificate to `*.cn-shanghai.nlb.aliyuncs.com `

Modify the configuration of EMQX Custom Resource, add relevant annotations to EMQX Custom Resource, and update the listening port in Service Template.


:::: tabs type:card
::: tab apps.emqx.io/v1beta4

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
       storageClassName: alibabacloud-cnfs-nas
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
       annotations:
         ## The regions and availability zones supported by NLB can be viewed by logging in to the NLB console, and at least two availability zones are required. Multiple availability ranges are separated by commas, such as cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321 .
         service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}"
         ## If the cluster is a Mainland China Region, the combined certificate ID is ${your-cert-id}-cn-hangzhou. If the cluster is another Region, the combined certificate ID is ${your-cert-id}-ap-southeast-1.
         service.beta.kubernetes.io/alibaba-cloud-loadbalancer-cert-id: "${combined certificate ID}"
         ## The SSL port that the LoadBalancer listens on
         service.beta.kubernetes.io/alibaba-cloud-loadbalancer-protocol-port: "tcpssl:30883"
     spec:
       type: LoadBalancer
       ## More content: https://www.alibabacloud.com/help/en/container-service-for-kubernetes/latest/configurenlbthroughannotation
       loadBalancerClass: "alibabacloud.com/nlb"
       ports:
         - name: tcpssl
           ## The SSL port that the LoadBalancer listens on
           port: 30883
           protocol: TCP
           ## MQTT TCP port
           targetPort: 1883
```
:::

::: tab apps.emqx.io/v2alpha1

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
   name: emqx
spec:
   image: "emqx:5.0"
   coreTemplate:
     spec:
       ## EMQX custom resources do not support updating this field at runtime
       volumeClaimTemplates:
         storageClassName: alibabacloud-cnfs-nas
         resources:
           requests:
             storage: 20Mi
         accessModes:
           - ReadWriteOnce
   dashboardServiceTemplate:
     metadata:
       annotations:
         ## The regions and availability zones supported by NLB can be viewed by logging in to the NLB console, and at least two availability zones are required. Multiple availability ranges are separated by commas, such as cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321 .
         service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}"
     spec:
       type: LoadBalancer
       ## More content: https://www.alibabacloud.com/help/en/container-service-for-kubernetes/latest/configurenlbthroughannotation
       loadBalancerClass: "alibabacloud.com/nlb"
   listenersServiceTemplate:
     metadata:
       annotations:
         ## The regions and availability zones supported by NLB can be viewed by logging in to the NLB console, and at least two availability zones are required. Multiple availability ranges are separated by commas, such as cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321 .
         service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}"
         ## If the cluster is a Mainland China Region, the combined certificate ID is ${your-cert-id}-cn-hangzhou. If the cluster is another Region, the combined certificate ID is ${your-cert-id}-ap-southeast-1.
         service.beta.kubernetes.io/alibaba-cloud-loadbalancer-cert-id: "${combined certificate ID}"
         ## The SSL port that the LoadBalancer listens on
         service.beta.kubernetes.io/alibaba-cloud-loadbalancer-protocol-port: "tcpssl:30883"
     spec:
       type: LoadBalancer
       ## More content: https://www.alibabacloud.com/help/en/container-service-for-kubernetes/latest/configurenlbthroughannotation
       loadBalancerClass: "alibabacloud.com/nlb"
       ports:
         - name: tcpssl
           ## The SSL port that the LoadBalancer listens on
           port: 30883
           protocol: TCP
           ## MQTT TCP port
           targetPort: 1883
```

:::
::::

<!-- TODO -->
<!-- LoadBalancer directly connected to Pod -->