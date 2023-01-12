# 产品概览

该项目提供了一个 Operator，用于在 Kubernetes 上管理 EMQX 集群。

**注意**: EMQX Operator 控制器需要 Kubernetes v1.20.11 或者以上。

## 部署 Operator 控制器

### 准备

我们使用 [cert-manager](https://github.com/cert-manager/cert-manager)来给 webhook 服务提供证书。你可以通过 [cert manager 文档](https://cert-manager.io/docs/installation/)来安装。

### 安装

1. 通过 Helm 安装

```bash
helm repo add emqx https://repos.emqx.io/charts
helm repo update
helm install emqx-operator emqx/emqx-operator --namespace emqx-operator-system --create-namespace
```

2. 等待 EMQX Operator 控制器就绪

```bash
$ kubectl get pods -l "control-plane=controller-manager" -n emqx-operator-system
NAME                                                READY   STATUS    RESTARTS   AGE
emqx-operator-controller-manager-68b866c8bf-kd4g6   1/1     Running   0          15s
```

## 部署 EMQX

### 部署 EMQX 5

1. 部署 EMQX 自定义资源

    ```bash
    cat << "EOF" | kubectl apply -f -
      apiVersion: apps.emqx.io/v2alpha1
      kind: EMQX
      metadata:
        name: emqx
      spec:
        image: emqx/emqx:5.0.9
    EOF
    ```

    完整的例子请查看 [`emqx-full.yaml`](https://github.com/emqx/emqx-operator/blob/main/config/samples/emqx/v2alpha1/emqx-full.yaml)

    每个字段的详细解释，请参考 [v2alpha1-reference](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/reference/v2alpha1-reference.md)

2. 检查 EMQX 自定义资源状态

    ```
    $ kubectl get pods
    $ kubectl get emqx emqx -o json | jq ".status.emqxNodes"
    ```

### 部署 EMQX 4

1. 部署 EMQX 自定义资源

    ```bash
    cat << "EOF" | kubectl apply -f -
      apiVersion: apps.emqx.io/v1beta3
      kind: EmqxBroker
      metadata:
        name: emqx
      spec:
        emqxTemplate:
          image: emqx/emqx:4.4.9
    EOF
    ```

    完整的例子请查看 [`emqxbroker-full.yaml`](https://github.com/emqx/emqx-operator/blob/main/config/samples/emqx/v1beta3/emqxbroker-full.yaml)

    每个字段的详细解释，请参考 [v1beta3-reference](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/reference/v1beta3-reference.md)

2. 检查 EMQX 自定义资源状态

    ```
    $ kubectl get pods
    $ kubectl get emqxbroker emqx -o json | jq ".status.emqxNodes"
    ```


### 部署 EMQX Enterprise 4

1. 部署 EMQX 自定义资源

    ```bash
    cat << "EOF" | kubectl apply -f -
      apiVersion: apps.emqx.io/v1beta3
      kind: EmqxEnterprise
      metadata:
        name: emqx-ee
      spec:
        emqxTemplate:
          image: emqx/emqx-ee:4.4.9
    EOF
    ```


    完整的例子请查看 [`emqxenterprise-full.yaml`](https://github.com/emqx/emqx-operator/blob/main/config/samples/emqx/v1beta3/emqxenterprise-full.yaml)

    每个字段的详细解释，请参考 [v1beta3-reference](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/reference/v1beta3-reference.md)

2. 检查 EMQX 自定义资源状态

    ```
    $ kubectl get pods
    $ kubectl get emqxenterprise emqx-ee -o json | jq ".status.emqxNodes"
    ```

## 备注

1. 在 Kubernetes 1.24 及以上默认开启 `MixedProtocolLBService` 特性，其文档可以参考：[ MixedProtocolLBService ](https://kubernetes.io/zh-cn/docs/reference/command-line-tools-reference/feature-gates/#feature-gates-for-alpha-or-beta-features)。`MixedProtocolLBService` 特性允许在同一 `LoadBalancer` 类型的 Service 实例中使用不同的协议。因此如果用户在 Kubernetes 上部署 EMQX 集群，并且使用 `LoadBalancer` 类型的 Service，Service 里面同时存在 TCP 和 UDP 两种协议，请注意升级 Kubernetes 版本到 1.24及以上，否则会导致 Service 创建失败。
