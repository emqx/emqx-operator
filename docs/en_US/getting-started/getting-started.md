# Overview

This project provides an operator for managing EMQX clusters on Kubernetes.

**Note**: EMQX Operator Controller requires Kubernetes v1.20.0 and up.

## Running the Operator

### Prepare

We use a [cert-manager](https://github.com/jetstack/cert-manager) for provisioning the certificates for the webhook server. You can follow [the cert-manager documentation](https://cert-manager.io/docs/installation/) to install it.

### Install EMQX Operator

1. Install by helm

 ```shell
 helm repo add emqx https://repos.emqx.io/charts
 helm repo update
 helm install emqx-operator emqx/emqx-operator --set installCRDs=true --namespace emqx-operator-system --create-namespace
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
        image: emqx/emqx:5.0.6
    EOF
    ```

    Full example please check [`emqx-full.yaml`](https://github.com/emqx/emqx-operator/blob/2.0.0/config/samples/emqx/v2alpha1/emqx-full.yaml).

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
          image: emqx/emqx:4.4.8
    EOF
    ```

    Full example please check [`emqxbroker-full.yaml`](https://github.com/emqx/emqx-operator/blob/2.0.0/config/samples/emqx/v1beta3/emqxbroker-full.yaml).

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
          image: emqx/emqx-ee:4.4.8
    EOF
    ```

    Full example please check [`emqxenterprise-full.yaml`](https://github.com/emqx/emqx-operator/blob/2.0.0/config/samples/emqx/v1beta3/emqxenterprise-full.yaml).

2. Check EMQX Custom Resource status

    ```
    $ kubectl get pods
    $ kubectl get emqxenterprise emqx -o json | jq ".status.emqxNodes"
    ```
