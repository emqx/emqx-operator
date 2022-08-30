# 产品概览

**注意**: EMQX Operator 控制器需要 Kubernetes v1.20.0 或者以上。

## 部署Operator控制器

### 准备

我们使用 [cert-manager](https://github.com/cert-manager/cert-manager)来给 webhook 服务提供证书。你可以通过 [cert manager 文档](https://cert-manager.io/docs/installation/)来安装。

### 安装
EMQX Operator 提供helm 以及 静态yaml安装，我们推荐使用 helm 来安装 EMQX Operator

#### 通过 Helm 安装

```bash
helm repo add emqx https://repos.emqx.io/charts
helm repo update
helm install emqx-operator emqx/emqx-operator --set installCRDs=true --namespace emqx-operator-system --create-namespace
```

#### 静态yaml安装

安装默认静态配置文件(如果已经通过helm安装，则跳过该步骤)

```bash
kubectl apply -f "https://github.com/emqx/emqx-operator/releases/download/2.0.0-alpha.1/emqx-operator-controller.yaml"
```



### 检查 EMQX Operator 控制器状态

```bash
$ kubectl get pods -l "control-plane=controller-manager" -n emqx-operator-system
NAME                                                READY   STATUS    RESTARTS   AGE
emqx-operator-controller-manager-68b866c8bf-kd4g6   1/1     Running   0          15s
```

## 部署 EMQX

1.  部署 EMQX 自定义资源

   ```bash
   cat << "EOF" | kubectl apply -f -
     apiVersion: apps.emqx.io/v2alpha1
     kind: EMQX
     metadata:
       name: emqx
     spec:
       emqxTemplate:
         image: emqx/emqx:5.0.6
   EOF
   ```

2. 检查 EMQX 状态

   ```bash
   $ kubectl get pods
   NAME                              READY   STATUS    RESTARTS        AGE
   emqx-core-0                       1/1     Running   0               75s
   emqx-core-1                       1/1     Running   0               75s
   emqx-core-2                       1/1     Running   0               75s
   emqx-replicant-6c8b4fccfb-bkk4s   1/1     Running   0               75s
   emqx-replicant-6c8b4fccfb-kmg9j   1/1     Running   0               75s
   emqx-replicant-6c8b4fccfb-zc929   1/1     Running   0               75s

   $ kubectl get emqx emqx -o json | jq ".status.emqxNodes"
   [
     {
       "node": "emqx@172.17.0.11",
       "node_status": "running",
       "otp_release": "24.2.1-1/12.2.1",
       "role": "replicant",
       "version": "5.0.6"
     },
     {
       "node": "emqx@172.17.0.12",
       "node_status": "running",
       "otp_release": "24.2.1-1/12.2.1",
       "role": "replicant",
       "version": "5.0.6"
     },
     {
       "node": "emqx@172.17.0.13",
       "node_status": "running",
       "otp_release": "24.2.1-1/12.2.1",
       "role": "replicant",
       "version": "5.0.6"
     },
     {
       "node": "emqx@emqx-core-0.emqx-headless.default.svc.cluster.local",
       "node_status": "running",
       "otp_release": "24.2.1-1/12.2.1",
       "role": "core",
       "version": "5.0.6"
     },
     {
       "node": "emqx@emqx-core-1.emqx-headless.default.svc.cluster.local",
       "node_status": "running",
       "otp_release": "24.2.1-1/12.2.1",
       "role": "core",
       "version": "5.0.6"
     },
     {
       "node": "emqx@emqx-core-2.emqx-headless.default.svc.cluster.local",
       "node_status": "running",
       "otp_release": "24.2.1-1/12.2.1",
       "role": "core",
       "version": "5.0.6"
     }
   ]
   ```
