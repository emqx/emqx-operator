# Deploy EMQX On Amazon Elastic Kubernetes Service

Amazon Elastic Kubernetes Service (Amazon EKS) is a managed service that you can use to run Kubernetes on AWS without needing to install, operate, and maintain your own Kubernetes control plane or nodes. Kubernetes is an open-source system for automating the deployment, scaling, and management of containerized applications.

## Before Begin
Before you begin, you must have the following:

- An Amazon EKS cluster. For details: [Getting started with Amazon EKS](https://docs.aws.amazon.com/eks/latest/userguide/getting-started.html)

- A LoadBalancer. For details: [Load Balancer introduction](https://docs.aws.amazon.com/eks/latest/userguide/network-load-balancing.html)

- A StorageClass. For details: [Storage classes](https://docs.aws.amazon.com/eks/latest/userguide/storage-classes.html)

## Enable EMQX Cluster Persistence

EMQX custom resources use StoreClass to save the state of the EMQX runtime. Before starting, you prepared StoreClass. The following is an example of how to configure EMQX custom resources with "ebs-sc".

Here are the relevant configurations for EMQX Custom Resource. You can choose the corresponding APIVersion based on the version of EMQX you wish to deploy. For specific compatibility relationships, please refer to [EMQX Operator Compatibility](../README.md):

:::: tabs type:card 
::: tab v2alpha1

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
  name: emqx
spec:
  image: emqx:5.0
  coreTemplate:
    spec:
      volumeClaimTemplates:
        storageClassName: ebs-sc
        resources:
          requests:
            storage: 20Mi
        accessModes:
        - ReadWriteOnce
```
::: 
::: tab v1beta4

```yaml
apiVersion: apps.emqx.io/v1beta4
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  persistent:
    spec:
      storageClassName: ebs-sc
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
```
::: 
::::

## Access EMQX Cluster
In public cloud providers, you can use the LoadBalancer to access the EMQX cluster. For details: [Service Annotations](https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/guide/service/annotations/)

Modify the configuration of EMQX Custom Resource, add corresponding annotations, and set the Service Type to LoadBalancer as shown below:

:::: tabs type:card 
::: tab v2alpha1

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
  name: emqx
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: "external"
    service.beta.kubernetes.io/aws-load-balancer-nlb-target-type: "ip"
    service.beta.kubernetes.io/aws-load-balancer-scheme: internet-facing
    service.beta.kubernetes.io/aws-load-balancer-attributes: load_balancing.cross_zone.enabled=true
    service.beta.kubernetes.io/aws-load-balancer-target-group-attributes: preserve_client_ip.enabled=true
    service.beta.kubernetes.io/aws-load-balancer-attributes: deletion_protection.enabled=true
#   service.beta.kubernetes.io/aws-load-balancer-subnets: subnet-xxx1,subnet-xxx2
spec:
  image: emqx/emqx:5.0.14
  imagePullPolicy: IfNotPresent
  listenersServiceTemplate:
    spec:
      type: LoadBalancer
```
::: 
::: tab v1beta4

```yaml
apiVersion: apps.emqx.io/v1beta4
kind: EmqxEnterprise
metadata:
  name: emqx-ee
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: "external"
    service.beta.kubernetes.io/aws-load-balancer-nlb-target-type: "ip"
    service.beta.kubernetes.io/aws-load-balancer-scheme: internet-facing
    service.beta.kubernetes.io/aws-load-balancer-attributes: load_balancing.cross_zone.enabled=true
    service.beta.kubernetes.io/aws-load-balancer-target-group-attributes: preserve_client_ip.enabled=true
    service.beta.kubernetes.io/aws-load-balancer-attributes: deletion_protection.enabled=true
#   service.beta.kubernetes.io/aws-load-balancer-subnets: subnet-xxx1,subnet-xxx2
spec:
  template:
    spec:
      emqxContainer:
        image: 
          repository: emqx/emqx-ee
          version: 4.4.14
  serviceTemplate:
    spec:
      type: LoadBalancer
```
::: 
::::

## TLS termination  
In Amazon EKS, you can use the NLB to do TLS termination, which you can do in the following steps:

1. Import relevant certificates in [AWS Console](https://us-east-2.console.aws.amazon.com/acm/home), then enter the details page by clicking the certificate ID,  after that copy ARN, just as shown in the picture below:

![](./assets/cert.png)

2. Add some annotations in EMQX custom resources' metadata, just as shown in below:


    ```yaml
    service.beta.kubernetes.io/aws-load-balancer-ssl-cert: arn:aws:acm:us-west-2:arn:arn:aws:acm:us-east-1:609217282285:certificate/326649a0-f3b3-4bdb-a478-5691b4ba0ef3
    service.beta.kubernetes.io/aws-load-balancer-backend-protocol: tcp
    service.beta.kubernetes.io/aws-load-balancer-ssl-ports: 1883,mqtt-tls
    ```

    > The value of `service.beta.kubernetes.io/aws-load-balancer-ssl-cert` is the ARN information we copied in step 1.
