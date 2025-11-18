# Hello EMQX Operator

In this guide, we will walk you through deploying Kubernetes locally using Kind, installing the EMQX Operator, and using it to deploy EMQX.

Kind (Kubernetes in docker) is a tool for running local Kubernetes clusters using Docker container nodes. Kind is primarily designed for testing Kubernetes itself, but it can also be used for local development or CI.

::: warning
Do not use Kind in production environments.
:::

## Install tools

### Docker

On Linux: [Docker installation guide](https://docs.docker.com/desktop/install/linux/)

On MacOS: [Orbstack](https://orbstack.dev/)

### Kind

Install Kind by following the [Kind installation guide](https://kind.sigs.k8s.io/docs/user/quick-start/#installing-from-release-binaries)

### kubectl

Install kubectl by following the [kubectl installation guide](https://kubernetes.io/docs/tasks/tools/#kubectl)

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

Now you have a Kubernetes cluster running locally, check out the [Kubernetes documentation](https://kubernetes.io/docs/home/) for more information.

### Install EMQX Operator

Run the following command to install the EMQX Operator:

```bash
$ kubectl apply --server-side=true \
  -f https://github.com/emqx/emqx-operator/releases/download/v2.3.0/install.yaml
```

Wait for the EMQX Operator to be installed, you can check the status by running:

```bash
$ kubectl wait --for=condition=Ready pods -l "control-plane=controller-manager" -n emqx-operator-system
```

Now you have successfully installed the EMQX Operator, continue to the next step.

## Deploy EMQX

EMQX operator provides a custom resource definition (CRD) called `EMQX`, which allows you to define and manage EMQX clusters in Kubernetes.

Create a file named `emqx.yaml` with the following content:

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
  coreTemplate:
    spec:
      replicas: 2
  replicantTemplate:
    spec:
      replicas: 3
```

In the `emqx.yaml` file, you define the `image` field to specify the EMQX image to use. And also define the `config` field to specify the EMQX configuration, in this example, set the `license.key` to license content. You also define the `coreTemplate` and `replicantTemplate` to specify the number of replicas for the core and replicant nodes.

For more configuration options, you can refer to the [EMQX v2 CRD Reference](https://docs.emqx.com/en/emqx-operator/latest/reference/v2-reference.html).

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

### Under the hood

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

## Manage EMQX

### Check EMQX configuration

You can check the _currently specified_ EMQX configuration by running the following command:

```bash
$ kubectl get emqx emqx-ee -o json | jq -r '.spec.config.data'
```

This will output the EMQX configuration data that you defined in the `emqx.yaml` file. EMQX Operator will keep track of the _last applied_ EMQX configuration in the `.metadata.annotations` field, so you can check the _last applied_ EMQX configuration by running the following command:

```bash
$ kubectl get emqx emqx-ee -o json | jq -r '.metadata.annotations."apps.emqx.io/last-emqx-configuration"'
```

Most of the time, this output should be the same as the output of the previous command.

Keep in mind that the changes made through the EMQX Dashboard will not be reflected anywhere in the EMQX CR fields. To consult the _effective_ EMQX cluster configuration, you can use the following command:

```bash
$ kubectl exec -it emqx-ee-core-0 -c emqx -- emqx ctl conf show
```

The output should be easy to feed back into the `.spec.config.data` field of the EMQX CR to preserve the effective configuration.

### Update EMQX configuration

EMQX Operator reflects the changes made to `.spec.config.data` field to the running EMQX cluster. Each time this field is changed, EMQX Operator will:
1. Request the EMQX Configuration API to update the EMQX cluster configuration in runtime.
2. Update the `.metadata.annotations."apps.emqx.io/last-emqx-configuration"` field to the new configuration.

### Check EMQX cluster status

You can check the EMQX cluster status by running the following command:

```bash
$ pod_name=$(kubectl get pods -l apps.emqx.io/db-role=core,apps.emqx.io/instance=emqx-ee -o jsonpath='{.items[0].metadata.name}')
$ kubectl exec -it $pod_name -- emqx ctl cluster status
Cluster status: #{running_nodes =>
                      ['emqx@10.244.0.12','emqx@10.244.0.13',
                       'emqx@10.244.0.14',
                       'emqx@emqx-ee-core-7494d76574-0.emqx-ee-headless.default.svc.cluster.local',
                       'emqx@emqx-ee-core-7494d76574-1.emqx-ee-headless.default.svc.cluster.local'],
                  stopped_nodes => []}
```

As specified in the CR, EMQX cluster consists of 2 core nodes and 3 replicant nodes. Core nodes use stable names to identify the node, derived from the pod name and respective [headless service](https://kubernetes.io/docs/concepts/services-networking/service/#headless-services). Replicant nodes, on the other hand, use the pod IP address to identify the node.

Refer to the [Kubernetes documentation](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#stable-network-id) to learn more about the stable network ID.

### Connect to EMQX cluster

Running the following command to see the managed EMQX services:

```bash
$ kubectl get services
NAME                TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)                               AGE
emqx-ee-dashboard   ClusterIP   10.96.118.2   <none>        18083/TCP                             52m
emqx-ee-headless    ClusterIP   None          <none>        4370/TCP,5369/TCP                     53m
emqx-ee-listeners   ClusterIP   10.96.1.74    <none>        8883/TCP,1883/TCP,8083/TCP,8084/TCP   52m
```

Service `emqx-ee-dashboard` is used to access the EMQX dashboard, it always routes to the core nodes. You can temporarily expose the EMQX service, and then access the EMQX dashboard by visiting [http://localhost:18083](http://localhost:18083) in your web browser.

```bash
$ kubectl port-forward svc/emqx-ee-dashboard 18083:18083
```

Service `emqx-ee-listeners` is used to access the EMQX listeners. If the EMQX cluster only has core nodes, `emqx-ee-listeners` routes to the core nodes. If the EMQX cluster has replicant nodes, `emqx-ee-listeners` routes to the replicant nodes only. You can temporarily expose the EMQX TCP listener:

```bash
$ kubectl port-forward svc/emqx-ee-listeners 1883:1883
```

Now you can connect to the EMQX TCP listener using your MQTT client. For example, you can use the [MQTTX CLI](https://mqttx.app/cli):

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

To delete the Kubernetes cluster created by Kind, you can run the following command:

```bash
$ kind delete cluster
```
