**Note**: EMQ X Operator requires Kubernetes v1.20.0 and up.

## Background

This article was deployed using minikube v1.20.0

## Deployment Operator

This project can be run inside a kubernetes cluster or outside of it, by taking either of the following two steps

### Deploy the operator in the Kubernetes cluster

1. Build the container image and push to the image repo

  ```bash
  IMG=emqx/emqx-operator-controller:0.1.0 make docker-build
  IMG=emqx/emqx-operator-controller:0.1.0 make docker-push
  ```

  **The `IMG` is related to the `spec.template.spec.containers[0].image` in `config/samples/operator/operator_deployment.yaml`**

2. Register the CustomResourceDefinitions into the Kubernetes Resources.

   `kubectl create -f config/samples/operator/apps.emqx.io_emqxes.yaml`

3. Enable RBAC rules for EMQ X Operator pods

   ```
   kubectl create -f config/samples/operator/operator_namespace.yaml
   kubectl create -f config/samples/operator/operator_service_account.yaml
   kubectl create -f config/samples/operator/operator_role.yaml
   kubectl create -f config/samples/operator/operator_role_binding.yaml
   ```

4. Deploy operator controller

   ```
   kubectl create -f config/samples/operator/operator_deployment.yaml
   ```

5. Check operator controller status

   ```
   kubectl get deployments controller-manager -n system
   NAME                 READY   UP-TO-DATE   AVAILABLE   AGE
   controller-manager   1/1     1            1           4h34m

   kubectl get pods -l "control-plane=controller-manager" -n system
   NAME                                  READY   STATUS    RESTARTS   AGE
   controller-manager-7f946dc6b4-l9vd2   1/1     Running   3          4h34m
   ```

### Deploy the operator out the Kubernetes cluster

> Prerequirements: Storage Class, Custom Resource Definition

1. Make sure kube config is configured properly

2. Run `main.go`

3. Create RBAC objects from manifest file

   ```
   kubectl create -f config/samples/operator/operator_namespace.yaml
   kubectl create -f config/samples/operator/operator_service_account.yaml
   kubectl create -f config/samples/operator/operator_role.yaml
   kubectl create -f config/samples/operator/operator_role_binding.yaml
   ```

## Deploy the EMQ X Broker

1. Enable RBAC rule for EMQ X pods

   ```
   kubectl create -f config/samples/emqx/emqx_serviceaccount.yaml
   kubectl create -f config/samples/emqx/emqx_role.yaml
   kubectl create -f config/samples/emqx/emqx_role_binding.yaml
   ```

2. Create EMQ X Custom Resource file like this

   ```
   cat config/samples/emqx/emqx_cr.yaml

   apiVersion: apps.emqx.io/v1alpha1
   kind: Emqx
   metadata:
     name: emqx
   spec:
     serviceAccountName: "emqx"
     image: registry-vpc.cn-hangzhou.aliyuncs.com/native/emqx:4.3.8
     replicas: 1
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
         address_type: dns
         suffix: pod.cluster.local
         app_name: emqx
         namespace: default
     env:
       - name: EMQX_NAME
         value: emqx
   ```

   > * [Details for *cluster* config](https://docs.emqx.io/en/broker/v4.3/configuration/configuration.html)
   > * [Details for *env* config](https://docs.emqx.io/en/broker/v4.3/configuration/configuration.html)

3. Deploy EMQ X Custom Resource and check EMQ X status

   ```
   kubectl create config/samples/emqx/emqx_cr.yaml
   emqx.apps.emqx.io/emqx created

   kubectl get pods
   NAME              READY   STATUS    RESTARTS   AGE
   emqx-0   1/1     Running   0          22m

   kubectl exec -it emqx-0 -- emqx_ctl status
   Node 'emqx@172-17-0-4.default.pod.cluster.local' 4.3.8 is started
   ```

4. If you want to expose the service to the public, then you can create a `k8s service` resource, here is an example of a `NodePort`

   ```
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
