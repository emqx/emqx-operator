# Getting Started

In this section, we will walk you through the steps required to efficiently set up the environment for EMQX Operator, install it, and then use it to deploy EMQX. By following the guidelines outlined in this section, you will be able to install and manage EMQX effectively using the Operator.

## Prepare the Environment

Before deploying EMQX Operator, please confirm that the following components have been ready:

- A [Kubernetes](https://kubernetes.io/docs/concepts/overview/) environment running Kubernetes version 1.24 or higher.

- A [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl) tool that can access the Kubernetes cluster. You can check the status of the Kubernetes cluster using `kubectl cluster-info` command.

## Install EMQX Operator

1. Install the EMQX Operator.

   ```bash
   $ kubectl apply --server-side=true -f https://github.com/emqx/emqx-operator/releases/download/v2.3.0/install.yaml
   ```

2. Wait until EMQX Operator is ready.

   ```bash
   $ kubectl wait --for=condition=Ready pods -l "control-plane=controller-manager" -n emqx-operator-system
   pod/emqx-operator-controller-manager-57bd7b8bd4-h2mcr condition met
   ```

In the following section, you will learn how to use the EMQX Operator to deploy EMQX.

## Deploy EMQX

1. Save the following content as a YAML file and deploy it with the `kubectl apply`.

   ```yaml
   apiVersion: apps.emqx.io/v2
   kind: EMQX
   metadata:
      name: emqx-ee
   spec:
     image: emqx/emqx:6.0
     config:
       data: |
         license {
           key = "..."
         }
   ```

   For more details about the EMQX CRD, check out the [reference documentation](../reference/v2-reference.md).

2. Wait until the EMQX cluster is ready.

   ```bash
   $ kubectl get emqx

   NAME      IMAGE           STATUS   AGE
   emqx-ee   emqx/emqx:6.0   Ready    2m55s
   ```

   Make sure the `STATUS` is `Ready`, it may take some time for the EMQX cluster to become ready. A lot of things happen behind the scenes, refer to the [Hello EMQX Operator](./hello-emqx-operator.md) guide for more details.
