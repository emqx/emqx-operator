# License 配置 (EMQX 企业版)

## 任务目标

- 配置 EMQX 企业版 License。
- 更新 EMQX 企业版 License。

## 配置 License

EMQX 企业版 License 可以在 EMQ 官网免费申请：[申请 EMQX 企业版 License](https://www.emqx.com/zh/apply-licenses/emqx)。

下面是 EMQX Custom Resource 的相关配置，你可以根据希望部署的 EMQX 的版本来选择对应的 APIVersion，具体的兼容性关系，请参考 [EMQX Operator 兼容性](../index.md):

## 配置 EMQX 集群

:::: tabs type:card
::: tab apps.emqx.io/v2beta1

`apps.emqx.io/v2beta1 EMQX` 支持通过 `.spec.config.data` 配置 EMQX 集群 License，EMQX 配置可以参考文档：[配置手册](https://www.emqx.io/docs/zh/v5.1/configuration/configuration-manual.html#%E8%8A%82%E7%82%B9%E8%AE%BE%E7%BD%AE)。

> 在创建 EMQX 集群之后，如果需要更新 License，请通过 EMQX Dashboard 进行更新。

+ 将下面的内容保存成 YAML 文件，并通过 `kubectl apply` 命令部署它

  ```yaml
  apiVersion: apps.emqx.io/v2beta1
  kind: EMQX
  metadata:
    name: emqx-ee
  spec:
    config:
      data: |
        license {
          key = "${your_license_key}"
        }
    image: emqx/emqx-enterprise:5.6
    listenersServiceTemplate:
      spec:
        type: LoadBalancer
    dashboardServiceTemplate:
      spec:
        type: LoadBalancer
  ```

  > `config.data` 字段里面的 `license.key` 表示 Licesne 内容，此例中 License 内容被省略，请用户自行填充。

+ 等待 EMQX 集群就绪，可以通过 `kubectl get` 命令查看 EMQX 集群的状态，请确保 `STATUS` 为 `Running`，这个可能需要一些时间

  ```bash
  $ kubectl get emqx emqx-ee
  NAME   IMAGE                        STATUS    AGE
  emqx   emqx/emqx-enterprise:5.1.0   Running   10m
  ```

+ 获取 EMQX 集群的 Dashboard External IP，访问 EMQX 控制台

  EMQX Operator 会创建两个 EMQX Service 资源，一个是 emqx-dashboard，一个是 emqx-listeners，分别对应 EMQX 控制台和 EMQX 监听端口。

  ```bash
  $ kubectl get svc emqx-ee-dashboard -o json | jq '.status.loadBalancer.ingress[0].ip'

  192.168.1.200
  ```

  通过浏览器访问 `http://192.168.1.200:18083` ，使用默认的用户名和密码 `admin/public` 登录 EMQX 控制台。

:::
::: tab apps.emqx.io/v1beta4

+ 基于 License 文件创建 Secret

  Secret 是一种包含少量敏感信息例如密码、令牌或密钥的对象。关于 Secret 更加详尽的文档可以参考：[Secret](https://kubernetes.io/zh-cn/docs/concepts/configuration/secret/)。EMQX Operator 支持使用 Secret 挂载 License 信息，因此在创建 EMQX 集群之前我们需要基于 License 创建好 Secret。

  ```bash
  $ kubectl create secret generic ${your_license_name} --from-file=emqx.lic=${/path/to/license/file}
  ```

  > `${your_license_name}` 表示创建的 Secret 名称。

  > `${/path/to/license/file}` 表示 EMQX 企业版 License 文件路径，可以是绝对路径，也可以是相对路径。更多使用 kubectl 创建 Secret 的细节可以参考文档：[使用 kubectl 创建 secret](https://kubernetes.io/zh-cn/docs/tasks/configmap-secret/managing-secret-using-kubectl/)。

+ 将下面的内容保存成 YAML 文件，并通过 `kubectl apply` 命令部署它

  `apps.emqx.io/v1beta4 EmqxEnterprise` 支持通过 `.spec.license` 字段来配置 EMQX 企业版 License，更多信息请查看：[license](../reference/v1beta4-reference.md#emqxlicense)。

  ```yaml
  apiVersion: apps.emqx.io/v1beta4
  kind: EmqxEnterprise
  metadata:
    name: emqx-ee
  spec:
    license:
      secretName: ${your_license_name}
    template:
      spec:
        emqxContainer:
          image:
            repository: emqx/emqx-ee
            version: 4.4.14
    serviceTemplate:
      spec:
        type: LoadBalancer
  ```
  > `secretName` 表示上一步中创建的 Secret 名称。

+ 等待 EMQX 集群就绪，可以通过 `kubectl get` 命令查看 EMQX 集群的状态，请确保 `STATUS` 为 `Running`，这个可能需要一些时间

  ```bash
  $ kubectl get emqxenterprises
  NAME      STATUS   AGE
  emqx-ee   Running  8m33s
  ```

+ 获取 EMQX 集群的 External IP，访问 EMQX 控制台

  ```bash
  $ kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip'

  192.168.1.200
  ```
  通过浏览器访问 `http://192.168.1.200:18083` ，使用默认的用户名和密码 `admin/public` 登录 EMQX 控制台。

:::
::::

## 更新 License

:::: tabs type:card
::: tab apps.emqx.io/v2beta1

+ 查看 License 信息

  ```bash
  $ pod_name="$(kubectl get pods -l 'apps.emqx.io/instance=emqx-ee,apps.emqx.io/db-role=core' -o json | jq --raw-output '.items[0].metadata.name')"
  $ kubectl exec -it ${pod_name} -c emqx -- emqx_ctl license info
  ```

  可以获取到如下输出，从输出结果可以看到我们申请的 License 的基本信息，包括申请人的信息和 License 支持最大连接数以及 License 过期时间等。
  ```bash
  customer        : Evaluation
  email           : contact@emqx.io
  deployment      : default
  max_connections : 100
  start_at        : 2023-01-09
  expiry_at       : 2028-01-08
  type            : trial
  customer_type   : 10
  expiry          : false
  ```

+ 修改 EMQX 自定义资源以更新 License
  ```bash
  $ kubectl edit emqx emqx-ee
  ...
  spec:
    image: emqx/emqx-enterprise:5.6
    config:
      data: |
        license {
          key = "${new_license_key}"
        }
  ...
  ```

  + 查看 EMQX 集群 License 是否被更新

  ```bash
  $ pod_name="$(kubectl get pods -l 'apps.emqx.io/instance=emqx-ee,apps.emqx.io/db-role=core' -o json | jq --raw-output '.items[0].metadata.name')"
  $ kubectl exec -it ${pod_name} -c emqx -- emqx_ctl license info
  ```
  可以获取到类似如下的信息，从获取到 `max_connections` 字段可以看出 License 的内容已经更新，则说明 EMQX 企业版 License 更新成功。若证书信息没有更新，可以等待一会，License 的更新会有些时延。

  ```bash
  customer        : Evaluation
  email           : contact@emqx.io
  deployment      : default
  max_connections : 100000
  start_at        : 2023-01-09
  expiry_at       : 2028-01-08
  type            : trial
  customer_type   : 10
  expiry          : false
  ```
:::
::: tab apps.emqx.io/v1beta4
+ 查看 License 信息

  ```bash
  $ kubectl exec -it emqx-ee-core-0 -c emqx -- emqx_ctl license info
  ```

  可以获取到如下输出，从输出结果可以看到我们申请的 License 的基本信息，包括申请人的信息和 License 支持最大连接数以及 License 过期时间等。

  ```bash
  customer        : EMQ
  email           : cloudnative@emqx.io
  deployment      : deployment-6159820
  max_connections : 10000
  start_at        : 2023-02-16
  expiry_at       : 2023-05-17
  type            : trial
  customer_type   : 0
  expiry          : false
  ```

+ 更新 EMQX 企业版 License Secret

  ```bash
  $ kubectl create secret generic ${your_license_name} --from-file=emqx.lic=${/path/to/license/file} --dry-run -o yaml | kubectl apply -f -
  ```
+ 查看 EMQX 集群 License 是否被更新

  ```bash
  $ kubectl exec -it emqx-ee-0 -c emqx -- emqx_ctl license info
  ```

  可以获取到类似如下的信息，从获取到 `max_connections` 字段可以看出 License 的内容已经更新，则说明 EMQX 企业版 License 更新成功。若证书信息没有更新，可以等待一会，License 的更新会有些时延。

  ```bash
  customer                 : cloudnative
  email                    : cloudnative@emqx.io
  max_connections          : 100000
  original_max_connections : 100000
  issued_at                : 2022-11-21 02:49:35
  expiry_at                : 2022-12-01 02:49:35
  vendor                   : EMQ Technologies Co., Ltd.
  version                  : 4.4.14
  type                     : official
  customer_type            : 2
  expiry                   : false
  ```
:::
::::
