# EMQX

例子

```yaml
apiVersion: apps.emqx.io/v1beta2
kind: EmqxBroker
metadata:
  name: emqx
spec:
  serviceAccountName: "emqx"
  image: emqx/emqx:4.3.11
  replicas: 3
  labels:
    cluster: emqx
  storage:
    storageClassName: standard
    resources:
      requests:
        storage: 20Mi
    accessModes:
    - ReadWriteOnce
  emqxTemplate:
    listener:
      type: ClusterIP
      ports:
        mqtt: 1883
        mqtts: 8883
        ws: 8083
        wss: 8084
        dashboard: 18083
        api: 8081
    acl:
      - permission: allow
        username: "dashboard"
        action: subscribe
        topics:
          filter:
            - "$SYS/#"
            - "#"
      - permission: allow
        ipaddress: "127.0.0.1"
        topics:
          filter:
            - "$SYS/#"
          equal:
            - "#"
      - permission: deny
        action: subscribe
        topics:
          filter:
            - "$SYS/#"
          equal:
            - "#"
      - permission: allow
    plugins:
      - name: emqx_management
        enable: true
      - name: emqx_recon
        enable: true
      - name: emqx_retainer
        enable: true
      - name: emqx_dashboard
        enable: true
      - name: emqx_telemetry
        enable: true
      - name: emqx_rule_engine
        enable: true
      - name: emqx_bridge_mqtt
        enable: false
    modules:
      - name: emqx_mod_acl_internal
        enable: true
      - name: emqx_mod_presence
        enable: true
```

- 配置镜像地址

```yaml
image: my-repo/emqx:4.3.10
```

- 配置存储

```yaml
storage:
volumeClaimTemplate:
  spec:
    storageClassName: standard
    resources:
      requests:
        storage: 20Mi
    accessModes:
    - ReadWriteOnce
```

| Field | Default | Description |
| --- | --- | --- |
| storageClassName | standard | 存储类的名称 |
| storage | 20Mi | 存储容量 |
| accessModes | ReadWriteOnce | 挂载模式，有 ReadWriteOnce，ReadOnlyMany 和 ReadWriteMany |
- 配置 Load Balancer

```yaml
listener:
  type: ClusterIP
  ports:
    mqtt: 1883
    mqtts: 8883
    ws: 8083
    wss: 8084
    dashboard: 18083
    api: 8081
```

| Field | Default | Description |
| --- | --- | --- |
| type | ClusterIP | 监听器类型，可以配置为 ClusterIP 或者 LoadBalancer |
| ports | mqtt: 1883 mqtts: 8883 ws: 8083 wss: 8084 dashboard: 18083 api: 8081 | EMQX 端口名称 详细请参考 [EMQX 官方文档](https://www.emqx.io/docs/zh/v4.3/tutorial/deploy.html) |

> 当 type 设置为 LoadBalancer，各个云平台需要配置不同的 annotations，例如在 EKS 平台设置为: `service.beta.kubernetes.io/aws-load-balancer-type: nlb`
>

- 配置ACL

```yaml
acl:
  - permission: allow
    username: "dashboard"
    action: subscribe
    topics:
      filter:
        - "$SYS/#"
        - "#"
  - permission: allow
    ipaddress: "127.0.0.1"
    topics:
      filter:
        - "$SYS/#"
      equal:
        - "#"
  - permission: deny
    action: subscribe
    topics:
      filter:
        - "$SYS/#"
      equal:
        - "#"
```

| Field | Description |
| --- | --- |
| permission | 执行权限控制操作，可取值为 allow，deny |
| username | 用户名 (username)为 "dashboard" 的用户生效 |
| action | 规则所控制的操作，可取值为 publish，subscribe，pubsub |
| topics.filter | 主题过滤器，表示规则可以匹配的主题 |
- 配置插件（plugins）

```yaml
plugins:
  - name: emqx_management
    enable: true
  - name: emqx_recon
    enable: true
  - name: emqx_retainer
    enable: true
  - name: emqx_dashboard
    enable: true
  - name: emqx_telemetry
    enable: true
  - name: emqx_rule_engine
    enable: true
  - name: emqx_bridge_mqtt
    enable: false
```

| Field | Description |
| --- | --- |
| name | 插件名称 |
| enable | 启用或者禁用插件，可取值 true 或者 false |

> 详细信息可参考[官方文档](https://docs.emqx.cn/broker/v4.3/advanced/plugins.html#%E6%8F%92%E4%BB%B6%E5%88%97%E8%A1%A8)
>
- 配置模块（modules）

```yaml
modules:
  - name: emqx_mod_acl_internal
    enable: true
  - name: emqx_mod_presence
    enable: true
```

| Field | Description |
| --- | --- |
| name | 模块名称 |
| enable | 启用或者禁用模块，可取值true或者false |

> 详细信息可参考[官方文档](https://docs.emqx.cn/broker/v4.3/advanced/internal-modules.html)
>

- 配置 license（针对 EMQX 企业版）

| Field | Default | Description |
| --- | --- | --- |
| license | 不配 | license内容 |
- 配置Prometheus监控

```yaml
plugins:
  - name: emqx_prometheus
    enable: true
env:
  - name: EMQX_PROMETHEUS__PUSH__GATEWAY__SERVER
    value: ${push_gateway_url}:9091
```

| Field | Description |
| --- | --- |
| env.name | 环境变量的名称 |
| env.value | 环境变量的值 |