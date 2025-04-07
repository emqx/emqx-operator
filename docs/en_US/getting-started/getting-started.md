# Getting Started

In this section, we will walk you through the steps required to efficiently set up the environment for the EMQX Operator, install the Operator, and then use it to deploy EMQX. By following the guidelines outlined in this section, you will be able to effectively install and manage EMQX using the EMQX Operator.

## Prepare the Environment

Before deploying EMQX Operator, please confirm that the following components have been ready:

- A running [Kubernetes cluster](https://kubernetes.io/docs/concepts/overview/), for a version of Kubernetes, please check [How to selector Kubernetes version](../index.md#how-to-selector-kubernetes-version)

- A [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl) tool that can access the Kubernetes cluster. You can check the status of the Kubernetes cluster using `kubectl cluster-info` command.

- [Helm](https://helm.sh) 3 or higher

## Install EMQX Operator

1. Install and start `cert-manager`.

   ::: tip
   `cert-manager` version `1.1.6` or higher is required. Skip this step if the `cert-manager` is already installed and started.
   :::

   You can use Helm to install `cert-manager`.

   ```bash
   $ helm repo add jetstack https://charts.jetstack.io
   $ helm repo update
   $ helm upgrade --install cert-manager jetstack/cert-manager \
     --namespace cert-manager \
     --create-namespace \
     --set crds.enabled=true
   ```

   Or you can follow the [cert-manager installation guide](https://cert-manager.io/docs/installation/) to install it.

   ::: warning
   If you install cert-manager on Google Kubernetes Engine (GKE) with default configuration may cause bootstrapping issues. Therefore, by adding the configuration of `--set global.leaderElection.namespace=cert-manager`, configure to use a different namespace in leader election. Please check [cert-manager compatibility](https://cert-manager.io/docs/installation/compatibility/)
   :::


2. Install the EMQX Operator with the command below:

   ```bash
   $ helm repo add emqx https://repos.emqx.io/charts
   $ helm repo update
   $ helm upgrade --install emqx-operator emqx/emqx-operator \
     --namespace emqx-operator-system \
     --create-namespace
   ```

3. Wait till EMQX Operator is ready:

   ```bash
   $ kubectl wait --for=condition=Ready pods -l "control-plane=controller-manager" -n emqx-operator-system

   pod/emqx-operator-controller-manager-57bd7b8bd4-h2mcr condition met
   ```

Now that you have successfully installed the operator, you are ready to proceed to the next step. In the [Deploy EMQX](#deploy-emqx) section, you will learn how to use the EMQX Operator to deploy EMQX.

Alternatively, if you are interested in learning how to upgrade or uninstall EMQX using the operator, you can continue reading this section.

## Deploy EMQX

:::: tabs type:card

::: tab EMQX Enterprise 5

1. Save the following content as a YAML file and deploy it with the `kubectl apply`.

   ```yaml
   apiVersion: apps.emqx.io/v2beta1
   kind: EMQX
   metadata:
      name: emqx-ee
   spec:
      image: emqx/emqx-enterprise:5.8
   ```

   For more details about the EMQX CRD, please check the [reference document](../reference/v2beta1-reference.md).

2. Wait the EMQX cluster is running.

   ```bash
   $ kubectl get emqx

   NAME      IMAGE                        STATUS    AGE
   emqx-ee   emqx/emqx-enterprise:5.8.6   Running   2m55s
   ```

   Make sure the `STATUS` is `Running`, it maybe takes some time to wait for the EMQX cluster to be ready.
:::

::: tab EMQX Open Source 5

1. Save the following content as a YAML file and deploy it with the `kubectl apply`.

   ```yaml
   apiVersion: apps.emqx.io/v2beta1
   kind: EMQX
   metadata:
      name: emqx
   spec:
      image: emqx/emqx:latest
   ```

   For more details about the EMQX CRD, please check the [reference document](../reference/v2beta1-reference.md).

2. Wait the EMQX cluster is running.

   ```bash
   $ kubectl get emqx

   NAME   IMAGE              STATUS    AGE
   emqx   emqx/emqx:latest   Running   2m55s
   ```

   Make sure the `STATUS` is `Running`, it maybe takes some time to wait for the EMQX cluster to be ready.
:::

::: tab EMQX Enterprise 4
1. Save the following content as a YAML file and deploy it with the `kubectl apply`.

    ```yaml
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
               version: 4.4.17
    ```

    For more details please check the [reference document](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/reference/v1beta4-reference.md).

2. Wait the EMQX cluster is running

   ```bash
   $ kubectl get emqxenterprises

   NAME      STATUS   AGE
   emqx-ee   Running  8m33s
   ```

   Make sure the `STATUS` is `Running`, it maybe takes some time to wait for the EMQX cluster to be ready.
:::

::: tab EMQX Open Source 4
1. Save the following content as a YAML file and deploy it with the `kubectl apply`.

   ```yaml
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
              version: "4.4.18"
   ```

   For more details please check the [reference document](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/reference/v1beta4-reference.md).

2. Wait the EMQX cluster is running.

   ```bash
   $ kubectl get emqxbrokers

   NAME   STATUS   AGE
   emqx   Running  8m33s
   ```

   Make sure the `STATUS` is `Running`, it maybe takes some time to wait for the EMQX cluster to be ready.
:::

::::
