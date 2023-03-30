# 配置 EMQX TLS 证书 

## 任务目标 

- 如何通过 extraVolumes 和 extraVolumeMounts 字段配置 EMQX TLS 证书。

## EMQX 集群 TLS 证书配置 

- 基于 TLS 证书创建 Secret

Secret 是一种包含少量敏感信息例如密码、令牌或密钥的对象，其文档可以参考：[Secret](https://kubernetes.io/zh-cn/docs/concepts/configuration/secret/#working-with-secrets)。在本文中我们使用 Secret 保存 TLS 证书信息，因此在创建 EMQX 集群之前我们需要基于 TLS 证书创建好 Secret。

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

将上述文件保存为：secret-tls.yaml，并执行如下命令创建 secret：

```
kubectl apply -f secret-tls.yaml 
```

输出类似于：

```
secret/emqx-tls created
```

- 配置 EMQX 集群

:::: tabs type:card
::: tab v2alpha1

EMQX CRD 支持使用 `.spec.coreTemplate.extraVolumes` 和 `.spec.coreTemplate.extraVolumeMounts` 以及 `.spec.replicantTemplate.extraVolumes` 和 `.spec.replicantTemplate.extraVolumeMounts` 字段给 EMQX 集群配置额外的卷和挂载点。在本文中我们可以使用这个两个字段为 EMQX 集群配置 TLS 证书。

Volumes 的类型有很多种，关于 Volumes 描述可以参考文档：[Volumes](https://kubernetes.io/zh-cn/docs/concepts/storage/volumes/#secret)。在本文中我们使用的是 `secret` 类型。

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
  name: emqx
spec:
  image: emqx/emqx:5.0.14
  bootstrapConfig: |
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
      replicas: 3
      extraVolumes:
        - name: emqx-tls
          secret:
            secretName: emqx-tls
      extraVolumeMounts:
        - name: emqx-tls 
          mountPath: /mounted/cert
  replicantTemplate:
    spec:
      replicas: 0
      extraVolumes:
        - name: emqx-tls
          secret:
            secretName: emqx-tls
      extraVolumeMounts:
        - name: emqx-tls 
          mountPath: /mounted/cert
  dashboardServiceTemplate:
    spec:
      type: NodePort
      ports:
        - name: "dashboard-listeners-http-bind"
          protocol: TCP
          port: 18083
          targetPort: 18083
          nodePort: 32015
  listenersServiceTemplate:
    spec:
      type: NodePort
      ports:
        - name: "ssl-default"
          protocol: TCP
          port: 8883
          targetPort: 8883
          nodePort: 32016
```

> `.spec.coreTemplate.extraVolumes` 字段配置了卷的类型为：secret，名称为：emqx-tls。`.spec.coreTemplate.extraVolumeMounts` 字段配置了 TLS 证书挂载到 EMQX 的目录为：`/mounted/cert`。`.spec.bootstrapConfig` 字段配置了 TLS 监听器证书路径，更多 TLS 监听器的配置可以参考文档：[ssllistener](https://www.emqx.io/docs/zh/v5.0/admin/cfg.html#broker-mqtt-ssl-listener)。 `.spec.listenersServiceTemplate` 字段配置了 EMQX 集群对外暴露服务的方式为：NodePort，并指定了 EMQX ssl-default 监听器 8883 端口对应的 nodePort 为 32016（nodePort 取值范围为：30000-32767）。

:::
::: tab v1beta4

EMQX CRD 支持通过 `.spec.template.spec.volumes` 和 `.spec.template.spec.emqxContainer.volumeMounts` 字段给 EMQX 集群配置卷和挂载点。在本文中我们可以使用这个两个字段为 EMQX 集群配置 TLS 证书。

Volumes 的类型有很多种，关于 Volumes 描述可以参考文档：[Volumes](https://kubernetes.io/zh-cn/docs/concepts/storage/volumes/)。在本文中我们使用的是 `secret` 类型。

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
      type: NodePort
      ports:
        - name: "mqtt-ssl-8883"
          protocol: "TCP"
          port: 8883
          targetPort: 8883
          nodePort: 32016
```

> `.spec.template.spec.volumes` 字段配置了卷的类型为：secret，名称为：emqx-tls。`.spec.template.spec.emqxContainer.volumeMounts` 字段配置了 TLS 证书挂载到 EMQX 的目录为：`/mounted/cert`。`.spec.template.spec.emqxContainer.emqxConfig` 字段配置了 TLS 监听器证书路径，更多 TLS 监听器的配置可以参考文档：[tlsexternal](https://docs.emqx.com/zh/enterprise/v4.4/configuration/configuration.html#tlsexternal)。 `.spec.serviceTemplate` 配置字段了 EMQX 集群对外暴露服务的方式为：NodePort ，并指定了 EMQX mqtt-ssl-8883 监听器 8883 端口对应的 nodePort 为 32016（nodePort 取值范围为：30000-32767）。

:::
::: tab v1beta3

EMQX CRD 支持通过 `.spec.emqxTemplate.extraVolumes` 和 `.spec.emqxTemplate.extraVolumeMounts` 字段给 EMQX 集群配置额外的卷和挂载点。在本文中我们可以使用这个两个字段为 EMQX 集群配置 TLS 证书。

Volumes 的类型有很多种，关于 Volumes 描述可以参考文档：[Volumes](https://kubernetes.io/zh-cn/docs/concepts/storage/volumes/)。在本文中我们使用的是 `secret` 类型。

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  emqxTemplate:
    image: emqx/emqx-ee:4.4.14
    extraVolumes:
      - name: emqx-tls
        secret:
          secretName: emqx-tls
    extraVolumeMounts:
      - name: emqx-tls 
        mountPath: /mounted/cert
    config:
      listener.ssl.external.cacertfile: /mounted/cert/ca.crt
      listener.ssl.external.certfile: /mounted/cert/tls.crt
      listener.ssl.external.keyfile: /mounted/cert/tls.key
      listener.ssl.external: "0.0.0.0:8883"
    serviceTemplate:
      spec:
        type: NodePort
        ports:
          - name: "mqtt-ssl-8883"
            protocol: "TCP"
            port: 8883
            targetPort: 8883
            nodePort: 32016
```

> `.spec.emqxTemplate.extraVolumes` 字段配置了卷的类型为：secret，名称为：emqx-tls。`.spec.emqxTemplate.extraVolumeMounts` 字段配置了 TLS 证书挂载到 EMQX 的目录为：`/mounted/cert`。`.spec.emqxTemplate.config` 字段配置了 TLS 监听器证书路径，更多 TLS 监听器的配置可以参考文档：[tlsexternal](https://docs.emqx.com/zh/enterprise/v4.4/configuration/configuration.html#tlsexternal)。 `.spec.emqxTemplate.serviceTemplate` 配置字段了 EMQX 集群对外暴露服务的方式为：NodePort ，并指定了 EMQX mqtt-ssl-8883 监听器 8883 端口对应的 nodePort 为 32016（nodePort 取值范围为：30000-32767）。

:::
::::

将上述文件保存为：emqx-tls.yaml，并执行如下命令部署 EMQX 集群：

```
kubectl apply -f emqx-tls.yaml
```

输出类似于：

```
emqx.apps.emqx.io/emqx created
```

- 检查 EMQX 集群是否就绪

:::: tabs type:card
::: tab v2alpha1

```bash
 kubectl get emqx emqx -o json | jq '.status.conditions[] | select( .type == "Running" and .status == "True")'
```

输出类似于：

```bash
{
  "lastTransitionTime": "2023-03-01T02:17:03Z",
  "lastUpdateTime": "2023-03-01T02:17:03Z",
  "message": "Cluster is running",
  "reason": "ClusterRunning",
  "status": "True",
  "type": "Running"
}
```

:::
::: tab v1beta4

```bash
kubectl get emqxEnterprise emqx-ee -o json | jq '.status.conditions[] | select( .type == "Running" and .status == "True")'
```

输出类似于：

```bash
{
  "lastTransitionTime": "2023-03-01T02:49:22Z",
  "lastUpdateTime": "2023-03-01T02:49:23Z",
  "message": "All resources are ready",
  "reason": "ClusterReady",
  "status": "True",
  "type": "Running"
}
```

:::
::: tab v1beta3

```bash
kubectl get emqxEnterprise emqx-ee -o json | jq '.status.conditions[] | select( .type == "Running" and .status == "True")'
```

输出类似于：

```bash
{
  "lastTransitionTime": "2023-03-01T02:49:22Z",
  "lastUpdateTime": "2023-03-01T02:49:23Z",
  "message": "All resources are ready",
  "reason": "ClusterReady",
  "status": "True",
  "type": "Running"
}
```

:::
::::

## 验证 TLS 证书是否生效

- 使用 MQTT X 连接 EMQX 集群发送消息

MQTT X 是一款完全开源的 MQTT 5.0 跨平台桌面客户端。支持快速创建多个同时在线的 MQTT 客户端连接，方便测试 MQTT/TCP、MQTT/TLS、MQTT/WebSocket 的连接、发布、订阅功能及其他 MQTT 协议特性。更多 MQTT X 的使用文档可以参考：[MQTT X](https://mqttx.app/zh/docs)。接下来我们会使用 MQTT X 连接 EMQX 集群进行消息的发送和订阅，来验证 TLS 证书是否生效。

在 MQTT X 页面点击创建新连接的按钮，按照如图所示配置 EMQX 集群节点信息和 CA 证书路径，在配置好连接信息之后，点击 connect 按钮连接 EMQX 集群：

![](./assets/configure-tls/tls-connect.png)

然后点击订阅按钮新建订阅，如图所示 MQTT X 已成功连接 EMQX 集群并且已经成功创建订阅：

![](./assets/configure-tls/sub.png)

在成功连接 EMQX 集群并创建订阅之后，我们就可以向 EMQX 集群发送消息，如下图所示：

![](./assets/configure-tls/tls-test.png)

从上面的图中可以看到，订阅端能正常接收到客户端发送的 MQTT 消息，则说明我们配置的 TLS 是生效的。
