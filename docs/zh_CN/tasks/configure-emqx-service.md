# 通过 LoadBalancer 访问 EMQX 集群

## 任务目标

通过 LoadBalancer 类型的 Service 访问 EMQX 集群。

## 配置 EMQX 集群

下面是 EMQX Custom Resource 的相关配置，你可以根据希望部署的 EMQX 的版本来选择对应的 APIVersion，具体的兼容性关系，请参考 [EMQX Operator 兼容性](../index.md):

:::: tabs type:card
::: tab apps.emqx.io/v2beta1

`apps.emqx.io/v2beta1 EMQX` 支持通过 `.spec.dashboardServiceTemplate` 配置 EMQX 集群 Dashboard Service ，通过 `.spec.listenersServiceTemplate` 配置 EMQX 集群 listener Service，其文档可以参考：[Service](../reference/v2beta1-reference.md#emqxspec)。

+ 将下面的内容保存成 YAML 文件，并通过 `kubectl apply` 命令部署它

  ```yaml
  apiVersion: apps.emqx.io/v2beta1
  kind: EMQX
  metadata:
    name: emqx
  spec:
    image: emqx:5.1
    listenersServiceTemplate:
      spec:
        type: LoadBalancer
    dashboardServiceTemplate:
      spec:
        type: LoadBalancer
  ```

  > EMQX 默认会开启一个 MQTT TCP 监听器 `tcp-default` 对应的端口为1883 以及 Dashboard 监听器 `dashboard-listeners-http-bind` 对应的端口为18083 。

  > 用户可以通过 `.spec.config.data` 字段或者 EMQX Dashboard 增加新的监听器。EMQX Operator 在创建 Service 时会将缺省的监听器信息自动注入到 Service 里面，但是当用户配置的 Service 和 EMQX 配置的监听器有冲突时（name 或者 port 字段重复），EMQX Operator 会以用户的配置为准。

+ 等待 EMQX 集群就绪，可以通过 `kubectl get` 命令查看 EMQX 集群的状态，请确保 `STATUS` 为 `Running`，这个可能需要一些时间

  ```bash
  $ kubectl get emqx emqx
  NAME   IMAGE      STATUS    AGE
  emqx   emqx:5.1   Running   10m
  ```
+ 获取 EMQX 集群的 Dashboard External IP，访问 EMQX 控制台

  EMQX Operator 会创建两个 EMQX Service 资源，一个是 emqx-dashboard，一个是 emqx-listeners，分别对应 EMQX 控制台和 EMQX 监听端口。

  ```bash
  $ kubectl get svc emqx-dashboard -o json | jq '.status.loadBalancer.ingress[0].ip'

  192.168.1.200
  ```

  通过浏览器访问 `http://192.168.1.200:18083` ，使用默认的用户名和密码 `admin/public` 登录 EMQX 控制台。

:::
::: tab apps.emqx.io/v1beta4

`apps.emqx.io/v1beta4 EmqxEnterprise` 支持通过 `.spec.serviceTemplate` 字段配置 EMQX 集群 Service 。serviceTemplate 字段的具体描述可以参考：[serviceTemplate](../reference/v1beta4-reference.md#servicetemplate)。

+ 将下面的内容保存成 YAML 文件，并通过 `kubectl apply` 命令部署它

  ```yaml
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
    serviceTemplate:
      spec:
        type: LoadBalancer
  ```

  > EMQX Operator 在创建 Service 时会将缺省的监听器信息自动注入到 Service 里面，但是当用户配置的 Service 和 EMQX 配置的监听器有冲突时（ name 或者 port 字段重复），EMQX Operator 会以用户的配置为准。

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

## 通过 MQTT X CLI 连接 EMQX Cluster

+ 获取 EMQX 集群的 External IP

  :::: tabs type:card
  ::: tab apps.emqx.io/v2beta1

  ```bash
  external_ip=$(kubectl get svc emqx-listeners -o json | jq '.status.loadBalancer.ingress[0].ip')
  ```
  :::
  ::: tab apps.emqx.io/v1beta4
  ```bash
  external_ip=$(kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip')
  ```
  :::
  ::::

+ 使用 MQTT X CLI 连接 EMQX 集群

  ```bash
  $ mqttx conn -h ${external_ip} -p 1883

  [4/17/2023] [5:17:31 PM] › …  Connecting...
  [4/17/2023] [5:17:31 PM] › ✔  Connected
  ```

## 通过 EMQX Dashboard 添加监听器

:::tip
下文中 Dashboard 的截图来自是 EMQX 5，[EMQX 4 Dashboard](https://docs.emqx.com/zh/enterprise/v4.4/getting-started/dashboard-ee.html#dashboard) 也支持相应的功能，请自行操作。
:::

+ 添加监听器

  打开浏览器登录 EMQX Dashboard，点击 Configuration → Listeners 进入监听器的页面，我们先点击 Add Listener 的按钮添加一个名称为 test，端口为1884的监听器，如下图所示：

  <div style="text-align:center">
  <img src="./assets/configure-service/emqx-add-listener.png" style="zoom: 50%;" />
  </div>

  然后点击 Add 按钮创建监听器，如下图所示：

  <img src="./assets/configure-service/emqx-listeners.png" style="zoom:50%;" />

  从图中可以看出，我们创建的 test 监听器已经生效。

+ 查看新增的监听器是否注入 Service

  ```bash
  kubectl get svc

  NAME             TYPE       CLUSTER-IP       EXTERNAL-IP   PORT(S)                                         AGE
  emqx-dashboard   NodePort   10.105.110.235   <none>        18083:32012/TCP                                 13m
  emqx-listeners   NodePort   10.106.1.58      <none>        1883:32010/TCP,1884:30763/TCP                   12m
  ```

  从输出结果可以看到，刚才新增加的监听器1884已经注入到 `emqx-listeners` 这个 Service 里面。
