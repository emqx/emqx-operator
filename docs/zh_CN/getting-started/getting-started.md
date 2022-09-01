# 产品概览

**注意**: EMQX Operator 控制器需要 Kubernetes v1.20.0 或者以上。

## 部署Operator控制器

### 准备

我们使用 [cert-manager](https://github.com/cert-manager/cert-manager)来给 webhook 服务提供证书。你可以通过 [cert manager 文档](https://cert-manager.io/docs/installation/)来安装。

### 安装

1. 通过 Helm 安装

```bash
helm repo add emqx https://repos.emqx.io/charts
helm repo update
helm install emqx-operator emqx/emqx-operator --set installCRDs=true --namespace emqx-operator-system --create-namespace
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
        image: emqx/emqx:5.0.6
    EOF
    ```

    完整的例子请查看 [`emqx-full.yaml`](https://github.com/emqx/emqx-operator/blob/2.0.0/config/samples/emqx/v2alpha1/emqx-full.yaml)

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
          image: emqx/emqx:4.4.8
    EOF
    ```

    完整的例子请查看 [`emqxbroker-full.yaml`](https://github.com/emqx/emqx-operator/blob/2.0.0/config/samples/emqx/v1beta3/emqxbroker-full.yaml).

2. 检查 EMQX 自定义资源状态

    ```
    $ kubectl get pods
    $ kubectl get emqx emqx -o json | jq ".status.emqxNodes"
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
          image: emqx/emqx-ee:4.4.8
    EOF
    ```


    完整的例子请查看 [`emqxenterprise-full.yaml`](https://github.com/emqx/emqx-operator/blob/2.0.0/config/samples/emqx/v1beta3/emqxenterprise-full.yaml).

2. 检查 EMQX 自定义资源状态

    ```
    $ kubectl get pods
    $ kubectl get emqx emqx -o json | jq ".status.emqxNodes"
    ```