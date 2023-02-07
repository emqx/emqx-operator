# 产品概览

该项目提供了一个 Operator，用于在 Kubernetes 上管理 EMQX 集群。

## 部署 EMQX Operator 

### 准备环境

EMQX Operator 部署前，请确认以下组件已经安装： 

|   软件                   |   版本要求       |
|:-----------------------:|:---------------:|
|  Kubernetes             |  >= 1.24        |          
|  Helm                   |  >= 3           |
|  cert-manager           |  >= 1.1.6       |

在 Kubernetes 1.24 及以上默认开启 `MixedProtocolLBService` 特性，其文档可以参考：[ MixedProtocolLBService ](https://kubernetes.io/zh-cn/docs/reference/command-line-tools-reference/feature-gates/#feature-gates-for-alpha-or-beta-features)。`MixedProtocolLBService` 特性允许在同一 `LoadBalancer` 类型的 Service 实例中使用不同的协议。因此如果用户在 Kubernetes 上部署 EMQX 集群，并且使用 `LoadBalancer` 类型的 Service，Service 里面同时存在 TCP 和 UDP 两种协议，请注意升级 Kubernetes 版本到 1.24及以上，否则会导致 Service 创建失败。

我们使用 [cert-manager](https://github.com/cert-manager/cert-manager)来给 webhook 服务提供证书。你可以通过 [cert manager 文档](https://cert-manager.io/docs/installation/)来安装。

### 安装 Helm 

1. 安装 Helm 

Helm 是 Kubernetes 包管理工具，执行下面的命令可以直接安装 Helm，更多 Helm 的安装方式可以参考：[安装 Helm](https://helm.sh/zh/docs/intro/install/)

```bash
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

2. 添加 Helm chart 仓库

```bash
helm repo add emqx https://repos.emqx.io/charts
helm repo update
```

### 安装 EMQX Operator 

```bash
helm install emqx-operator emqx/emqx-operator --namespace emqx-operator-system --create-namespace
```

检查 EMQX Operator 是否就绪

```bash
kubectl get pods -l "control-plane=controller-manager" -n emqx-operator-system
```

输出类似于：

```bash
NAME                                                READY   STATUS    RESTARTS   AGE
emqx-operator-controller-manager-68b866c8bf-kd4g6   1/1     Running   0          15s
```

### 升级 EMQX Operator 

执行下面的命令可以升级 EMQX Operator，若想指定到升级版只需要增加 --version=x.x.x 即可

```bash 
helm upgrade emqx-operator emqx/emqx-operator -n emqx-operator-system 
```

**备注：** 不支持 1.x.x 版本 EMQX Operator 升级到 2.x.x 版本 

### 卸载 EMQX Operator 

执行如下命令卸载 EMQX Operator

```bash
helm uninstall emqx-operator -n emqx-operator-system
```

## 部署 EMQX

### 部署 EMQX 5

1. 部署 EMQX 

```bash
cat << "EOF" | kubectl apply -f -
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
  name: emqx
spec:
  image: emqx/emqx:5.0.14
EOF
```

完整的例子请查看 [emqx-full.yaml](https://github.com/emqx/emqx-operator/blob/main/config/samples/emqx/v2alpha1/emqx-full.yaml)

每个字段的详细解释，请参考 [v2alpha1-reference](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/reference/v2alpha1-reference.md)

2. 检查 EMQX 集群是否就绪

```bash
kubectl get emqx emqx -o json | jq ".status.conditions"
```

输出类似于：

```bash 
[
  {
    "lastTransitionTime": "2023-02-07T06:46:39Z",
    "lastUpdateTime": "2023-02-07T06:46:39Z",
    "message": "Updating core nodes in cluster",
    "reason": "ClusterCoreUpdating",
    "status": "True",
    "type": "CoreNodesUpdating"
  },
  {
    "lastTransitionTime": "2023-02-07T06:46:38Z",
    "lastUpdateTime": "2023-02-07T06:46:38Z",
    "message": "Creating EMQX cluster",
    "reason": "ClusterCreating",
    "status": "True",
    "type": "Creating"
  }
]
```

### 部署 EMQX 4

1. 部署 EMQX 

```bash
cat << "EOF" | kubectl apply -f -
apiVersion: apps.emqx.io/v1beta4
kind: EmqxBroker
metadata:
  name: emqx
spec:
  template:
    spec:
      emqxContainer:
        image:
          repository: emqx/emqx-ee
          version: 4.4.14
EOF
```

完整的例子请查看 [emqxbroker-full.yaml](https://github.com/emqx/emqx-operator/blob/main/config/samples/emqx/v1beta4/emqxenterprise-full.yaml)

每个字段的详细解释，请参考 [v1beta3-reference](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/reference/v1beta4-reference.md)

2. 检查 EMQX 集群是否就绪

```bash
kubectl get emqxbroker emqx -o json | jq ".status.conditions"
```

输出类似于：

```bash 
[
  {
    "lastTransitionTime": "2023-02-07T02:42:05Z",
    "lastUpdateTime": "2023-02-07T06:41:05Z",
    "message": "All resources are ready",
    "reason": "ClusterReady",
    "status": "True",
    "type": "Running"
  }
]
```

### 部署 EMQX Enterprise 4

1. 部署 EMQX 

```bash
cat << "EOF" | kubectl apply -f -
apiVersion: apps.emqx.io/v1beta4
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  template:
    spec:
      emqxContainer:
        image:
          repository: emqx/emqx-ee
          version: 4.4.14
EOF
```

完整的例子请查看 [emqxenterprise-full.yaml](https://github.com/emqx/emqx-operator/blob/main/config/samples/emqx/v1beta4/emqxenterprise-full.yaml)

每个字段的详细解释，请参考 [v1beta3-reference](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/reference/v1beta4-reference.md)

2. 检查 EMQX 集群是否就绪

```bash 
kubectl get emqxenterprise emqx-ee -o json | jq ".status.conditions"
```

输出类似于：

```bash 
[
  {
    "lastTransitionTime": "2023-02-07T06:42:13Z",
    "lastUpdateTime": "2023-02-07T06:45:12Z",
    "message": "All resources are ready",
    "reason": "ClusterReady",
    "status": "True",
    "type": "Running"
  }
]
```
