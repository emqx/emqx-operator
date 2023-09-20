# 产品概览

**注意**: EMQX Operator 控制器需要 Kubernetes v1.20.0 或者以上。

## 部署Operator控制器

### 准备

我们使用 [cert manager](https://github.com/cert-manager/cert-manager)来给 webhook 服务提供证书。你可以通过 [cert manager 文档](https://cert-manager.io/docs/installation/)来安装。

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
kubectl apply -f "https://github.com/emqx/emqx-operator/releases/download/1.2.7-ecp.5/emqx-operator-controller.yaml"
```

### 检查 EMQX Operator 控制器状态

```bash
$ kubectl get pods -l "control-plane=controller-manager" -n emqx-operator-system
NAME                                                READY   STATUS    RESTARTS   AGE
emqx-operator-controller-manager-68b866c8bf-kd4g6   1/1     Running   0          15s
```


## 部署 EMQX 4

1. 部署 EMQX 自定义资源

   ```bash
   cat << "EOF" | kubectl apply -f -
   apiVersion: apps.emqx.io/v1beta3
   kind: EmqxBroker
   metadata:
     name: emqx
     labels:
       "foo": "bar"
   spec:
     emqxTemplate:
       image: emqx/emqx:4.4.8
   EOF
   ```
    完整的例子请查看 [emqxbroker-full.yaml.](https://github.com/emqx/emqx-operator/blob/2.0.0/config/samples/emqx/v1beta3/emqxbroker-full.yaml)

2. 检查 EMQX 自定义资源状态

   ```bash
    $ kubectl get pods
    $ kubectl get emqx emqx -o json | jq ".status.emqxNodes"
   ```


## 部署 EMQX Enterprise 4

1. 部署 EMQX 自定义资源

   ```bash
   cat << "EOF" | kubectl apply -f -
   apiVersion: apps.emqx.io/v1beta3
   kind: EmqxEnterprise
   metadata:
     name: emqx-ee
     labels:
       "foo": "bar"
   spec:
     emqxTemplate:
       image: emqx/emqx-ee:4.4.8
   EOF
   ```
   完整的例子请查看 [emqxenterprise-full.yaml.](https://github.com/emqx/emqx-operator/blob/2.0.0/config/samples/emqx/v1beta3/emqxenterprise-full.yaml)

2. 检查 EMQX 自定义资源状态

   ```bash
    $ kubectl get pods
    $ kubectl get emqx emqx -o json | jq ".status.emqxNodes"
   ```
