# Deploy EMQX on Amazon Elastic Kubernetes Service

EMQX Operator supports deploying EMQX on Amazon Container Service EKS (Elastic Kubernetes Service). Amazon EKS is a managed Kubernetes service that makes it easy to deploy, manage, and scale containerized applications. EKS provides the Kubernetes control plane and node groups, automatically handling node replacements, upgrades, and patching. It supports AWS services such as Load Balancers, RDS, and IAM, and integrates seamlessly with other Kubernetes ecosystem tools. For details, please see [What is Amazon EKS](https://docs.aws.amazon.com/eks/latest/userguide/what-is-eks.html)

## Before You Begin

Before you begin, you must have the following:

- Activate Amazon Container Service and create an EKS cluster. For details, please refer to: [Create an Amazon EKS cluster](https://docs.aws.amazon.com/eks/latest/userguide/getting-started.html)

- Connect to EKS cluster by installing kubectl tool locally: For details, please refer to: [Using kubectl to connect to the cluster](https://docs.aws.amazon.com/eks/latest/userguide/getting-started-console.html#eks-configure-kubectl)

- Deploy an AWS Load Balancer Controller on a cluster, for details, please refer to: [Create a Network Load Balancer](https://docs.aws.amazon.com/eks/latest/userguide/network-load-balancing.html)

- Install EMQX Operator: For details, please refer to: [Install EMQX Operator](../getting-started/getting-started.md)

## Quickly Deploy An EMQX Cluster

The following is the relevant configuration of EMQX custom resources. You can select the corresponding APIVersion according to the EMQX version you want to deploy. For the specific compatibility relationship, please refer to [Compatibility list between EMQX and EMQX Operator](../index.md)

:::: tabs type:card
::: tab apps.emqx.io/v2beta1

+ Save the following content as a YAML file and deploy it via the `kubectl apply` command

  ```yaml
  apiVersion: apps.emqx.io/v2beta1
  kind: EMQX
  metadata:
    name: emqx
  spec:
    image: emqx:5.1
    coreTemplate:
      spec:
        ## EMQX custom resources do not support updating this field at runtime
        volumeClaimTemplates:
          ## More content: https://docs.aws.amazon.com/eks/latest/userguide/storage-classes.html
          ## Please manage the Amazon EBS CSI driver as an Amazon EKS add-on.
          ## For more documentation please refer to: https://docs.aws.amazon.com/zh_cn/eks/latest/userguide/managing-ebs-csi.html
          storageClassName: gp2
          resources:
            requests:
              storage: 10Gi
          accessModes:
            - ReadWriteOnce
    dashboardServiceTemplate:
      metadata:
        ## More content: https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/guide/service/annotations/
        annotations:
          ## Specifies whether the NLB is Internet-facing or internal. If not specified, defaults to internal.
          service.beta.kubernetes.io/aws-load-balancer-scheme: internet-facing
          ## Specify the availability zone to which the NLB will route traffic. Specify at least one subnet, either subnetID or subnetName (subnet name label) can be used.
          service.beta.kubernetes.io/aws-load-balancer-subnets: subnet-xxx1,subnet-xxx2
      spec:
        type: LoadBalancer
        ## More content: https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/guide/service/nlb/
        loadBalancerClass: service.k8s.aws/nlb
    listenersServiceTemplate:
      metadata:
        ## More content: https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/guide/service/annotations/
        annotations:
          ## Specifies whether the NLB is Internet-facing or internal. If not specified, defaults to internal.
          service.beta.kubernetes.io/aws-load-balancer-scheme: internet-facing
          ## Specify the availability zone to which the NLB will route traffic. Specify at least one subnet, either subnetID or subnetName (subnet name label) can be used.
          service.beta.kubernetes.io/aws-load-balancer-subnets: subnet-xxx1,subnet-xxx2
      spec:
        type: LoadBalancer
        ## More content: https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/guide/service/nlb/
        loadBalancerClass: service.k8s.aws/nlb
  ```

+ Wait for EMQX cluster to be ready, you can check the status of EMQX cluster through `kubectl get` command, please make sure that `STATUS` is `Running`, this may take some time

  ```bash
  $ kubectl get emqx
  NAME   IMAGE      STATUS    AGE
  emqx   emqx:5.1   Running   18m
  ```

+ Obtain Dashboard External IP of EMQX cluster and access EMQX console

  EMQX Operator will create two EMQX Service resources, one is emqx-dashboard and the other is emqx-listeners, corresponding to EMQX console and EMQX listening port respectively.

  ```bash
  $ kubectl get svc emqx-dashboard -o json | jq '.status.loadBalancer.ingress[0].ip'

  192.168.1.200
  ```

  Access `http://192.168.1.200:18083` through a browser, and use the default username and password `admin/public` to login EMQX console.

:::
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
        ## More content: https://docs.aws.amazon.com/eks/latest/userguide/storage-classes.html
        ## Please manage the Amazon EBS CSI driver as an Amazon EKS add-on.
        ## For more documentation please refer to: https://docs.aws.amazon.com/zh_cn/eks/latest/userguide/managing-ebs-csi.html
        storageClassName: gp2
        resources:
          requests:
            storage: 10Gi
        accessModes:
          - ReadWriteOnce
    template:
      spec:
        ## If persistence is enabled, you need to configure podSecurityContext.
        ## For details, please refer to the discussion: https://github.com/emqx/emqx-operator/discussions/716
        podSecurityContext:
          runAsUser: 1000
          runAsGroup: 1000
          fsGroup: 1000
          fsGroupChangePolicy: Always
          supplementalGroups:
            - 1000
        emqxContainer:
          image:
            repository: emqx/emqx-ee
            version: 4.4.14
    serviceTemplate:
      metadata:
        ## More content: https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/guide/service/annotations/
        annotations:
          ## Specifies whether the NLB is Internet-facing or internal. If not specified, defaults to internal.
          service.beta.kubernetes.io/aws-load-balancer-scheme: internet-facing
          ## Specify the availability zone to which the NLB will route traffic. Specify at least one subnet, either subnetID or subnetName (subnet name label) can be used.
          service.beta.kubernetes.io/aws-load-balancer-subnets: subnet-xxx1,subnet-xxx2
      spec:
        type: LoadBalancer
        ## More content: https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/guide/service/nlb/
        loadBalancerClass: service.k8s.aws/nlb
  ```

+ Wait for EMQX cluster to be ready, you can check the status of EMQX cluster through `kubectl get` command, please make sure that `STATUS` is `Running`, this may take some time

  ```bash
  $ kubectl get emqxenterprises
  NAME      STATUS   AGE
  emqx-ee   Running  26m
  ```

+ Obtain External IP of EMQX cluster and access EMQX console

  ```bash
  $ kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip'

  192.168.1.200
  ```

  Access `http://192.168.1.200:18083` through a browser, and use the default username and password `admin/public` to login EMQX console.

:::
::::

## Use MQTT X CLI To Publish/Subscribe Messages

[MQTT X CLI](https://mqttx.app/cli) is an open source MQTT 5.0 command line client tool, designed to help developers to more Quickly develop and debug MQTT services and applications.

+ Obtain External IP of EMQX cluster

  :::: tabs type:card
  ::: tab apps.emqx.io/v2beta1

  ```bash
  external_ip=$(kubectl get svc emqx-listeners -o json | jq '.status.loadBalancer.ingress[0].ip')
  ```
  :::
  ::: tab apps.emqx.io/v1beta4

  ```bash
  external_ip=$(kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip')
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

+ create a new terminal window and publish message

  ```bash
  $ mqttx pub -t 'hello' -h ${external_ip} -p 1883 -m 'hello world'

  [10:00:58] › … Connecting...
  [10:00:58] › ✔ Connected
  [10:00:58] › … Message Publishing...
  [10:00:58] › ✔ Message published
  ```

+ View messages received in the subscribed terminal window

  ```bash
  [10:00:58] › payload: hello world
  ```

## Terminate TLS Encryption With LoadBalancer

On Amazon EKS, you can use the NLB to do TLS termination, which you can do in the following steps:

1. Import relevant certificates in [AWS Console](https://us-east-2.console.aws.amazon.com/acm/home), then enter the details page by clicking the certificate ID, Then record the ARN information

    :::tip

    For the import format of certificates and keys, please refer to [import certificate](https://docs.aws.amazon.com/acm/latest/userguide/import-certificate-format.html)

    :::

2. Add some annotations in EMQX custom resources' metadata, just as shown below:

    ```yaml
    ## Specifies the ARN of one or more certificates managed by the AWS Certificate Manager.
    service.beta.kubernetes.io/aws-load-balancer-ssl-cert: arn:aws:acm:us-west-2:xxxxx:certificate/xxxxxxx
    ## Specifies whether to use TLS for the backend traffic between the load balancer and the kubernetes pods.
    service.beta.kubernetes.io/aws-load-balancer-backend-protocol: tcp
    ## Specifies a frontend port with a TLS listener. This means that accessing port 1883 through AWS NLB service requires TLS authentication,
    ## but direct access to K8S service port does not require TLS authentication
    service.beta.kubernetes.io/aws-load-balancer-ssl-ports: "1883"
    ```

    > The value of `service.beta.kubernetes.io/aws-load-balancer-ssl-cert` is the ARN information we record in step 1.
