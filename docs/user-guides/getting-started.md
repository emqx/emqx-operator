**Note**: EMQ X Operator requires Kubernetes v1.20.0 and up.

## Quickstart

### Deployment

* Deploy the operator in the Kubernetes cluster
* Run the project out of the Kubernetes cluster

>Example: Run the Operator in the ACK service in ALiCloud, the LB and Persistence Volume should prepared before deploying.

* Register the CustomResourceDefinitions into the Kubernetes Resources.

```bash
kubectl create -f config/samples/operator/apps.emqx.io_emqxes.yaml
```

1. Build the container image and push to the image repo

```bash
IMG=emqx/emqx-operator:0.1.0 make docker-build
IMG=emqx/emqx-operator:0.1.0 make docker-push
```

**The `IMG` is related to the `spec.template.spec.containers[0].image` in `config/samples/operator/operator_deployment.yaml`**

* pv.yaml:

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv-emqx-log-dir-emqx-0
  labels:
    cluster: emqx
spec:
  accessModes:
    - ReadWriteMany
  capacity:
    storage: 1Gi
  volumeMode: Filesystem
  csi:
    driver: nasplugin.csi.alibabacloud.com
    volumeAttributes:
      path: /opt/emqx-log
      server: # to be completed 
      vers: "3"
    volumeHandle: pv-emqx-log-dir-emqx
  persistentVolumeReclaimPolicy: Retain
  storageClassName: nas
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv-emqx-data-dir-emqx-0
  labels:
    cluster: emqx
spec:
  accessModes:
    - ReadWriteMany
  capacity:
    storage: 1Gi
  volumeMode: Filesystem
  csi:
    driver: nasplugin.csi.alibabacloud.com
    volumeAttributes:
      path: /opt/emqx-data
      server: # to be completed
      vers: "3"
    volumeHandle: pv-emqx-data-dir-emqx
  persistentVolumeReclaimPolicy: Retain
  storageClassName: nas
```

* lb.yaml:

```yaml
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
  type: LoadBalancer
```

* Enable RBAC rules for EMQ X Operator pods

```bash
kubectl create -f config/samples/operator/operator_namespace.yaml
kubectl create -f config/samples/operator/operator_service_account.yaml
kubectl create -f config/samples/operator/operator_role.yaml
kubectl create -f config/samples/operator/operator_role_binding.yaml
kubectl create -f config/samples/operator_deployment.yaml
```

* Enable RBAC rule for EMQ X pods

```bash
kubectl create -f config/samples/emqx/emqx_serviceaccount.yaml
kubectl create -f config/samples/emqx/emqx_role.yaml
kubectl create -f config/samples/emqx/emqx_role_binding.yaml
```

* emqx_cr.yaml

```yaml
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
        storageClassName: nas
        resources:
          requests:
            storage: 1Gi
        accessModes:
        - ReadWriteMany
  cluster:
    name: emqx
    k8s:   
      apiserver: # k8s apiserver address
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
  
* Make sure the EMQ X pods running as expected

```bash
kubectl get pods               
NAME              READY   STATUS    RESTARTS   AGE
emqx-0   1/1     Running   0          22m
$ kubectl logs -f emqx-0
cluster.autoclean = "5m0s"
cluster.autoheal = "on"
cluster.discovery = "k8s"
cluster.k8s.address_type = "dns"
cluster.k8s.apiserver = "https://xxxxxxx"
cluster.k8s.app_name = "emqx"
cluster.k8s.namespace = "default"
cluster.k8s.service_name = "emqx"
cluster.k8s.suffix = "pod.cluster.local"
cluster.name = "emqx"
cluster.proto_dist = "inet_tcp"
listener.ssl.external.acceptors = "32"
listener.ssl.external.max_connections = "102400"
listener.tcp.external.acceptors = "64"
listener.tcp.external.max_connections = "1024000"
listener.ws.external.acceptors = "16"
listener.ws.external.max_connections = "102400"
listener.wss.external.acceptors = "16"
listener.wss.external.max_connections = "102400"
log.to = "console"
node.max_ets_tables = "2097152"
node.max_ports = "1048576"
node.name = "emqx@172-19-96-138.default.pod.cluster.local"
node.process_limit = "2097152"
rpc.port_discovery = "manual"
Starting emqx on node emqx@172-19-96-138.default.pod.cluster.local
Start http:management listener on 8081 successfully.
Start http:dashboard listener on 18083 successfully.
Start mqtt:tcp:internal listener on 127.0.0.1:11883 successfully.
Start mqtt:tcp:external listener on 0.0.0.0:1883 successfully.
Start mqtt:ws:external listener on 0.0.0.0:8083 successfully.
Start mqtt:ssl:external listener on 0.0.0.0:8883 successfully.
Start mqtt:wss:external listener on 0.0.0.0:8084 successfully.
EMQ X Broker 4.3.8 is running now
```

2. Running project out of the cluster.

> Prerequirements: LB, Persistence Volume, CRD

* Make sure kube config is configured properly
* Run `main.go`
* Create RBAC objects from manifest file

```bash
kubectl create -f config/samples/emqx/broker_serviceaccount.yaml
kubectl create -f config/samples/emqx/broker_role.yaml
kubectl create -f config/samples/emqx/broker_role_binding.yaml
kubectl create -f config/samples/emqx/custom_v1alpha1_broker.yaml
```

* Verify the  `EMQ X pods` running successfully

### Scaling the cluster

[cluster-expansion](docs/cluster-expansion.md)
