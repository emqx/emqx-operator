## 部署 EMQX Operator

在本文中，我们将指导您完成高效设置 EMQX Operator 环境、安装 EMQX Operator，然后使用它部署 EMQX 所需的步骤。通过遵循本节中概述的指南，您将能够使用 EMQX Operator 有效地安装和管理 EMQX。

### 准备环境

EMQX Operator 部署前，请确认以下组件已经安装：

|   软件                   |   版本要求       |
|:-----------------------:|:---------------:|
|  [Helm](https://helm.sh)                 |  3 或更高          |
|  [cert-manager](https://cert-manager.io) |  1.1.6 或更高  |

### 安装 EMQX Operator

运行以下命令来安装 EMQX Operator

```bash
$ helm repo add emqx https://repos.emqx.io/charts
$ helm repo update
$ helm install emqx-operator emqx/emqx-operator --namespace emqx-operator-system --create-namespace
```

等待 EMQX Operator 就绪：

```bash
$ kubectl wait --for=condition=Ready pods -l "control-plane=controller-manager" -n emqx-operator-system

pod/emqx-operator-controller-manager-57bd7b8bd4-h2mcr condition met
```

现在你已经成功的安装的 EMQX Operator，你可以继续下一步了。在部署 EMQX 部分中，您将学习如何使用 EMQX Operator 来部署 EMQX。


### 升级 EMQX Operator

执行下面的命令可以升级 EMQX Operator，若想指定到升级版只需要增加 --version=x.x.x 参数即可

```bash
$ helm upgrade emqx-operator emqx/emqx-operator -n emqx-operator-system
```

> 不支持从版本1.x.x 升级到版本 2.x.x。

### 卸载 EMQX Operator

执行如下命令卸载 EMQX Operator

```bash
$ helm uninstall emqx-operator -n emqx-operator-system
```

## 部署 EMQX

### 部署 EMQX 5

1. 部署 EMQX。

   ```bash
   $ cat << "EOF" | kubectl apply -f -
   apiVersion: apps.emqx.io/v2alpha1
   kind: EMQX
   metadata:
      name: emqx
   spec:
      image: emqx:5.0
   EOF
   ```

2. 检查 EMQX 集群状态，请确保 STATUS 为 Running，这可能需要一些时间等待 EMQX 集群准备就绪。

   ```bash
   $ kubectl get emqx

   NAME   IMAGE      STATUS    AGE
   emqx   emqx:5.0   Running   2m55s
   ```

### 部署 EMQX 4

1. 部署 EMQX。

   ```bash
   $ cat << "EOF" | kubectl apply -f -
   apiVersion: apps.emqx.io/v1beta4
   kind: EmqxBroker
   metadata:
      name: emqx
   spec:
      template:
        spec:
          emqxContainer:
            image:
              repository: emqx
              version: 4.4
   EOF
   ```

2. 等待 EMQX 集群就绪。

   ```bash
   $ kubectl get emqxbrokers

   NAME   STATUS   AGE
   emqx   Running  8m33s
   ```

  请确保 `STATUS` 为 `Running`，这可能需要一些时间等待 EMQX 集群准备就绪。

### 部署 EMQX Enterprise 4

1. 部署 EMQX。

    ```bash
    $ cat << "EOF" | kubectl apply -f -
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
               version: 4.4.15
    EOF
    ```

2. 等待 EMQX 集群就绪。

   ```bash
   $ kubectl get emqxenterprises

   NAME      STATUS   AGE
   emqx-ee   Running  8m33s
   ```

  请确保 `STATUS` 为 `Running`，这可能需要一些时间等待 EMQX 集群准备就绪。
