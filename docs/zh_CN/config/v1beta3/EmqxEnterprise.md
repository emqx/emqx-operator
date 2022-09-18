## 通用配置

### Replicas 配置

指定 EMQX 实例个数

例子：

```yaml
spec:
  replicas: 3
```

字段说明：

| 字段 | 默认值 | 描述 |
| --- | --- | --- |
| replicas | 3 | EMQX实例个数 |

### 环境变量配置

用来设置实例的环境变量

例子：

```yaml
spec:
	env:
    - name: Foo
      value: Bar
```

字段说明：

| 字段 | 描述 |
| --- | --- |
| .spec.env.name | 变量名 |
| .spec.env.value | 变量值 |

### 镜像拉取密钥

例子：

```yaml
spec:
	imagePullSecrets: [fake-secrets]
```

### 节点配置

- nodeName

通过这些配置来进行Pod的调度。

如果 `nodeName`字段不为空，调度器会忽略该 Pod， 而指定节点上的 kubelet 会尝试将 Pod 放到该节点上。 使用 `nodeName` 规则的优先级会高于使用 `nodeSelector`或亲和性与非亲和性的规则。

例子：

```yaml
spec:
	nodeName: kube-01
```

将调度到节点 kube-01

- nodeSelector

`nodeSelector`是节点选择约束的最简单推荐形式。Kubernetes 只会将 Pod 调度到拥有你所指定的每个标签的节点上。

例子：

```yaml
spec:
	nodeSelector:
		key: value
```

指定调度节点为带有label标记为 key=value 的 node 节点

### 亲和性配置

节点亲和性概念上类似于 `nodeSelector` 它使你可以根据节点上的标签来约束 Pod 可以调度到哪些节点上

例子

```yaml
spec:
	affinity: [config of affinity]
```

