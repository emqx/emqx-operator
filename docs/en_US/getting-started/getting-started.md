## Deploy EMQX Operator

In this section, we will walk you through the steps required to efficiently set up the environment for the EMQX Operator, install the Operator, and then use it to deploy EMQX. By following the guidelines outlined in this section, you will be able to effectively install and manage EMQX using the EMQX Operator.

### Prepare the Environment

Before deploying EMQX Operator, please confirm that the following components have been installed:

| Software                | Version Requirements |
|:-----------------------:|:--------------------:|
|  [Helm](https://helm.sh)                 |  3 or higher  |
|  [cert-manager](https://cert-manager.io) |  1.1.6 or higher  |

### Install EMQX Operator

Run the command below to install the Operator:

```bash
$ helm repo add emqx https://repos.emqx.io/charts
$ helm repo update
$ helm install emqx-operator emqx/emqx-operator --namespace emqx-operator-system --create-namespace
```

Wait till EMQX Operator is ready:

```bash
$ kubectl wait --for=condition=Ready pods -l "control-plane=controller-manager" -n emqx-operator-system

pod/emqx-operator-controller-manager-57bd7b8bd4-h2mcr condition met
```

Now that you have successfully installed the operator, you are ready to proceed to the next step. In the [Deploy EMQX](#deploy-emqx) section, you will learn how to use the EMQX Operator to deploy EMQX. 

Alternatively, if you are interested in learning how to upgrade or uninstall EMQX using the operator, you can continue reading this section. 

### Upgrade EMQX Operator

Execute the following command to upgrade EMQX Operator. If you want to specify the upgraded version, you only need to add the parameter `--version=x.x.x`.

```bash
$ helm upgrade emqx-operator emqx/emqx-operator -n emqx-operator-system
```

> Upgrade from version 1.x.x to version 2.x.x not supported. 

### Uninstall EMQX Operator

Execute the following command to uninstall EMQX Operator.

```bash
$ helm uninstall emqx-operator -n emqx-operator-system
```

## Deploy EMQX

### Deploy EMQX 5 

<!--Distinguish enterprise and opensource after 5.0 stablized-->

1. Deploy EMQX.

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

2. Wait the EMQX cluster is running.

   ```bash
   $ kubectl get emqx
   
   NAME   IMAGE      STATUS    AGE
   emqx   emqx:5.0   Running   2m55s
   ```

   Make sure the `STATUS` is `Running`, it maybe takes some time to wait for the EMQX cluster to be ready.

### Deploy EMQX 4

1. Deploy EMQX.

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

2. Wait the EMQX cluster is running.

   ```bash
   $ kubectl get emqxbrokers
   
   NAME   STATUS   AGE
   emqx   Running  8m33s
   ```

   Make sure the `STATUS` is `Running`, it maybe takes some time to wait for the EMQX cluster to be ready.

### Deploy EMQX Enterprise 4

1. Deploy EMQX.

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
