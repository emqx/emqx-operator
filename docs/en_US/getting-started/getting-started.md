# Product Overview

This project provides an Operator for managing EMQX clusters on Kubernetes.

## Deploy EMQX Operator

### Prepare the environment

Before deploying EMQX Operator, please confirm that the following components have been installed:

| Software                | Version Requirements |
|:-----------------------:|:--------------------:|
|  [Helm](https://helm.sh)                 |  >= 3           |
|  [cert-manager](https://cert-manager.io) |  >= 1.1.6       |

### Install EMQX Operator

> Please make sure the [cert-manager](https://cert-manager.io) is ready

```bash
$ helm repo add emqx https://repos.emqx.io/charts
$ helm repo update
$ helm install emqx-operator emqx/emqx-operator --namespace emqx-operator-system --create-namespace
```

Check whether the cert-manager service is ready with the following command:

```bash
$ kubectl get pods -l "control-plane=controller-manager" -n emqx-operator-system

NAME                                                READY   STATUS    RESTARTS   AGE
emqx-operator-controller-manager-68b866c8bf-kd4g6   1/1     Running   0          15s
```

### Upgrade EMQX Operator

Execute the following command to upgrade EMQX Operator. If you want to specify the upgraded version, you only need to add the parameter --version=x.x.x

```bash
$ helm upgrade emqx-operator emqx/emqx-operator -n emqx-operator-system
```

> Does not support version 1.x.x EMQX Operator upgrade to version 2.x.x .

### Uninstall EMQX Operator

Execute the following command to uninstall EMQX Operator

```bash
$ helm uninstall emqx-operator -n emqx-operator-system
```

## Deploy EMQX

### Deploy EMQX 5

1. Deploy EMQX

   ```bash
   $ cat << "EOF" | kubectl apply -f -
   apiVersion: apps.emqx.io/v2alpha1
   kind: EMQX
   metadata:
      name: emqx
   spec:
      image: emqx:5.0
   EOF
   ```
   
   For more details please check the [reference document](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/reference/v2alpha1-reference.md).
   
2. Wait the EMQX cluster is running

   ```bash
   $ kubectl get emqx
   
   NAME   IMAGE      STATUS    AGE
   emqx   emqx:5.0   Running   2m55s
   ```

   Make sure the `STATUS` is `Running`, it maybe takes some time to wait for the EMQX cluster to be ready.

### Deploy EMQX 4

1. Deploy EMQX

   ```bash
   $ cat << "EOF" | kubectl apply -f -
   apiVersion: apps.emqx.io/v1beta4
   kind: EmqxBroker
   metadata:
      name: emqx
   spec:
      template:
        spec:
          emqxContainer:
            image:
              repository: emqx
              version: 4.4
   EOF
   ```
   
   For more details please check the [reference document](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/reference/v1beta4-reference.md).
   
2. Wait the EMQX cluster is running

   ```bash
   $ kubectl get emqxbrokers                                         

   NAME   STATUS   AGE
   emqx   Running  8m33s
   ```

   Make sure the `STATUS` is `Running`, it maybe takes some time to wait for the EMQX cluster to be ready.

### Deploy EMQX Enterprise 4

1. Deploy EMQX

    ```bash
    $ cat << "EOF" | kubectl apply -f -
    apiVersion: apps.emqx.io/v1beta4
    kind: EmqxEnterprise
    metadata:
       name: emqx-ee
    spec:
       template:
         spec:
           emqxContainer:
             image:
               repository: emqx/emqx-ee
               version: 4.4.15
    EOF
    ```

    For more details please check the [reference document](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/reference/v1beta4-reference.md).

2. Wait the EMQX cluster is running

   ```bash
   $ kubectl get emqxenterprises

   NAME      STATUS   AGE
   emqx-ee   Running  8m33s
   ```

   Make sure the `STATUS` is `Running`, it maybe takes some time to wait for the EMQX cluster to be ready.