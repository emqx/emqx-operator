# Overview

This project provides an operator for managing EMQX clusters on Kubernetes.

**Note**: EMQX Operator Controller requires Kubernetes v1.20.0 and up.

## Running the Operator

### Prepare

We use a [cert-manager](https://github.com/jetstack/cert-manager) for provisioning the certificates for the webhook server. You can follow [the cert-manager documentation](https://cert-manager.io/docs/installation/) to install it.

### Install

EMQX Operator provides helm and static yaml install, we recommend using helm to install EMQX Operator

#### Using helm

 ```shell
 helm repo add emqx https://repos.emqx.io/charts
 helm repo update
 helm install emqx-operator emqx/emqx-operator --set installCRDs=true --namespace emqx-operator-system --create-namespace
 ```

#### Default static install

The default static configuration can be installed as follows(If you have already installed using Helm, please ignore this step):

```shell
kubectl apply -f "https://github.com/emqx/emqx-operator/releases/download/1.2.5/emqx-operator-controller.yaml"
```

### Check EMQX Operator Controller status

```shell
$ kubectl get pods -l "control-plane=controller-manager" -n emqx-operator-system
NAME                                                READY   STATUS    RESTARTS   AGE
emqx-operator-controller-manager-68b866c8bf-kd4g6   1/1     Running   0          15s
```

## Deploy the EMQX Enterprise

1. Deploy EMQX Enterprise Custom Resource  

   ```bash
   cat << "EOF" | kubectl apply -f -
   apiVersion: apps.emqx.io/v1beta3
   kind: EmqxEnterprise
   metadata:
     name: emqx-ee
     labels:
       "foo": "bar"
   spec:
     emqxTemplate:
       image: emqx/emqx-ee:4.4.7
   EOF
   ```

2. Check EMQX status

   ```bash  
   $ kubectl get pods  
   NAME              READY   STATUS    RESTARTS   AGE  
   emqx-ee-0   2/2     Running   0          22m  
   emqx-ee-1   2/2     Running   0          22m  
   emqx-ee-2   2/2     Running   0          22m  

   $ kubectl exec -it emqx-ee-0 -c emqx -- emqx_ctl status  
   Node 'emqx-ee@emqx-ee-0.emqx-ee-headless.default.svc.cluster.local' 4.4.7 is started  

   $ kubectl exec -it emqx-ee-0 -c emqx -- emqx_ctl cluster status  
   Cluster status: #{running_nodes =>
                      ['emqx-ee@emqx-ee-0.emqx-ee-headless.default.svc.cluster.local',
                       'emqx-ee@emqx-ee-1.emqx-ee-headless.default.svc.cluster.local',
                       'emqx-ee@emqx-ee-2.emqx-ee-headless.default.svc.cluster.local'],
                  stopped_nodes => []}
   ```


## Deploy the EMQX Broker

1. Deploy EMQX Broker Custom Resource

   ```bash
   cat << "EOF" | kubectl apply -f -
   apiVersion: apps.emqx.io/v1beta3
   kind: EmqxBroker
   metadata:
     name: emqx
     labels:
       "foo": "bar"
   spec:
     emqxTemplate:
       image: emqx/emqx:4.4.7
   EOF
   ```

2. Check EMQX status

   ```bash
   $ kubectl get pods
   NAME              READY   STATUS    RESTARTS   AGE
   emqx-0   2/2     Running   0          22m
   emqx-1   2/2     Running   0          22m
   emqx-2   2/2     Running   0          22m

   $ kubectl exec -it emqx-0 -c emqx -- emqx_ctl status
   Node 'emqx@emqx-0.emqx-headless.default.svc.cluster.local' 4.4.7 is started

   $ kubectl exec -it emqx-0 -c emqx -- emqx_ctl cluster status
   Cluster status: #{running_nodes =>
                      ['emqx@emqx-0.emqx-headless.default.svc.cluster.local',
                       'emqx@emqx-1.emqx-headless.default.svc.cluster.local',
                       'emqx@emqx-2.emqx-headless.default.svc.cluster.local'],
                  stopped_nodes => []} 
   ```
