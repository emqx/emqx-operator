# 产品概览

**注意**: EMQX Operator 控制器需要 Kubernetes v1.20.0 或者以上。

## 背景

此教程采用 minikube v1.20.0 安装部署

## 部署Operator控制器

### 准备

我们使用 [cert manager](https://github.com/cert-manager/cert-manager)来给 webhook 服务提供证书。你可以通过 [cert manager 文档](https://cert-manager.io/docs/installation/)来安装。

### 默认静态安装

安装默认静态配置文件

```bash
$ kubectl apply -f https://raw.githubusercontent.com/emqx/emqx-operator/1.1.2/config/samples/operator/controller.yaml
```

### 通过 Helm 安装

1. 添加 EMQX Helm 仓库

```bash
$ helm repo add emqx https://repos.emqx.io/charts
$ helm repo update
```

1. 用 Helm 安装 EMQX Operator 控制器

```bash
$ helm install emqx-operator emqx/emqx-operator \
   --set installCRDs=true \
   --namespace emqx-operator-system \
   --create-namespace
```

### 检查 EMQX Operator 控制器状态

```bash
$ kubectl get pods -l "control-plane=controller-manager" -n emqx-operator-system
NAME                                                READY   STATUS    RESTARTS   AGE
emqx-operator-controller-manager-68b866c8bf-kd4g6   1/1     Running   0          15s
```

## 部署 EMQX Broker

1. 创建 EMQX Custom Resource

```bash
$ cat https://raw.githubusercontent.com/emqx/emqx-operator/1.1.2/config/samples/emqx/v1beta2/emqx.yaml
```

```yaml

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
    - ReadWriteOnce
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

Note:

> [Details for *cluster* config](https://docs.emqx.io/en/broker/v4.3/configuration/configuration.html)
[Details for *env* config](https://docs.emqx.io/en/broker/v4.3/configuration/configuration.html)
>

1. 部署 EMQX Custom Resource 并且检查 EMQX 状态

```bash
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

Note:

> EMQX Operator 给 EMQX 集群提供默认的监听器。默认的服务类型是 ClusterIP，用户也可以选择 LoadBalancer 和 NodePort。
`ws`、`wss`、`mqtt`、`mqtts`、`dashboard`、 `api` 的端口需要提前配置，否则在集群允许时无法修改。
>