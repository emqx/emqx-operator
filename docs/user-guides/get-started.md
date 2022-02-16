**Note**: EMQX Operator Controller requires Kubernetes v1.20.0 and up.

## Background

This article was deployed using minikube v1.20.0

## Deployment Operator Controller

This project can be run inside a kubernetes cluster

### Prepare

We using [cert manager](https://github.com/jetstack/cert-manager) for provisioning the certificates for the webhook server. You can follow [the cert manager documentation](https://cert-manager.io/docs/installation/) to install it.

### Default static install

> You don’t require any tweaking of the EMQX Operator Controller install parameters.

The default static configuration can be installed as follows:

```shell
$ kubectl apply -f https://raw.githubusercontent.com/emqx/emqx-operator/1.1.2/config/samples/operator/controller.yaml
```

### Installing with Helm

1.  Add the EMQX Helm repository:
   ```
   $ helm repo add emqx https://repos.emqx.io/charts
   $ helm repo update
   ```
2. Install EMQX Operator Controller by helm
   ```
   $ helm install emqx-operator emqx/emqx-operator \
      --set installCRDs=true \
      --namespace emqx-operator-system \
      --create-namespace
   ```

### Check EMQX Operator Controller status

   ```shell
   $ kubectl get pods -l "control-plane=controller-manager" -n emqx-operator-system
   NAME                                                READY   STATUS    RESTARTS   AGE
   emqx-operator-controller-manager-68b866c8bf-kd4g6   1/1     Running   0          15s
   ```

## Deploy the EMQX Broker

1. Create EMQX Custom Resource file like this

   ```shell
   $ cat https://raw.githubusercontent.com/emqx/emqx-operator/1.1.2/config/samples/emqx/v1beta2/emqx.yaml

   apiVersion: apps.emqx.io/v1beta2
   kind: EmqxBroker
   metadata:
     name: emqx
   spec:
     serviceAccountName: "emqx"
     image: emqx/emqx:4.3.11
     replicas: 3
     labels:
       cluster: emqx
     storage:
       storageClassName: standard
       resources:
         requests:
           storage: 20Mi
       accessModes:
       - ReadWriteMany
     emqxTemplate:
       listener:
         type: ClusterIP
         ports:
           mqtt: 1883
           mqtts: 8883
           ws: 8083
           wss: 8084
           dashboard: 18083
           api: 8081
       acl:
         - permission: allow
           username: "dashboard"
           action: subscribe
           topics:
             filter:
               - "$SYS/#"
               - "#"
         - permission: allow
           ipaddress: "127.0.0.1"
           topics:
             filter:
               - "$SYS/#"
             equal:
               - "#"
         - permission: deny
           action: subscribe
           topics:
             filter:
               - "$SYS/#"
             equal:
               - "#"
         - permission: allow
       plugins:
         - name: emqx_management
           enable: true
         - name: emqx_recon
           enable: true
         - name: emqx_retainer
           enable: true
         - name: emqx_dashboard
           enable: true
         - name: emqx_telemetry
           enable: true
         - name: emqx_rule_engine
           enable: true
         - name: emqx_bridge_mqtt
           enable: false
       modules:
         - name: emqx_mod_acl_internal
           enable: true
         - name: emqx_mod_presence
           enable: true
   ```

   > * [Details for *cluster* config](https://docs.emqx.io/en/broker/v4.3/configuration/configuration.html)
   > * [Details for *env* config](https://docs.emqx.io/en/broker/v4.3/configuration/configuration.html)

2. Deploy EMQX Custom Resource and check EMQX status

   ```shell
   $ kubectl apply https://raw.githubusercontent.com/emqx/emqx-operator/1.1.2/config/samples/emqx/v1beta2/emqx.yaml
   emqx.apps.emqx.io/emqx created

   $ kubectl get pods
   NAME              READY   STATUS    RESTARTS   AGE
   emqx-0   1/1     Running   0          22m
   emqx-1   1/1     Running   0          22m
   emqx-2   1/1     Running   0          22m

   $ kubectl exec -it emqx-0 -- emqx_ctl status
   Node 'emqx@emqx-0.emqx.default.svc.cluster.local' 4.3.11 is started

   $ kubectl exec -it emqx-0 -- emqx_ctl cluster status
   Cluster status: #{running_nodes =>
                         ['emqx@emqx-0.emqx.default.svc.cluster.local',
                          'emqx@emqx-1.emqx.default.svc.cluster.local',
                          'emqx@emqx-2.emqx.default.svc.cluster.local'],
                     stopped_nodes => []}
   ```

>**Note**:
>
>* EMQX Operator provides the default listener for EMQX Cluster to connect. The default `Type` of service is `ClusterIP`,which can be modified as `LoadBalance` or `NodePort`.
>* The ports about `ws`、`wss`、`mqtt`、`mqtts`、`dashboard`、`api` need to be set before deploying because they can't be updated while `EMQX Cluster` running.

### Scaling the cluster

[cluster-expansion](../cluster-expansion.md)
