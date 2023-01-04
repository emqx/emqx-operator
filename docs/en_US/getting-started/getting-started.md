# Overview

This project provides an operator for managing EMQX clusters on Kubernetes.

**Note**: EMQX Operator Controller requires Kubernetes v1.20.11 and up.

## Running the Operator

### Prepare

We use a [cert-manager](https://github.com/jetstack/cert-manager) for provisioning the certificates for the webhook server. You can follow [the cert-manager documentation](https://cert-manager.io/docs/installation/) to install it.

### Install EMQX Operator

1. Install by helm

 ```shell
 helm repo add emqx https://repos.emqx.io/charts
 helm repo update
 helm install emqx-operator emqx/emqx-operator --namespace emqx-operator-system --create-namespace
 ```
2. Wait EMQX Operator Controller running

```shell
$ kubectl get pods -l "control-plane=controller-manager" -n emqx-operator-system
NAME                                                READY   STATUS    RESTARTS   AGE
emqx-operator-controller-manager-68b866c8bf-kd4g6   1/1     Running   0          15s
```

## Deploy the EMQX

### Deploy EMQX 5

1. Deploy EMQX Custom Resource

    ```bash
    cat << "EOF" | kubectl apply -f -
      apiVersion: apps.emqx.io/v2alpha1
      kind: EMQX
      metadata:
        name: emqx
      spec:
        image: emqx/emqx:5.0.9
    EOF
    ```

    Full example please check [`emqx-full.yaml`](https://github.com/emqx/emqx-operator/blob/main/config/samples/emqx/v2alpha1/emqx-full.yaml).

    Detailed explanation of each field please check [v2alpha1-reference](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/reference/v2alpha1-reference.md)

2. Check EMQX Custom Resource status

    ```
    $ kubectl get pods
    $ kubectl get emqx emqx -o json | jq ".status.emqxNodes"
    ```

### Deploy EMQX 4

1. Deploy EMQX Custom Resource

    ```bash
    cat << "EOF" | kubectl apply -f -
      apiVersion: apps.emqx.io/v1beta3
      kind: EmqxBroker
      metadata:
        name: emqx
      spec:
        emqxTemplate:
          image: emqx/emqx:4.4.9
    EOF
    ```

    Full example please check [`emqxbroker-full.yaml`](https://github.com/emqx/emqx-operator/blob/main/config/samples/emqx/v1beta3/emqxbroker-full.yaml).

    Detailed explanation of each field please check [v1beta3-reference](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/reference/v1beta3-reference.md)

2. Check EMQX Custom Resource status

    ```
    $ kubectl get pods
    $ kubectl get emqxbroker emqx -o json | jq ".status.emqxNodes"
    ```

### Deploy EMQX Enterprise 4

1. Deploy EMQX Custom Resource

    ```bash
    cat << "EOF" | kubectl apply -f -
      apiVersion: apps.emqx.io/v1beta3
      kind: EmqxEnterprise
      metadata:
        name: emqx-ee
      spec:
        emqxTemplate:
          image: emqx/emqx-ee:4.4.9
    EOF
    ```

    Full example please check [`emqxenterprise-full.yaml`](https://github.com/emqx/emqx-operator/blob/main/config/samples/emqx/v1beta3/emqxenterprise-full.yaml).

    Detailed explanation of each field please check [v1beta3-reference](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/reference/v1beta3-reference.md)

2. Check EMQX Custom Resource status

    ```
    $ kubectl get pods
    $ kubectl get emqxenterprise emqx-ee -o json | jq ".status.emqxNodes"
    ```

## Remark
1. The `MixedProtocolLBService` feature is enabled by default in Kubernetes 1.24 and above. For its documentation, please refer to: [ MixedProtocolLBService ](https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/#feature-gates-for-alpha-or-beta-features). The `MixedProtocolLBService` attribute allows different protocols to be used within the same Service instance of type `LoadBalancer`. Therefore, if the user deploys the EMQX cluster on Kubernetes and uses the `LoadBalancer` type of Service, there are both TCP and UDP protocols in the Service, please pay attention to upgrading the Kubernetes version to 1.24 or above, otherwise the Service creation will fail.