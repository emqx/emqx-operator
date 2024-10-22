# Hello EMQX Operator

In this guide, we will walk you through deploying Kubernetes locally using Kind, installing the EMQX Operator, and using it to deploy EMQX.

Kind (Kubernetes in docker) is a tool for running local Kubernetes clusters using Docker container nodes. Kind is primarily designed for testing Kubernetes itself, but it can also be used for local development or CI, please do not use Kind in production environments.

## Install tools

### Docker

On Linux: [Docker installation guide](https://docs.docker.com/desktop/install/linux/)

On MacOS: [Orbstack](https://orbstack.dev/)

### Kind

Install Kind by following the [Kind installation guide](https://kind.sigs.k8s.io/docs/user/quick-start/#installing-from-release-binaries)

### kubectl

Install kubectl by following the [kubectl installation guide](https://kubernetes.io/docs/tasks/tools/#kubectl)

### Helm

Install Helm 3 or higher by following the [Helm installation guide](https://helm.sh/docs/intro/install/)

## Prepare the environment

### Create a Kubernetes cluster

Create a Kubernetes cluster using Kind:

```bash
$ kind create cluster
```

After the cluster is created, you can use the following command to verify the cluster status:

```bash
$ kubectl cluster-info
```

Now you have a Kubernetes cluster running locally, you can check [kubernetes documents](https://kubernetes.io/docs/home/) for more information.

### Install cert-manager

[cert-manager](https://cert-manager.io/docs/) is a Kubernetes add-on to automate the management and issuance of TLS certificates from various issuing sources. It will ensure certificates are valid and up to date.

EMQX operator needs cert-manager for managing certificates, you can install cert-manager using Helm:

```bash
$ helm repo add jetstack https://charts.jetstack.io
$ helm repo update
$ helm upgrade --install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --set crds.enabled=true
```

Or follow the [cert-manager installation guide](https://cert-manager.io/docs/installation/).

### Install EMQX Operator

Run the following command to install the EMQX Operator:

```bash
$ helm repo add emqx https://repos.emqx.io/charts
$ helm repo update
$ helm upgrade --install emqx-operator emqx/emqx-operator \
  --namespace emqx-operator-system \
  --create-namespace \
  --set development=true
```

Wait for the EMQX Operator to be installed, you can check the status by running:

```bash
$ kubectl wait --for=condition=Ready pods -l "control-plane=controller-manager" -n emqx-operator-system
```

Now you have successfully installed the EMQX Operator, you can continue to the next step. In the Deploy EMQX section, you will learn how to deploy EMQX using the EMQX Operator.

## Deploy EMQX

EMQX operator provides a custom resource definition (CRD) called `EMQX`, which allows you to define and manage EMQX clusters in Kubernetes.

Create a file named `emqx.yaml` with the following content:

```yaml
apiVersion: apps.emqx.io/v2beta1
kind: EMQX
metadata:
  name: emqx-ee
spec:
  image: emqx/emqx-enterprise:5.8
  config:
    data:
      log.console.level = debug
  coreTemplate:
    spec:
      replicas: 2
  replicantTemplate:
    spec:
      replicas: 3
```

In the `emqx.yaml` file, you define the `image` field to specify the EMQX image to use. And also define the `config` field to specify the EMQX configuration, in this example, set the `log.console.level` to `debug`. You also define the `coreTemplate` and `replicantTemplate` to specify the number of replicas for the core and replicant nodes.

EMQX custom resource definition (CRD) also provides `dashboardServiceTemplate` and `listenersServiceTemplate` to configure the EMQX dashboard and listeners service. For
more configuration options, you can refer to the [EMQX Operator documentation](https://docs.emqx.com/en/emqx-operator/latest/reference/v2beta1-reference.html).

And use the `kubectl apply` command to deploy EMQX:

```bash
$ kubectl apply -f emqx.yaml
```

After the EMQX cluster is deployed, you can check the status by running:

```bash
$ kubectl get emqx
NAME      STATUS   AGE
emqx-ee   Ready    110s
```

You should see the EMQX cluster status as `Ready`, which may take some time for the EMQX cluster to be ready.

## What happens when you deploy EMQX

When you deploy the EMQX custom resource, for EMQX core nodes, the EMQX Operator will create a StatefulSet, for EMQX replicant nodes, the EMQX Operator will create a ReplicaSet. The EMQX Operator will also create the necessary services, and other resources

You can check the resources created by the EMQX Operator by running:

```bash
$ kubectl get statefulsets,replicasets,services,pods -l apps.emqx.io/instance=emqx-ee -o wide
NAME                                       READY   AGE     CONTAINERS   IMAGES
statefulset.apps/emqx-ee-core-7494d76574   2/2     4m20s   emqx         emqx/emqx-enterprise:5.8

NAME                                          DESIRED   CURRENT   READY   AGE     CONTAINERS   IMAGES                     SELECTOR
replicaset.apps/emqx-ee-replicant-6c79c4c45   3         3         3       3m23s   emqx         emqx/emqx-enterprise:5.8   apps.emqx.io/db-role=replicant,apps.emqx.io/instance=emqx-ee,apps.emqx.io/managed-by=emqx-operator,apps.emqx.io/pod-template-hash=6c79c4c45

NAME                        TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)                               AGE     SELECTOR
service/emqx-ee-dashboard   ClusterIP   10.96.118.2   <none>        18083/TCP                             3m23s   apps.emqx.io/db-role=core,apps.emqx.io/instance=emqx-ee,apps.emqx.io/managed-by=emqx-operator
service/emqx-ee-headless    ClusterIP   None          <none>        4370/TCP,5369/TCP                     4m20s   apps.emqx.io/db-role=core,apps.emqx.io/instance=emqx-ee,apps.emqx.io/managed-by=emqx-operator
service/emqx-ee-listeners   ClusterIP   10.96.1.74    <none>        8883/TCP,1883/TCP,8083/TCP,8084/TCP   3m23s   apps.emqx.io/db-role=replicant,apps.emqx.io/instance=emqx-ee,apps.emqx.io/managed-by=emqx-operator

NAME                                    READY   STATUS    RESTARTS   AGE     IP            NODE                 NOMINATED NODE   READINESS GATES
pod/emqx-ee-core-7494d76574-0           1/1     Running   0          4m20s   10.244.0.10   kind-control-plane   <none>           1/1
pod/emqx-ee-core-7494d76574-1           1/1     Running   0          4m20s   10.244.0.11   kind-control-plane   <none>           1/1
pod/emqx-ee-replicant-6c79c4c45-lrt68   1/1     Running   0          3m23s   10.244.0.13   kind-control-plane   <none>           1/1
pod/emqx-ee-replicant-6c79c4c45-r2ffd   1/1     Running   0          3m23s   10.244.0.14   kind-control-plane   <none>           1/1
pod/emqx-ee-replicant-6c79c4c45-sqpzg   1/1     Running   0          3m23s   10.244.0.12   kind-control-plane   <none>           1/1
```

## Manager EMQX configuration

### Check EMQX configuration

In above example, we set the `log.console.level` to `debug` in the EMQX configuration, you can check the EMQX configuration by running the following command:

```bash
$ kubectl get emqx emqx-ee -o json | jq '.spec.config.data'
```

This will output the EMQX configuration data that you defined in the `emqx.yaml` file. And EMQX operator will record the EMQX configuration in the `.metadata.annotations` field, you can check the EMQX configuration by running the following command:

```bash
$ kubectl get emqx emqx-ee -o json | jq '.metadata.annotations."apps.emqx.io/last-emqx-configuration"'
```

This output should be the same as the output of the previous command.

### Update EMQX configuration

EMQX operator also provides a way to update the EMQX configuration, you can update the EMQX configuration by editing the `emqx.yaml` file and running `kubectl apply -f emqx.yaml` again. When EMQX operator find the `.metadata.annotations."apps.emqx.io/last-emqx-configuration"` field is different from the `.spec.config.data` field, it will put all of the `.spec.config.data` configures to the `/api/v5/config` endpoint of EMQX's core node for update the EMQX configuration. And then, it will update the `.metadata.annotations."apps.emqx.io/last-emqx-configuration"` field to the new configuration.

### Check EMQX cluster status

You can check the EMQX cluster status by running the following command:

```bash
$ pod_name=$(kubectl get pods -l apps.emqx.io/db-role=core,apps.emqx.io/instance=emqx-ee -o jsonpath='{.items[0].metadata.name}')
$ kubectl exec -it $pod_name -- emqx_ctl cluster status
Cluster status: #{running_nodes =>
                      ['emqx@10.244.0.12','emqx@10.244.0.13',
                       'emqx@10.244.0.14',
                       'emqx@emqx-ee-core-7494d76574-0.emqx-ee-headless.default.svc.cluster.local',
                       'emqx@emqx-ee-core-7494d76574-1.emqx-ee-headless.default.svc.cluster.local'],
                  stopped_nodes => []}
```

You should see the EMQX cluster status with the 2 core nodes and 3 replicant nodes. For EMQX's core node, it's must be a stateful node, so use the stable network ID to identify the node. More about the pod network ID, you can refer to the [Kubernetes documentation](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#stable-network-id). For EMQX's replicant node, it's a stateless node, so use the pod IP to identify the node, you can get the pod IP by running `kubectl get pods -o wide`.

### Connect to EMQX cluster

Running the following command to check the EMQX service:

```bash
$ kubectl get services
NAME                TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)                               AGE
emqx-ee-dashboard   ClusterIP   10.96.118.2   <none>        18083/TCP                             52m
emqx-ee-headless    ClusterIP   None          <none>        4370/TCP,5369/TCP                     53m
emqx-ee-listeners   ClusterIP   10.96.1.74    <none>        8883/TCP,1883/TCP,8083/TCP,8084/TCP   52m
```

You should see the EMQX service with the `emqx-ee-dashboard`, `emqx-ee-headless`, and `emqx-ee-listeners` services.

Ignore the `emqx-ee-headless` service, it's a headless service for the Kubernetes StatefulSet, you can refer to the [Kubernetes documentation](https://kubernetes.io/docs/concepts/services-networking/service/#headless-services) for more information.

The `emqx-ee-dashboard` service is used to access the EMQX dashboard, it always route to the core node, you can expose the EMQX service using the following command, and then access the EMQX dashboard by visiting [http://localhost:18083](http://localhost:18083) in your web browser, you can explore more features and configurations of EMQX by referring to the [EMQX documentation](https://docs.emqx.com/).

```bash
$ kubectl port-forward svc/emqx-ee-dashboard 18083:18083
```

The `emqx-ee-listeners` service is used to access the EMQX listeners, if EMQX cluster just have core nodes, the `emqx-ee-listeners` service will route to the core node, if EMQX cluster have replicant nodes, the `emqx-ee-listeners` service will route to the replicant node without the core node. You can expose the EMQX service using the following command

```bash
$ kubectl port-forward svc/emqx-ee-listeners 1883:1883
```

Now you can access the EMQX listeners by visiting `tcp://localhost:1883` in your MQTT client.

For example, you can use the MQTT X CLI to connect to the EMQX cluster, you can refer to the [MQTT X CLI documentation](https://mqttx.app/cli) for more information.

```bash
$ mqttx conn -h localhost -p 1883
```

## Clean up

To clean up the resources created by the EMQX Operator, you can run the following command:

```bash
$ kubectl delete emqx emqx-ee
```

To delete the EMQX Operator, you can run the following command:

```bash
$ helm uninstall emqx-operator -n emqx-operator-system
```

To delete the cert-manager, you can run the following command:

```bash
$ helm uninstall cert-manager -n cert-manager
```

To delete the Kubernetes cluster created by Kind, you can run the following command:

```bash
$ kind delete cluster
```
