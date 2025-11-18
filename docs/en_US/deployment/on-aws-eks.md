# Deploy EMQX on Amazon Elastic Kubernetes Service

EMQX Operator supports running on Amazon Container Service EKS (Elastic Kubernetes Service). Amazon EKS is a managed Kubernetes service that makes it easy to deploy, manage, and scale containerized applications. EKS provides the Kubernetes control plane and node groups, automatically handling node replacements, upgrades, and patching. It supports AWS services such as Load Balancers, RDS, and IAM, and integrates seamlessly with other Kubernetes ecosystem tools.

For a deeper introduction, please refer to [What is Amazon EKS](https://docs.aws.amazon.com/eks/latest/userguide/what-is-eks.html).

## Before You Begin

Before you begin, you must have the following:

- Activate Amazon Container Service and create an EKS cluster.<br/>Please refer to [Create an Amazon EKS cluster](https://docs.aws.amazon.com/eks/latest/userguide/getting-started.html) for more details.

- Connect to EKS cluster by installing kubectl tool locally.<br/>Refer to [Using kubectl to connect to the cluster](https://docs.aws.amazon.com/eks/latest/userguide/getting-started-console.html#eks-configure-kubectl) for more details.

- Deploy an AWS Load Balancer Controller on a cluster.<br/>See [Create a Network Load Balancer](https://docs.aws.amazon.com/eks/latest/userguide/network-load-balancing.html) for more details.

- Install the Amazon EBS CSI driver on the cluster.<br/>See [Amazon EBS CSI driver](https://docs.aws.amazon.com/eks/latest/userguide/ebs-csi.html) for further details.

- Install EMQX Operator.<br/>Please refer to [Install EMQX Operator](../getting-started/getting-started.md) for further details.

## Deploy EMQX Cluster Quickly

The following is the relevant configuration of an EMQX Custom Resource (CR).

+ Save the following content as a YAML file and deploy it with `kubectl apply`.

  ```yaml
  # Configure EBS StorageClass with WaitForFirstConsumer binding mode
  # This ensures volumes are created in the same AZ as the pods that will use them
  apiVersion: storage.k8s.io/v1
  kind: StorageClass
  metadata:
    name: ebs-sc
  provisioner: ebs.csi.aws.com
  volumeBindingMode: WaitForFirstConsumer
  ---
  apiVersion: apps.emqx.io/v2
  kind: EMQX
  metadata:
    name: emqx
  spec:
    image: emqx/emqx:6.0
    config:
      data: |
        license {
          key = "..."
        }
    coreTemplate:
      spec:
        ## EMQX custom resources do not support updating this field at runtime
        volumeClaimTemplates:
          storageClassName: ebs-sc
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
          service.beta.kubernetes.io/aws-load-balancer-type: external
          service.beta.kubernetes.io/aws-load-balancer-scheme: internet-facing
      spec:
        type: LoadBalancer
        ## More content: https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/guide/service/nlb/
        loadBalancerClass: service.k8s.aws/nlb
    listenersServiceTemplate:
      metadata:
        ## More content: https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/guide/service/annotations/
        annotations:
          ## Specifies whether the NLB is Internet-facing or internal. If not specified, defaults to internal.
          service.beta.kubernetes.io/aws-load-balancer-type: external
          service.beta.kubernetes.io/aws-load-balancer-scheme: internet-facing
      spec:
        type: LoadBalancer
        ## More content: https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/guide/service/nlb/
        loadBalancerClass: service.k8s.aws/nlb
  ```

+ Wait for EMQX cluster to become ready.

  Check the status of EMQX cluster through `kubectl get` command, make sure that `STATUS` is `Ready`. This may take some time.

  ```bash
  $ kubectl get emqx
  NAME   STATUS    AGE
  emqx   Ready     55s
  ```

+ Obtain external IP of the EMQX Dashboard and access it.

  EMQX Operator will create a Service resource for the EMQX Dashboard according to the `dashboardServiceTemplate` configuration.

  ```bash
  $ kubectl get svc emqx-dashboard -o json | jq -r '.status.loadBalancer.ingress[0].ip'
  192.168.1.200
  ```

  Access `http://192.168.1.200:18083` through the browser.

  Use the default username `admin` and password `public` to log into the EMQX Dashboard.

## Use MQTTX to Subscribe and Publish

[MQTTX CLI](https://mqttx.app/cli) is an open source MQTT 5.0 command line client tool, designed to help developers to start using MQTT services and applications quickly.

+ Obtain the external IP of the EMQX TCP listener.

  EMQX Operator will create a respective Service resource for the configured listeners.

  ```bash
  external_ip=$(kubectl get svc emqx-listeners -o json | jq -r '.status.loadBalancer.ingress[0].ip')
  ```

+ Subscribe to messages.

  ```bash
  $ mqttx sub -t 'hello' -h ${external_ip} -p 1883

  [10:00:25] › … Connecting...
  [10:00:25] › ✔ Connected
  [10:00:25] › … Subscribing to hello...
  [10:00:25] › ✔ Subscribed to hello
  ```

+ In a separate shell, connect to the EMQX cluster and publish a message.

  ```bash
  $ mqttx pub -t 'hello' -h ${external_ip} -p 1883 -m 'hello world'

  [10:00:58] › … Connecting...
  [10:00:58] › ✔ Connected
  [10:00:58] › … Message Publishing...
  [10:00:58] › ✔ Message published
  ```

+ Observe the subscriber client receiving the message.

  ```bash
  [10:00:58] › payload: hello world
  ```

## Terminate TLS Encryption with LoadBalancer

On Amazon EKS, you can use the NLB to terminate TLS encryption. Follow these steps:

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

    :::tip
    The value of `service.beta.kubernetes.io/aws-load-balancer-ssl-cert` is the ARN information recorded in step 1.
    :::
