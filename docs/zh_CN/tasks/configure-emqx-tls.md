# 开启 TLS

## 任务目标

通过 `extraVolumes` 和 `extraVolumeMounts` 字段自定义 TLS 证书。

## 基于 TLS 证书创建 Secret

Secret 是一种包含少量敏感信息例如密码、令牌或密钥的对象，其文档可以参考：[Secret](https://kubernetes.io/zh-cn/docs/concepts/configuration/secret/#working-with-secrets)。在本文中我们使用 Secret 保存 TLS 证书信息，因此在创建 EMQX 集群之前我们需要基于 TLS 证书创建好 Secret。

+ 将下面的内容保存成 YAML 文件，并通过 `kubectl apply` 命令部署它

  ```yaml
  apiVersion: v1
  kind: Secret
  metadata:
    name: emqx-tls
  type: kubernetes.io/tls
  stringData:
    ca.crt: |
      -----BEGIN CERTIFICATE-----
      ...
      -----END CERTIFICATE-----
    tls.crt: |
      -----BEGIN CERTIFICATE-----
      ...
      -----END CERTIFICATE-----
    tls.key: |
      -----BEGIN RSA PRIVATE KEY-----
      ...
      -----END RSA PRIVATE KEY-----
  ```

  > `ca.crt` 表示 CA 证书内容，`tls.crt` 表示服务端证书内容，`tls.key` 表示服务端私钥内容。此例中上述三个字段的内容被省略，请用自己证书的内容进行填充。

## 配置 EMQX 集群

下面是 EMQX Custom Resource 的相关配置，你可以根据希望部署的 EMQX 的版本来选择对应的 APIVersion，具体的兼容性关系，请参考 [EMQX Operator 兼容性](../index.md):

:::: tabs type:card
::: tab apps.emqx.io/v1beta4

`apps.emqx.io/v1beta4 EmqxEnterprise` 支持通过 `.spec.template.spec.volumes` 和 `.spec.template.spec.emqxContainer.volumeMounts` 字段给 EMQX 集群配置卷和挂载点。在本文中我们可以使用这个两个字段为 EMQX 集群配置 TLS 证书。

Volumes 的类型有很多种，关于 Volumes 描述可以参考文档：[Volumes](https://kubernetes.io/zh-cn/docs/concepts/storage/volumes/)。在本文中我们使用的是 `secret` 类型。

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
          emqxConfig:
            listener.ssl.external.cacertfile: /mounted/cert/ca.crt
            listener.ssl.external.certfile: /mounted/cert/tls.crt
            listener.ssl.external.keyfile: /mounted/cert/tls.key
            listener.ssl.external: "0.0.0.0:8883"
          volumeMounts:
            - name: emqx-tls
              mountPath: /mounted/cert
        volumes:
          - name: emqx-tls
            secret:
              secretName: emqx-tls
    serviceTemplate:
      spec:
        type: LoadBalancer
  ```

  > `.spec.template.spec.volumes` 字段配置了卷的类型为：secret，名称为：emqx-tls。

  > `.spec.template.spec.emqxContainer.volumeMounts` 字段配置了 TLS 证书挂载到 EMQX 的目录为：`/mounted/cert`。

  > `.spec.template.spec.emqxContainer.emqxConfig` 字段配置了 TLS 监听器证书路径，更多 TLS 监听器的配置可以参考文档：[tlsexternal](https://docs.emqx.com/zh/enterprise/v4.4/configuration/configuration.html#tlsexternal)。

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
::: tab apps.emqx.io/v2alpha2

`apps.emqx.io/v2alpha2 EMQX` 支持通过 `.spec.coreTemplate.extraVolumes` 和 `.spec.coreTemplate.extraVolumeMounts` 以及 `.spec.replicantTemplate.extraVolumes` 和 `.spec.replicantTemplate.extraVolumeMounts` 字段给 EMQX 集群配置额外的卷和挂载点。在本文中我们可以使用这个两个字段为 EMQX 集群配置 TLS 证书。

Volumes 的类型有很多种，关于 Volumes 描述可以参考文档：[Volumes](https://kubernetes.io/zh-cn/docs/concepts/storage/volumes/#secret)。在本文中我们使用的是 `secret` 类型。

+ 将下面的内容保存成 YAML 文件，并通过 `kubectl apply` 命令部署它

  ```yaml
  apiVersion: apps.emqx.io/v2alpha2
  kind: EMQX
  metadata:
    name: emqx
  spec:
    image: emqx:5.1
    config:
      data: |
        listeners.ssl.default {
          bind = "0.0.0.0:8883"
          ssl_options {
            cacertfile = "/mounted/cert/ca.crt"
            certfile = "/mounted/cert/tls.crt"
            keyfile = "/mounted/cert/tls.key"
          }
        }
    coreTemplate:
      spec:
        extraVolumes:
          - name: emqx-tls
            secret:
              secretName: emqx-tls
        extraVolumeMounts:
          - name: emqx-tls
            mountPath: /mounted/cert
    replicantTemplate:
      spec:
        extraVolumes:
          - name: emqx-tls
            secret:
              secretName: emqx-tls
        extraVolumeMounts:
          - name: emqx-tls
            mountPath: /mounted/cert
    dashboardServiceTemplate:
      spec:
        type: LoadBalancer
    listenersServiceTemplate:
      spec:
        type: LoadBalancer
  ```

  > `.spec.coreTemplate.extraVolumes` 字段配置了卷的类型为：secret，名称为：emqx-tls。

  >`.spec.coreTemplate.extraVolumeMounts` 字段配置了 TLS 证书挂载到 EMQX 的目录为：`/mounted/cert`。

  >`.spec.config.data` 字段配置了 TLS 监听器证书路径，更多 TLS 监听器的配置可以参考文档：[配置手册](https://www.emqx.io/docs/zh/v5.1/configuration/configuration-manual.html#%E9%85%8D%E7%BD%AE%E6%89%8B%E5%86%8C)。

+ 等待 EMQX 集群就绪，可以通过 `kubectl get` 命令查看 EMQX 集群的状态，请确保 `STATUS` 为 `Running`，这个可能需要一些时间

  ```bash
  $ kubectl get emqx
  NAME   IMAGE      STATUS    AGE
  emqx   emqx:5.1   Running   10m
  ```

+ 获取 EMQX 集群的 Dashboard External IP，访问 EMQX 控制台

  EMQX Operator 会创建两个 EMQX Service 资源，一个是 emqx-dashboard，一个是 emqx-listeners，分别对应 EMQX 控制台和 EMQX 监听端口。

  ```bash
  $ kubectl get svc emqx-dashboard -o json | jq '.status.loadBalancer.ingress[0].ip'

  192.168.1.200
  ```

  通过浏览器访问 `http://192.168.1.200:18083`，使用默认的用户名和密码 `admin/public` 登录 EMQX 控制台。
:::
::::

## 使用 MQTT X CLI 验证 TLS 连接

[MQTT X CLI](https://mqttx.app/zh/cli) 是一款开源的 MQTT 5.0 命令行客户端工具，旨在帮助开发者在不需要使用图形化界面的基础上，也能更快的开发和调试 MQTT 服务与应用。

+ 获取 EMQX 集群的 External IP

  :::: tabs type:card
  ::: tab apps.emqx.io/v1beta4

  ```bash
  external_ip=$(kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip')
  ```
  :::
  ::: tab apps.emqx.io/v2alpha2

  ```bash
  external_ip=$(kubectl get svc emqx-listeners -o json | jq '.status.loadBalancer.ingress[0].ip')
  ```
  :::
  ::::

+ 使用 MQTT X CLI 订阅消息

  ```bash
  mqttx sub -h ${external_ip} -p 8883 -t "hello"  -l mqtts --insecure

  [10:00:25] › …  Connecting...
  [10:00:25] › ✔  Connected
  [10:00:25] › …  Subscribing to hello...
  [10:00:25] › ✔  Subscribed to hello
  ```

+ 创建一个新的终端窗口并使用 MQTT X CLI 发布消息

  ```bash
  mqttx pub -h ${external_ip} -p 8883 -t "hello" -m "hello world" -l mqtts --insecure

  [10:00:58] › …  Connecting...
  [10:00:58] › ✔  Connected
  [10:00:58] › …  Message Publishing...
  [10:00:58] › ✔  Message published
  ```

+ 查看订阅终端窗口收到的消息

  ```bash
  [10:00:58] › payload: hello world
  ```
