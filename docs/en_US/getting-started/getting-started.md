# Product Overview

This project provides an Operator for managing EMQX clusters on Kubernetes.

## Deploy EMQX Operator

### Prepare the environment

Before deploying EMQX Operator, please confirm that the following components have been installed:

| Software                | Version Requirements |
|:-----------------------:|:--------------------:|
|  [Kubernetes](https://kubernetes.io/)    |  >= 1.24        |
|  [Helm](https://helm.sh)                 |  >= 3           |
|  [cert-manager](https://cert-manager.io) |  >= 1.1.6       |

> ### Why we need kubernetes 1.24:
>
> The `MixedProtocolLBService` feature is enabled by default in Kubernetes 1.24 and above. For its documentation, please refer to: [MixedProtocolLBService](https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/). The `MixedProtocolLBService` attribute allows different protocols to be used within the same Service instance of type `LoadBalancer`. Therefore, if the user deploys the EMQX cluster on Kubernetes and uses the `LoadBalancer` type of Service, there are both TCP and UDP protocols in the Service, please pay attention to upgrading the Kubernetes version to 1.24 or above, otherwise the Service creation will fail.
>
> **If user doesn't need `MixedProtocolLBService` feature, the EMQX Operator requires a Kubernetes cluster of version `>=1.21`.**

### Install EMQX Operator

> Make sure the [cert-manager](https://cert-manager.io) is ready

```bash
helm repo add emqx https://repos.emqx.io/charts
helm repo update
helm install emqx-operator emqx/emqx-operator --namespace emqx-operator-system --create-namespace
```

Check whether the cert-manager service is ready with the following command:

```bash
kubectl get pods -l "control-plane=controller-manager" -n emqx-operator-system
```

The output is similar to:

```bash
NAME                                                READY   STATUS    RESTARTS   AGE
emqx-operator-controller-manager-68b866c8bf-kd4g6   1/1     Running   0          15s
```

### Upgrade EMQX Operator

Execute the following command to upgrade EMQX Operator. If you want to specify the upgraded version, you only need to add parameter --version=x.x.x

```bash
helm upgrade emqx-operator emqx/emqx-operator -n emqx-operator-system
```

> Does not support version 1.x.x EMQX Operator upgrade to version 2.x.x .

### Uninstall EMQX Operator

Execute the following command to uninstall EMQX Operator

```bash
helm uninstall emqx-operator -n emqx-operator-system
```

## Deploy EMQX

### Deploy EMQX 5

1. Deploy EMQX

```bash
cat << "EOF" | kubectl apply -f -
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
   name: emqx
spec:
   image: emqx/emqx:5.0.14
EOF
```

For a complete example, please see [emqx-full.yaml](https://github.com/emqx/emqx-operator/blob/main/config/samples/emqx/v2alpha1/emqx-full.yaml), For detailed explanation of each field please refer to [v2alpha1-reference](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/reference/v2alpha1-reference.md).

2. Check whether the EMQX cluster is ready

```bash
kubectl get emqx emqx -o json | jq '.status.conditions[] | select( .type == "Running" and .status == "True")'
```

This may take a while for the command to execute successfully, as it needs to wait for all EMQX nodes to start and join the cluster.

The output is similar to:

```bash
{
   "lastTransitionTime": "2023-02-10T02:46:36Z",
   "lastUpdateTime": "2023-02-07T06:46:36Z",
   "message": "Cluster is running",
   "reason": "ClusterRunning",
   "status": "True",
   "type": "Running"
}
```

### Deploy EMQX 4

1. Deploy EMQX

```bash
cat << "EOF" | kubectl apply -f -
apiVersion: apps.emqx.io/v1beta4
kind: EmqxBroker
metadata:
   name: emqx
spec:
   template:
     spec:
       emqxContainer:
         image:
           repository: emqx/emqx-ee
           version: 4.4.14
EOF
```

For a complete example, please see [emqxbroker-full.yaml](https://github.com/emqx/emqx-operator/blob/main/config/samples/emqx/v1beta4/emqxenterprise-full.yaml), For a detailed explanation of each field please refer to [v1beta4-reference](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/reference/v1beta4-reference.md).

2. Check whether the EMQX cluster is ready

```bash
kubectl get emqxBroker emqx -o json | jq '.status.conditions[] | select( .type == "Running" and .status == "True")'
```

This may take a while for the command to execute successfully, as it needs to wait for all EMQX nodes to start and join the cluster.

The output is similar to:

```bash
{
  "lastTransitionTime": "2023-02-13T02:38:25Z",
  "lastUpdateTime": "2023-02-13T02:44:19Z",
  "message": "All resources are ready",
  "reason": "ClusterReady",
  "status": "True",
  "type": "Running"
}
```

### Deploy EMQX Enterprise 4

1. Deploy EMQX

```bash
cat << "EOF" | kubectl apply -f -
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
           version: 4.4.14
EOF
```

For a complete example, please see [emqxenterprise-full.yaml](https://github.com/emqx/emqx-operator/blob/main/config/samples/emqx/v1beta4/emqxenterprise-full.yaml), For a detailed explanation of each field please refer to [v1beta4-reference](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/reference/v1beta4-reference.md).

2. Check whether the EMQX cluster is ready

```bash
kubectl get emqxEnterprise emqx-ee -o json | jq '.status.conditions[] | select( .type == "Running" and .status == "True")'
```

This may take a while for the command to execute successfully, as it needs to wait for all EMQX nodes to start and join the cluster.

The output is similar to:

```bash
{
  "lastTransitionTime": "2023-02-13T02:38:25Z",
  "lastUpdateTime": "2023-02-13T02:44:19Z",
  "message": "All resources are ready",
  "reason": "ClusterReady",
  "status": "True",
  "type": "Running"
}
```
