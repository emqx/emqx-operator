**Note**: EMQ X Operator requires Kubernetes v1.20.0 and up.

## Background

This article was deployed using minikube v1.20.0

## Deployment Operator

This project can be run inside a kubernetes cluster or outside of it, by taking either of the following two steps

### Deploy the operator in the Kubernetes cluster

1. Build the container image and push to the image repo

  ```bash
  $ IMG=emqx/emqx-operator-controller:0.1.0 make docker-build
  $ IMG=emqx/emqx-operator-controller:0.1.0 make docker-push
  ```

  **The `IMG` is related to the `spec.template.spec.containers[0].image` in `https://raw.githubusercontent.com/emqx/emqx-operator/main/config/samples/operator/controller.yaml`**

2. Register the CustomResourceDefinitions into the Kubernetes Resources.

   ```shell
   $ kubectl create -f https://raw.githubusercontent.com/emqx/emqx-operator/main/config/crd/bases

   $ kubectl get crd
   NAME                  CREATED AT
   emqxbrokers.apps.emqx.io                    2021-12-01T03:15:52Z
   emqxenterprises.apps.emqx.io                2021-12-01T03:15:58Z
   ```

3. Deploy operator controller

   ```shell
   $ kubectl create -f https://raw.githubusercontent.com/emqx/emqx-operator/main/config/samples/operator
   ```

4. Check operator controller status

   ```shell
   $ kubectl get deployments controller-manager -n system
   NAME                 READY   UP-TO-DATE   AVAILABLE   AGE
   controller-manager   1/1     1            1           4h34m

   $ kubectl get pods -l "control-plane=controller-manager" -n system
   NAME                                  READY   STATUS    RESTARTS   AGE
   controller-manager-7f946dc6b4-l9vd2   1/1     Running   3          4h34m
   ```

5. Monitor the metrics about the EMQ X with [**Prometheus**](https://prometheus.io/)
   
  ```yaml

  apiVersion: apps.emqx.io/v1alpha2
  kind: EmqxBroker
  metadata:
    name: emqx
  spec:
   ...
   # Set the prometheus push gate server in env 
  env:
   ...
    # Make sure enable the plugin to support
    - name: EMQX_LOADED_PLUGINS
      value: emqx_prometheus
    # The configure for the prometheus
    - name: EMQX_PROMETHEUS__PUSH__GATEWAY__SERVER
      value: http://prometheus-pushgateway.prom.svc.cluster.local:9091
    PROMETHEUS__PUSH__GATEWAY__SERVER 

  ```

### Deploy the operator out the Kubernetes cluster

> Prerequirements: Storage Class, Custom Resource Definition

1. Make sure kube config is configured properly

2. Run `main.go`

3. Create RBAC objects from manifest file

   ```shell
   $ kubectl create -f https://raw.githubusercontent.com/emqx/emqx-operator/main/config/samples/operator/rbac.yaml
   ```

## Deploy the EMQ X Broker

1. Enable RBAC rule for EMQ X pods

   ```shell
   $ kubectl create -f https://raw.githubusercontent.com/emqx/emqx-operator/main/config/samples/emqx/rbac.yaml
   ```

2. Create EMQ X Custom Resource file like this

   ```shell
   $ cat https://raw.githubusercontent.com/emqx/emqx-operator/main/config/samples/emqx/emqx.yaml

   apiVersion: apps.emqx.io/v1alpha2
   kind: EmqxBroker
   metadata:
     name: emqx
   spec:
     serviceAccountName: "emqx"
     image: emqx/emqx:4.3.10
     replicas: 3
     labels:
       cluster: emqx
     storage:
       volumeClaimTemplate:
         spec:
           storageClassName: standard
           resources:
             requests:
               storage: 20Mi
           accessModes:
           - ReadWriteMany
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

3. Deploy EMQ X Custom Resource and check EMQ X status

   ```shell
   $ kubectl create https://raw.githubusercontent.com/emqx/emqx-operator/main/config/samples/emqx/emqx.yaml
   emqx.apps.emqx.io/emqx created

   $ kubectl get pods
   NAME              READY   STATUS    RESTARTS   AGE
   emqx-0   1/1     Running   0          22m
   emqx-1   1/1     Running   0          22m
   emqx-2   1/1     Running   0          22m

   $ kubectl exec -it emqx-0 -- emqx_ctl status
   Node 'emqx@emqx-0.emqx.default.svc.cluster.local' 4.3.8 is started

   $ kubectl exec -it emqx-0 -- emqx_ctl cluster status
   Cluster status: #{running_nodes =>
                         ['emqx@emqx-0.emqx.default.svc.cluster.local',
                          'emqx@emqx-1.emqx.default.svc.cluster.local',
                          'emqx@emqx-2.emqx.default.svc.cluster.local'],
                     stopped_nodes => []}
   ```

>**Note**:
>
>* EMQ X Operator provides the default listener for EMQ X Cluster to connect. The default `Type` of service is `ClusterIP`,which can be modified as `LoadBalance` or `NodePort`.
>* The ports about `ws`、`wss`、`mqtt`、`mqtts`、`dashboard`、`api` need to ensure before deploying which means they can't be updated while the EMQ X Cluster in the running status**

### Scaling the cluster

[cluster-expansion](../cluster-expansion.md)
