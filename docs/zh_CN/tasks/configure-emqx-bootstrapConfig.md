# EMQX 配置

## 任务目标

通过 `bootstrapConfig` 字段修改 EMQX 节点配置。

## 配置 EMQX 集群

EMQX 主配置文件为 `etc/emqx.conf`，从 5.0 版本开始，EMQX 采用 [HOCON](https://www.emqx.io/docs/zh/v5.0/configuration/configuration.html#hocon-%E9%85%8D%E7%BD%AE%E6%A0%BC%E5%BC%8F) 作为配置文件格式。

`apps.emqx.io/v2alpha1 EMQX` 支持通过 `.spec.bootstrapConfig` 字段配置 EMQX 集群。bootstrapConfig 配置可以参考文档：[bootstrapConfig](https://www.emqx.io/docs/zh/v5.0/admin/cfg.html)。

:::tip
如果在创建 EMQX 之后需要修改集群配置，请通过 EMQX Dashboard 进行修改。
:::

+ 将下面的内容保存成 YAML 文件，并通过 `kubectl apply` 命令部署它

  ```yaml
  apiVersion: apps.emqx.io/v2alpha1
  kind: EMQX
  metadata:
    name: emqx
  spec:
    image: emqx:5.0
    bootstrapConfig: |
      listeners.tcp.test {
        bind = "0.0.0.0:1884"
        max_connections = 1024000
      }
    listenersServiceTemplate:
      spec:
        type: LoadBalancer
    dashboardServiceTemplate:
      spec:
        type: LoadBalancer
  ```

> 在 `.spec.bootstrapConfig` 字段里面，我们为 EMQX 集群配置了一个 TCP listener，这个 listener 名称为：test，监听的端口为：1884。

+ 等待 EMQX 集群就绪，可以通过 `kubectl get` 命令查看 EMQX 集群的状态，请确保 `STATUS` 为 `Running`，这个可能需要一些时间

  ```
  $ kubectl get emqx
  NAME   IMAGE      STATUS    AGE
  emqx   emqx:5.0   Running   2m55s
  ```

+ 获取 EMQX 集群的 Dashboard External IP，访问 EMQX 控制台

  EMQX Operator 会创建两个 EMQX Service 资源，一个是 emqx-dashboard，一个是 emqx-listeners，分别对应 EMQX 控制台和 EMQX 监听端口。

  ```bash
  $ kubectl get svc emqx-dashboard -o json | jq '.status.loadBalancer.ingress[0].ip'

  192.168.1.200
  ```

  通过浏览器访问 `http://192.168.1.200:18083` ，使用默认的用户名和密码 `admin/public` 登录 EMQX 控制台。

## 验证配置

+ 查看 EMQX 集群 listener 信息

  ```bash
  $ kubectl exec -it emqx-core-0 -c emqx -- emqx_ctl listeners
  ```

  可以获取到类似如下的打印，这意味着们配置的名称为 `test` 的 listener 已经生效。

  ```bash
  tcp:default
    listen_on       : 0.0.0.0:1883
    acceptors       : 16
    proxy_protocol  : false
    running         : true
    current_conn    : 0
    max_conns       : 1024000
  tcp:test
    listen_on       : 0.0.0.0:1884
    acceptors       : 16
    proxy_protocol  : false
    running         : true
    current_conn    : 0
    max_conns       : 1024000
  ```