详情请参考 [Kubernetes 官方文档](https://kubernetes.io/zh-cn/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity)

### 容忍度配置

如果一个节点标记为 Taints ，除非 POD 也被标识为可以容忍污点节点，否则该 Taints 节点不会被调度pod。

如果仍然希望某个 POD 调度到 taint 节点上，则必须在 Spec 中做出`Toleration`定义，才能调度到该节点。

例子：

```yaml
spec:
	toleRations:
		- key: "key"
			operator: "Equal"
			value: "value"
			effect: "NoSchedule"
```

详情请参考 [Kubernetes 官方文档](https://kubernetes.io/zh-cn/docs/concepts/scheduling-eviction/taint-and-toleration/)

### 持久化配置

配置pvc属性

例子

```yaml
spec:
  persistent:
    storageClassName: standard
    resources:
      requests:
        storage: 20Mi
    accessModes:
    - ReadWriteOnce
```

| 字段 | 默认值 | 描述 |
| --- | --- | --- |
| .spec.persistent.storageClassName | standard | storage class名称 |
| .spec.persistent.resources.requests.storage | 20Mi | storage容量大小 |
| .spec.persistent.accessModes | ReadWriteOnce | 访问模式，只推荐使用ReadWriteOnce |

### 初始化容器配置

Init 容器是一种特殊容器，在 [Pod](https://kubernetes.io/docs/concepts/workloads/pods/pod-overview/) 内的应用容器启动之前运行。Init 容器可以包括一些应用镜像中不存在的实用工具和安装脚本。

例子：

```yaml
spec:
	name: busybox
  image: busybox:stable
  securityContext:
    runAsUser: 0
    runAsGroup: 0
    capabilities:
      add:
      - SYS_ADMIN
      drop:
      - ALL
  command:
    - /bin/sh
    - -c
    - |
      mount -o remount rw /proc/sys
      sysctl -w net.core.somaxconn=65535
      sysctl -w net.ipv4.ip_local_port_range="1024 65535"
      sysctl -w kernel.core_uses_pid=0
      sysctl -w net.ipv4.tcp_tw_reuse=1
      sysctl -w fs.nr_open=1000000000
      sysctl -w fs.file-max=1000000000
      sysctl -w net.ipv4.ip_local_port_range='1025 65534'
      sysctl -w net.ipv4.udp_mem='74583000 499445000 749166000'
      sysctl -w net.ipv4.tcp_max_sync_backlog=163840
      sysctl -w net.core.netdev_max_backlog=163840
      sysctl -w net.core.optmem_max=16777216
      sysctl -w net.ipv4.tcp_rmem='1024 4096 16777216'
      sysctl -w net.ipv4.tcp_wmem='1024 4096 16777216'
      sysctl -w net.ipv4.tcp_max_tw_buckets=1048576
      sysctl -w net.ipv4.tcp_fin_timeout=15
      sysctl -w net.core.rmem_default=262144000
      sysctl -w net.core.wmem_default=262144000
      sysctl -w net.core.rmem_max=262144000
      sysctl -w net.core.wmem_max=262144000
      sysctl -w net.ipv4.tcp_mem='378150000  504200000  756300000'
      sysctl -w net.netfilter.nf_conntrack_max=1000000
      sysctl -w net.netfilter.nf_conntrack_tcp_timeout_time_wait=30
```

上述这段配置用来对 EMQX 容器进行内核和网络优化

## 额外容器配置

类似 side-car 容器，可以和 EMQX 容器同时运行，用来对用户的业务进行处理。

例子：

```yaml
spec:
	extraContainers:
  - name: extra
    image: busybox:stable
    command:
      - /bin/sh
      - -c
      - |
        tail -f /dev/null
```

## EMQX模版配置

### EMQX Dashboard账号配置

配置Dashboard登录账号

例子：

```yaml
spec:
	emqxTemplate:
    username: "admin"
    password: "public"
```

字段说明：

| 字段 | 默认值 | 描述 |
| --- | --- | --- |
| .spec.emqxTemplate.imageusername | admin | 用户名 |
| .spec.emqxTemplate.imagepassword | public | 密码 |

### 镜像配置

指定 EMQX 的镜像和拉取策略

例子：

```yaml
spec:
	emqxTemplate:
		image: emqx/emqx-ee:4.4.8
		imagePullPolicy: IfNotPresent
```

字段说明：

| 字段 | 默认值 | 描述 |
| --- | --- | --- |
| .spec.emqxTemplate.image |  | 企业版镜像地址 |
| .spec.emqxTemplate.imagePullPolicy | IfNotPresent | IfNotPresent
IfNotPresent: 只有当镜像在本地不存在时才会拉取。 |

详情请参考 [Kubernetes 官方文档](https://kubernetes.io/zh-cn/docs/concepts/containers/images/)

### 上下文安全配置

安全上下文（Security Context）定义 Pod 或 Container 的特权与访问控制设置。

例子：

```yaml
spec:
	emqxTemplate:
		securityContext:
      runAsUser: 1000
      runAsGroup: 1000
      fsGroup: 1000
      fsGroupChangePolicy: Always
```

详情请参考 [Kubernetes 官方文档](https://kubernetes.io/zh-cn/docs/tasks/configure-pod-container/security-context/)

### 额外卷配置

挂载额外的卷，例如：secret 或者 configmap。

例子：

```yaml
spec:
	emqxTemplate:
		extraVolumes:
      - name: fake-volume
        emptyDir: {}
    extraVolumeMounts:
      - name: fake-volume
        mountPath: /tmp/fake
```

参考这个 [issue](https://github.com/emqx/emqx-operator/pull/72)

### EMQX entrypoint 配置

 EMQX 容器的 entrypoint，如果没有提供，那么就使用EMQX 镜像的CMD。

例子：

```yaml
spec:
	emqxTemplate:
		args:
      - bash
      - -c
      - |
        printenv | grep "^EMQX_"
        emqx foreground
```

 详情请参考 [Kubernetes 官方文档](https://kubernetes.io/zh-cn/docs/tasks/inject-data-application/define-command-argument-container/)

### EMQX 参数配置

EMQX 参数配置项

例子：

```yaml
spec:
	emqxTemplate:
		config:
      name: emqx-ee
      cluster.discovery: dns
      cluster.dns.type: srv
      cluster.dns.app: emqx-ee
      cluster.dns.name: emqx-ee-headless.default.svc.cluster.local
      listener.tcp.external: "1883"
```

**注意**：`spec.env` 会覆盖 `spec.emqxTemplate.config`

详情请参考 [EMQX 官方文档](https://docs.emqx.com/zh/enterprise/v4.4/configuration/configuration.html)

### ACL配置

EMQX ACL 规则

例子：

```yaml
spec:
	emqxTemplate:
		- "{allow, all}."
```

详情请参考 [EMQX 官方文档](https://docs.emqx.com/zh/enterprise/v4.4/advanced/acl.html)

### EMQX 模块配置

配置 EMQX 中的功能模块

例子：

```yaml
spec:
	emqxTemplate:
		modules:
      - name: "internal_acl"
        enable: true
        configs:
          acl_rule_file: "/mounted/acl/acl.conf"
      - name: "retainer"
        enable: true
        configs:
          expiry_interval: 0
          max_payload_size: "1MB"
          max_retained_messages: 0
          storage_type: "ram"
```

详情请参考 [EMQX 官方文档](https://docs.emqx.com/zh/enterprise/v4.4/modules/modules.html)

### EMQX License配置

配置 EMQX License

例子：

```yaml
spec:
	emqxTemplate:
		license:
			data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUVOekNDQXgrZ0F3SUJBZ0lEZE12Vk1BMEdDU3FHU0liM0RRRUJCUVVBTUlHRE1Rc3dDUVlEVlFRR0V3SkQKVGpFUk1BOEdBMVVFQ0F3SVdtaGxhbWxoYm1jeEVUQVBCZ05WQkFjTUNFaGhibWQ2YUc5MU1Rd3dDZ1lEVlFRSwpEQU5GVFZFeEREQUtCZ05WQkFzTUEwVk5VVEVTTUJBR0ExVUVBd3dKS2k1bGJYRjRMbWx2TVI0d0hBWUpLb1pJCmh2Y05BUWtCRmc5NmFHRnVaM2RvUUdWdGNYZ3VhVzh3SGhjTk1qQXdOakl3TURNd01qVXlXaGNOTkRrd01UQXgKTURNd01qVXlXakJqTVFzd0NRWURWUVFHRXdKRFRqRVpNQmNHQTFVRUNnd1FSVTFSSUZnZ1JYWmhiSFZoZEdsdgpiakVaTUJjR0ExVUVBd3dRUlUxUklGZ2dSWFpoYkhWaGRHbHZiakVlTUJ3R0NTcUdTSWIzRFFFSkFSWVBZMjl1CmRHRmpkRUJsYlhGNExtbHZNSUlCSWpBTkJna3Foa2lHOXcwQkFRRUZBQU9DQVE4QU1JSUJDZ0tDQVFFQXJ3KzMKMnc5QjdScjNNN0lPaU1jN09EM056djJLVXd0SzZPU1EwN1k3aWtESmgwanluV2N3NlFhbVRpUldNMkFsZThqcgowWEFtS2d3VVNJNDIrZjR3ODRuUHBBSDRrMUwwenVwYVIxMFZZS0lvd1pxWFZFdlN5VjhHMk43MDkxKzZKY29uCkRjYU5CcVpMUmUxRGlaWE1KbGhYbkRncTE0RlBBeGZmS2hDWGlDZ1l0bHVMRERMS3YrdzlCYVFHWlZqeGxGZTUKY3czMit6L3hIVTM2Nm5wSEJwYWZDYnhCdFdzTnZjaE1WdExCcXY5eVBtck1xZUJST3lvSmFJM25MNzh4RGdwZApjUm9ycW8rdVExSFdkY002SW5FRkVUNnB3a2V1QUY4L2pKUmxUMTJYR2daS0tnRlFUQ2taaTRodjdheXdrR0JFCkpydVBpZi93bEswWXVQSnU2UUlEQVFBQm80SFNNSUhQTUJFR0NTc0dBUVFCZzVvZEFRUUVEQUl4TURDQmxBWUoKS3dZQkJBR0RtaDBDQklHR0RJR0RaVzF4ZUY5aVlXTnJaVzVrWDNKbFpHbHpMR1Z0Y1hoZlltRmphMlZ1WkY5dAplWE54YkN4bGJYRjRYMkpoWTJ0bGJtUmZjR2R6Y1d3c1pXMXhlRjlpWVdOclpXNWtYMjF2Ym1kdkxHVnRjWGhmClltRmphMlZ1WkY5allYTnpZU3hsYlhGNFgySnlhV1JuWlY5cllXWnJZU3hsYlhGNFgySnlhV1JuWlY5eVlXSmkKYVhRd0VBWUpLd1lCQkFHRG1oMERCQU1NQVRFd0VRWUpLd1lCQkFHRG1oMEVCQVFNQWpFd01BMEdDU3FHU0liMwpEUUVCQlFVQUE0SUJBUURIVWU2K1AyVTRqTUQyM3U5NnZ4Q2VRcmhjL3JYV3ZwbVU1WEI4US9WR25KVG12M3lVCkVQeVRGS3RFWllWWDI5ejE2eG9pcFVFNmNybEhoRVRPZmV6WXNtOUswRHhGM2ZOaWxPTFJLa2c5VkVXY2I1aGoKaUwzYTJ0ZFo0c3EraC9aMWVsSVhENzFKSkJBSW1qcjZCbGpUSWRVQ2ZWdE52eGxFOE0wRC9yS1NuMmp3enNqSQpVclc4OFRITXRsejlzYjU2a21NM0pJT29JSm9lcDZ4TkVhaklCbm9DaFNHanRCWUZORnd6ZHdTVENvZFlrZ1B1CkppZnF4VEtTdXdBR1NscXhKVXdoaldHOHVsekwzL3BDQVlFd2xXbWQyK25zZm90UWRpQU5kYVBuZXo3bzB6MHMKRXVqT0NaTWJLOHFOZlNieW81MHE1aUlYaHoyWklHbCs0aGRwCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
```

字段说明：

| 字段 | 描述 |
| --- | --- |
| .spec.emqxTemplate.license.data | license内容，字符串表示 |

### 探针配置

- readinessProbe 探针

周期性地检查 EMQX 容器是否就绪

例子：

```yaml
spec:
	emqxTemplate:
		readinessProbe:
      httpGet:
        path: /status
        port: 8081
      initialDelaySeconds: 10
      periodSeconds: 5
      failureThreshold: 12
```

字段说明：

| 字段 | 默认值 | 描述 |
| --- | --- | --- |
| .spec.emqxTemplate.readinessProbe.httpGet.path | 无 | Get 接口 path |
| .spec.emqxTemplate.readinessProbe.httpGet.port | 无 | Get 接口 port |
| .spec.emqxTemplate.readinessProbe.initialDelaySeconds | 0 | 告诉 kubelet 在执行第一次探测前应该等待 10 秒 |
| .spec.emqxTemplate.readinessProbe.periodSeconds | 10 | 指定了 kubelet 每隔 5 秒执行一次就绪探测 |
| .spec.emqxTemplate.readinessProbe.failureThreshold | 3 | 当探测失败时，Kubernetes 的重试次数 |
- livenessProbe 探针

周期性检查 EMQX 容器是否存活

例子：

```yaml
spec:
	emqxTemplate:
		livenessProbe:
      httpGet:
        path: /status
        port: 8081
      initialDelaySeconds: 60
      periodSeconds: 30
      failureThreshold: 3
```

字段说明：

| 字段 | 默认值 | 描述 |
| --- | --- | --- |
|  |  |  |
| .spec.emqxTemplate.livenessProbe.httpGet.path | 无 | Get 接口 path |
| .spec.emqxTemplate.livenessProbe.httpGet.port | 无 | Get 接口 port |
| .spec.emqxTemplate.livenessProbe.initialDelaySeconds | 0 | 告诉 kubelet 在执行第一次探测前应该等待 60 秒 |
| .spec.emqxTemplate.livenessProbe.periodSeconds | 10 | 指定了 kubelet 每隔 30 秒执行一次存活探测 |
| .spec.emqxTemplate.livenessProbe.failureThreshold | 3 | 当探测失败时，Kubernetes 的重试次数 |

- startupProbe 探针

检查 EMQX 容器是否成功启动

例子：

```yaml
spec:
	emqxTemplate:
		startupProbe:
		  httpGet:
		    path: /status
		    port: 8081
		  initialDelaySeconds: 10
		  periodSeconds: 5
		  failureThreshold: 12
```

字段说明：

| 字段 | 默认值 | 描述 |
| --- | --- | --- |
|  |  |  |
| .spec.emqxTemplate.startupProbe.httpGet.path | 无 | Get 接口 path |
| .spec.emqxTemplate.startupProbe.httpGet.port | 无 | Get 接口 port |
| .spec.emqxTemplate.startupProbe.initialDelaySeconds | 0 | 告诉 kubelet 在执行第一次探测前应该等待 10 秒 |
| .spec.emqxTemplate.startupProbe.periodSeconds | 10 | 指定了 kubelet 每隔 5 秒执行一次启动探测 |
| .spec.emqxTemplate.startupProbe.failureThreshold | 3 | 当探测失败时，Kubernetes 的重试次数 |

详情请参考 [Kubernetes 官方文档](https://kubernetes.io/zh-cn/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)

### 容器资源配置

配置 EMQX 容器的 CPU 和内存资源

例子：

```yaml
spec:
	emqxTemplate:
		resources:
      requests:
        memory: "64Mi"
        cpu: "125m"
      limits:
        memory: "1024Mi"
        cpu: "500m"
```

字段说明：

| 字段 | 描述 |
| --- | --- |
| .spec.emqxTemplate.resources.requests | 对资源的请求 |
| .spec.emqxTemplate.resources.requests.limits | 对资源的限制 |

详情请参考 [Kubernetes 官方文档](https://kubernetes.io/zh-cn/docs/concepts/configuration/manage-resources-containers/)

### service 模版配置

配置 EMQX service 模版

例子：

```yaml
spec:
	emqxTemplate:
		metadata:
        name: emqx-ee
        namespace: default
        labels:
          "apps.emqx.io/instance": "emqx-ee"
      spec:
        type: ClusterIP
        selector:
          "apps.emqx.io/instance": "emqx-ee"
        ports:
          - name: "http-management-8081"
            port: 8081
            protocol: "TCP"
            targetPort: 8081
```