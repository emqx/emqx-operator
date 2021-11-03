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
   $ kubectl create -f https://raw.githubusercontent.com/emqx/emqx-operator/main/config/crd/bases/apps.emqx.io_emqxbrokers.yaml

   $ kubectl get crd
   NAME                  CREATED AT
   emqxbrokers.apps.emqx.io   2021-10-27T08:02:45Z
   ```


3. Enable RBAC rules for EMQ X Operator pods

   ```shell
   $ kubectl create -f https://raw.githubusercontent.com/emqx/emqx-operator/main/config/samples/operator/namespace.yaml
   $ kubectl create -f https://raw.githubusercontent.com/emqx/emqx-operator/main/config/samples/operator/rbac.yaml
   ```

4. Deploy operator controller

   ```shell
   $ kubectl create -f https://raw.githubusercontent.com/emqx/emqx-operator/main/config/samples/operator/controller.yaml
   ```

5. Check operator controller status

   ```shell
   $ kubectl get deployments controller-manager -n system
   NAME                 READY   UP-TO-DATE   AVAILABLE   AGE
   controller-manager   1/1     1            1           4h34m

   $ kubectl get pods -l "control-plane=controller-manager" -n system
   NAME                                  READY   STATUS    RESTARTS   AGE
   controller-manager-7f946dc6b4-l9vd2   1/1     Running   3          4h34m
   ```

### Deploy the operator out the Kubernetes cluster

> Prerequirements: Storage Class, Custom Resource Definition

1. Make sure kube config is configured properly

2. Run `main.go`

3. Create RBAC objects from manifest file

   ```shell
   $ kubectl create -f https://raw.githubusercontent.com/emqx/emqx-operator/main/config/samples/operator/operator_namespace.yaml
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

   apiVersion: apps.emqx.io/v1alpha1
   kind: EmqxBroker
   metadata:
     name: emqx
   spec:
     serviceAccountName: "emqx"
     image: emqx/emqx:4.3.8
     replicas: 3
     labels:
       cluster: emqx
     storage:
       volumeClaimTemplate:
         spec:
           storageClassName: standard
           resources:
             requests:
               storage: 64Mi
           accessModes:
           - ReadWriteMany
     cluster:
       name: emqx
       k8s:
         apiserver: "https://kubernetes.default.svc:443"
         service_name: emqx
         address_type: hostname
         suffix: svc.cluster.local
         app_name: emqx
         namespace: default
     env:
       - name: EMQX_NAME
         value: emqx
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

   $ kubectl exec -it emqx-0 -- emqx_ctl status
   Node 'emqx@emqx-0.emqx.default.svc.cluster.local' 4.3.8 is started

   $ kubectl exec -it emqx-0 -- emqx_ctl cluster status
   Cluster status: #{running_nodes =>
                         ['emqx@emqx-0.emqx.default.svc.cluster.local',
                          'emqx@emqx-1.emqx.default.svc.cluster.local',
                          'emqx@emqx-2.emqx.default.svc.cluster.local'],
                     stopped_nodes => []}
   ```

4. If you want to expose the service to the public, then you can create a `k8s service` resource, here is an example of a `NodePort`

   ```shell
   apiVersion: v1
   kind: Service
   metadata:
     name: emqx-lb
     namespace: default
   spec:
     selector:
        cluster: emqx
     ports:
       - name: tcp
         port: 1883
         protocol: TCP
         targetPort: 1883
       - name: tcps
         port: 8883
         protocol: TCP
         targetPort: 8883
       - name: ws
         port: 8083
         protocol: TCP
         targetPort: 8083
       - name: wss
         port: 8084
         protocol: TCP
         targetPort: 8084
       - name: dashboard
         port: 18083
         protocol: TCP
         targetPort: 18083
     type: NodePort
   ```

### Scaling the cluster

[cluster-expansion](../cluster-expansion.md)
