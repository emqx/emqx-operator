# Overview

This project provides an operator for managing EMQX clusters on Kubernetes.

**Note**: EMQX Operator Controller requires Kubernetes v1.20.0 and up.

## Running the Operator

### Prepare

We use a [cert-manager](https://github.com/jetstack/cert-manager) for provisioning the certificates for the webhook server. You can follow [the cert-manager documentation](https://cert-manager.io/docs/installation/) to install it.

### Installation

EMQX Operator provides helm and static yaml installation, we recommend using helm to install EMQX Operator

#### Using helm

 ```shell
 helm repo add emqx https://repos.emqx.io/charts
 helm repo update
 helm install emqx-operator emqx/emqx-operator --set installCRDs=true --namespace emqx-operator-system --create-namespace
 ```

#### Default static install

The default static configuration can be installed as follows(If you have already installed using Helm, please ignore this step):

```shell
kubectl apply -f "https://github.com/emqx/emqx-operator/releases/download/2.0.0-alpha.1/emqx-operator-controller.yaml"
```

### Check EMQX Operator Controller status

```shell
$ kubectl get pods -l "control-plane=controller-manager" -n emqx-operator-system
NAME                                                READY   STATUS    RESTARTS   AGE
emqx-operator-controller-manager-68b866c8bf-kd4g6   1/1     Running   0          15s
```

## Deploy the EMQX

1. Deploy EMQX Custom Resource

   ```bash
   cat << "EOF" | kubectl apply -f -
     apiVersion: apps.emqx.io/v2alpha1
     kind: EMQX
     metadata:
       name: emqx
     spec:
       emqxTemplate:
         image: emqx/emqx:5.0.6
   EOF
   ```

2. Check EMQX status

   ```bash
   $ kubectl get pods
   NAME                              READY   STATUS    RESTARTS        AGE
   emqx-core-0                       1/1     Running   0               75s
   emqx-core-1                       1/1     Running   0               75s
   emqx-core-2                       1/1     Running   0               75s
   emqx-replicant-6c8b4fccfb-bkk4s   1/1     Running   0               75s
   emqx-replicant-6c8b4fccfb-kmg9j   1/1     Running   0               75s
   emqx-replicant-6c8b4fccfb-zc929   1/1     Running   0               75s

   $ kubectl get emqx emqx -o json | jq ".status.emqxNodes"
   [
     {
       "node": "emqx@172.17.0.11",
       "node_status": "running",
       "otp_release": "24.2.1-1/12.2.1",
       "role": "replicant",
       "version": "5.0.6"
     },
     {
       "node": "emqx@172.17.0.12",
       "node_status": "running",
       "otp_release": "24.2.1-1/12.2.1",
       "role": "replicant",
       "version": "5.0.6"
     },
     {
       "node": "emqx@172.17.0.13",
       "node_status": "running",
       "otp_release": "24.2.1-1/12.2.1",
       "role": "replicant",
       "version": "5.0.6"
     },
     {
       "node": "emqx@emqx-core-0.emqx-headless.default.svc.cluster.local",
       "node_status": "running",
       "otp_release": "24.2.1-1/12.2.1",
       "role": "core",
       "version": "5.0.6"
     },
     {
       "node": "emqx@emqx-core-1.emqx-headless.default.svc.cluster.local",
       "node_status": "running",
       "otp_release": "24.2.1-1/12.2.1",
       "role": "core",
       "version": "5.0.6"
     },
     {
       "node": "emqx@emqx-core-2.emqx-headless.default.svc.cluster.local",
       "node_status": "running",
       "otp_release": "24.2.1-1/12.2.1",
       "role": "core",
       "version": "5.0.6"
     }
   ]
   ```
