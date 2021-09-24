# emqx-operator

EMQ X Broker Operator

## 快速指导

### 部署

* 有两种方式运行 `operator`:
  * 在 `kubernetes` 集群中以`Deployment`部署`operator`
  * 在集群外作为`Go`工程项目运行

> 示例为在阿里云`ACK`容器服务运行，`lb`及`pv`做为依赖资源，需提前准备

* *在运行 `operator` 之前，首先得将 `CRD` 注册到 `kubernetes apiserver`中*

```bash
kubectl create -f config/samples/apps.emqx.io_emqxes.yaml
```

1. 编译及制作镜像，并推送到镜像仓库部署 `operator deployment`

```bash
IMG=emqx/emqx-operator:0.1.0 make docker-build
IMG=emqx/emqx-operator:0.1.0 make docker-push
```

*此处的 `IMG` 镜像名称对应 `config/samples/operator/operator_deployment.yaml` 中 `spec.template.spec.containers[0].image`字段*

* 创建 `pv` 的 `yaml` 示例文件

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
      server: # 持久化 server 配置 
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
      server: # 持久化 server 配置
      vers: "3"
    volumeHandle: pv-emqx-data-dir-emqx
  persistentVolumeReclaimPolicy: Retain
  storageClassName: nas
```

* 创建 `lb` 的示例`yaml`文件：

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

* 创建 `operator manager controller` 相关组件服务

```bash
kubectl create -f config/samples/operator/operator_namespace.yaml
kubectl create -f config/samples/operator/operator_service_account.yaml
kubectl create -f config/samples/operator/operator_role.yaml
kubectl create -f config/samples/operator/operator_role_binding.yaml
kubectl create -f config/samples/operator_deployment.yaml
```

* 创建 `cr` 关联的 `RBAC` 及 `cr`

```bash
kubectl create -f config/samples/emqx/emqx_serviceaccount.yaml
kubectl create -f config/samples/emqx/emqx_role.yaml
kubectl create -f config/samples/emqx/emqx_role_binding.yaml
```

* `emqx_cr.yaml` 示例文件

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

> * `cluster` 配置，明细请参考[`cluster`参数配置](https://docs.emqx.cn/enterprise/v4.3/configuration/configuration.html#cluster)
> * `env` 配置，明细清参考[参数配置](https://docs.emqx.cn/enterprise/v4.3/configuration/configuration.html)

* 确认 `emqx pod` 正常运行

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

2. 集群外运行本地工程项目

> `lb` 及持久化前置依赖资源同上，需提前规划准备部署
> 注册`crd`资源，同上

* 将 `kubernetes` 集群的配置文件 `.kube/config` 复制到本地 `$HOME/.kube/config`
* 运行 `main.go` 文件
* 创建 `cr` 关联的 `RBAC` 及 `cr`

```bash
kubectl create -f config/samples/emqx/broker_serviceaccount.yaml
kubectl create -f config/samples/emqx/broker_role.yaml
kubectl create -f config/samples/emqx/broker_role_binding.yaml
kubectl create -f config/samples/emqx/custom_v1alpha1_broker.yaml
```

* 确认 `emqx pod` 成功运行

### 集群扩容

![集群扩容](docs/cluster-scal-cn.md)

## Q&A

1. 在本地执行 `make docker build` 时出现如下报错：

```bash
Unexpected error: msg: "failed to start the controlplane. retried 5 times: fork/exec /usr/local/kubebuilder/bin/etcd: no such file or directory"
```

[解决方法参考](https://github.com/kubernetes-sigs/kubebuilder/issues/1599)
执行如下脚本

```bash
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/')
curl -fsL "https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-1.16.4-${OS}-${ARCH}.tar.gz" -o kubebuilder-tools
tar -zvxf kubebuilder-tools
sudo mv kubebuilder/ /usr/local/kubebuilder
```
