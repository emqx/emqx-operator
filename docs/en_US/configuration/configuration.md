# EMQX

Sample

```yaml
apiVersion: apps.emqx.io/v1beta2
kind: EmqxBroker
metadata:
  name: emqx
spec:
  serviceAccountName: "emqx"
  image: emqx/emqx:4.4.3
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

- Config image

```yaml
image: my-repo/emqx:4.3.10
```

- Config storage

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
| storageClassName | standard | the name of storage class |
| storage | 20Mi | storage size |
| accessModes | ReadWriteOnce | access mode，include ReadWriteOnce，ReadOnlyMany or ReadWriteMany |

- Config Load Balancer

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
| type | ClusterIP | listener type，include ClusterIP or LoadBalancer |
| ports | mqtt: 1883 mqtts: 8883 ws: 8083 wss: 8084 dashboard: 18083 api: 8081 | ports of EMQX, please [EMQX docs](https://www.emqx.io/docs/en/v4.3/tutorial/deploy.html) |

> There are different annotations depend on cloud platform. For example, `service.beta.kubernetes.io/aws-load-balancer-type: nlb` need to be set on AWS.

- Config ACL

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
| permission | the permission control operation is performed. The possible values are: allow or deny |
| username | the rule only takes effect for users whose Username is dashboard |
| action |  the operation controlled by the rule with the possible value: publish，subscribe，pubsub |
| topics.filter | which means that the rule is applied to topics |

- config plugins

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
| name | the name of plugins |
| enable | enable or disable plugins with possible values: true or false |

> Refer to [EMQX docs](https://www.emqx.io/docs/zh/v4.3/advanced/plugins.html#%E6%8F%92%E4%BB%B6%E5%88%97%E8%A1%A8)


- config modules

```yaml
modules:
  - name: emqx_mod_acl_internal
    enable: true
  - name: emqx_mod_presence
    enable: true
```

| Field | Description |
| --- | --- |
| name | the name of module |
| enable | enable or disable modules with possible values: true or false |

> Refer to [EMQX docs](https://www.emqx.io/docs/zh/v4.3/advanced/internal-modules.html)
>

- config license（only for EMQX Enterprise）

| Field | Default | Description |
| --- | --- | --- |
| license | n/a | license context |

- config prometheus monitoring

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
| env.name | the name of env |
| env.value | the value of env |