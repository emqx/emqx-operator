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
kubectl apply -f "https://github.com/emqx/emqx-operator/releases/download/1.2.5/emqx-operator-controller.yaml"
```

### 通过 Helm 安装

1. 添加 EMQX Helm 仓库

```bash
helm repo add emqx https://repos.emqx.io/charts
helm repo update
```

2. 用 Helm 安装 EMQX Operator 控制器

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

## 部署 EMQX 企业版本

1. 部署 EMQX Enterprise Custom Resource

   ```
   cat << "EOF" | kubectl apply -f -
   apiVersion: apps.emqx.io/v1beta3
   kind: EmqxEnterprise
   metadata:
     name: emqx-ee
     labels:
       "foo": "bar"
   spec:
     emqxTemplate:
       image: emqx/emqx-ee:4.4.6
   EOF
   ```

2. 检查 EMQX 企业版状态

   ```bash
   $ kubectl get pods
   NAME              READY   STATUS    RESTARTS   AGE
   emqx-ee-0   2/2     Running   0          22m
   emqx-ee-1   2/2     Running   0          22m
   emqx-ee-2   2/2     Running   0          22m

   $ kubectl exec -it emqx-ee-0 -c emqx -- emqx_ctl status  
   Node 'emqx-ee@emqx-ee-0.emqx-ee-headless.default.svc.cluster.local' 4.4.5 is started  

   $ kubectl exec -it emqx-ee-0 -c emqx -- emqx_ctl cluster status  
   Cluster status: #{running_nodes =>
                      ['emqx-ee@emqx-ee-0.emqx-ee-headless.default.svc.cluster.local',
                       'emqx-ee@emqx-ee-1.emqx-ee-headless.default.svc.cluster.local',
                       'emqx-ee@emqx-ee-2.emqx-ee-headless.default.svc.cluster.local'],
                  stopped_nodes => []}
   ```

## 部署 EMQX 开源版

1. 部署 EMQX Broker Custom Resource

   ```
   cat << "EOF" | kubectl apply -f -
   apiVersion: apps.emqx.io/v1beta3
   kind: EmqxBroker
   metadata:
     name: emqx
     labels:
       "foo": "bar"
   spec:
     emqxTemplate:
       image: emqx/emqx:4.4.6
   EOF
   ```

2. 检查 EMQX 状态

   ```bash
   $ kubectl get pods
   NAME              READY   STATUS    RESTARTS   AGE
   emqx-0   2/2     Running   0          22m
   emqx-1   2/2     Running   0          22m
   emqx-2   2/2     Running   0          22m

   $ kubectl exec -it emqx-0 -c emqx -- emqx_ctl status
   Node 'emqx@emqx-0.emqx-headless.default.svc.cluster.local' 4.4.5 is started

   $ kubectl exec -it emqx-0 -c emqx -- emqx_ctl cluster status
   Cluster status: #{running_nodes =>
                      ['emqx@emqx-0.emqx-headless.default.svc.cluster.local',
                       'emqx@emqx-1.emqx-headless.default.svc.cluster.local',
                       'emqx@emqx-2.emqx-headless.default.svc.cluster.local'],
                  stopped_nodes => []} 
   ```
