## Common Configuration

### Replicas

Specify the number of EMQX instances

Example:

```yaml
spec:
  replicas: 3
```

Field Description:

| Field | Default | Description |
| --- | --- | --- |
| replicas | 3 | Number of EMQX instances |

### Environment Variables

Used to set the environment variables of the instance

Example:

```yaml
spec:
	env:
    - name: Foo
      value: Bar
```

Field Description:

| Field | Description |
| --- | --- |
| .spec.env.name | variable name |
| .spec.env.value | variable value |

### Image pull secret

Example:

```yaml
spec:
	imagePullSecrets: [fake-secrets]
```

### Node Configuration

- nodeName

nodeName is a more direct form of node selection than affinity or nodeSelector.

If the `nodeName` field is not empty, the scheduler ignores the Pod and the kubelet on the named node tries to place the Pod on that node. Using `nodeName` overrules using `nodeSelector` or affinity and anti-affinity rules.

Example:

```yaml
spec:
	nodeName: kube-01
```

Schedule to node kube-01

- nodeSelector

`nodeSelector` is the simplest recommended form of node selection constraint. Kubernetes only schedules the Pod onto nodes that have each of the labels you specify.

Example:

```yaml
spec:
	nodeSelector:
		key: value
```

Schedule to the node which is labeled with key=value

### Node Affinity

Node affinity is conceptually similar to `nodeSelector`, allowing you to constrain which nodes your Pod can be scheduled on based on node labels.

Example:

```yaml
spec:
	affinity: [config of affinity]
```

Please refer to [Kubernetes Docs](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity)

### Tolerations

Tolerations are applied to pods. Tolerations allow the scheduler to schedule pods with matching taints.

Taints and tolerations work together to ensure that pods are not scheduled onto inappropriate nodes.

Example:

```yaml
spec:
	toleRations:
		- key: "key"
			operator: "Equal"
			value: "value"
			effect: "NoSchedule"
```

Please refer to [Kubernetes Docs](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/)

### Persistence

pvc configuration

Example:

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

| Field | Default | Description |
| --- | --- | --- |
| .spec.persistent.storageClassName | standard | the name of storage class |
| .spec.persistent.resources.requests.storage | 20Mi | storage size |
| .spec.persistent.accessModes | ReadWriteOnce | access mode，only support ReadWriteOnce |

### Init Containers

Init containers can contain utilities or setup scripts not present in an app image.

A Pod can have multiple containers running apps within it, but it can also have one or more init containers, which are run before the app containers are started.


Example:

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

This configuration above is used to perform kernel and network optimizations for the EMQX container


### Extra Containers

Similar to the side-car container, it can run simultaneously with the EMQX container and be used to process the user-defined routines.

Example:

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

## EMQX Template

### EMQX Dashboard

Dashboard Account configurations

Example:

```yaml
spec:
	emqxTemplate:
    username: "admin"
    password: "public"
```

Field Description:

| Field | Default | Description |
| --- | --- | --- |
| .spec.emqxTemplate.imageusername | admin | username |
| .spec.emqxTemplate.imagepassword | public | password |

### Image configuration

specify image and pull policy

Example:

```yaml
spec:
	emqxTemplate:
		image: emqx/emqx-ee:4.4.8
		imagePullPolicy: IfNotPresent
```

Field Description:

| Field | Default | Description |
| --- | --- | --- |
| .spec.emqxTemplate.image |  | image address |
| .spec.emqxTemplate.imagePullPolicy | IfNotPresent | IfNotPresent: the image is pulled only if it is not already present locally.|

Please refer to [Kubernetes Docs](https://kubernetes.io/docs/concepts/containers/images/)

### Security Context

A security context defines privilege and access control settings for a Pod or Container.

Example:

```yaml
spec:
	emqxTemplate:
		securityContext:
      runAsUser: 1000
      runAsGroup: 1000
      fsGroup: 1000
      fsGroupChangePolicy: Always
```

Please refer to [Kubernetes Docs](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/)

### Extra Volumes

Mount extra volumes, eg: secret or configmap

Example:

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

Please refer to [issue](https://github.com/emqx/emqx-operator/pull/72)

### EMQX Entrypoint

The entrypoint for the EMQX container, or if not provided, then the CMD for the EMQX image is used.

Example:

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

 Please refer to [Kubernetes Docs](https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/)

### EMQX Configurations

Example:

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

**Note**: `spec.env` override `spec.emqxTemplate.config`

Please refer to [EMQX Docs](https://docs.emqx.com/en/enterprise/v4.4/configuration/configuration.html)

### ACL

EMQX ACL configuration

Example:

```yaml
spec:
	emqxTemplate:
		- "{allow, all}."
```

Please refer to [EMQX Docs](https://docs.emqx.com/en/enterprise/v4.4/advanced/acl.html)

### EMQX Modules

Example:

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

Please refer to [EMQX Docs](https://docs.emqx.com/en/enterprise/v4.4/modules/modules.html)

### EMQX License

Example:

```yaml
spec:
	emqxTemplate:
		license:
			data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUVOekNDQXgrZ0F3SUJBZ0lEZE12Vk1BMEdDU3FHU0liM0RRRUJCUVVBTUlHRE1Rc3dDUVlEVlFRR0V3SkQKVGpFUk1BOEdBMVVFQ0F3SVdtaGxhbWxoYm1jeEVUQVBCZ05WQkFjTUNFaGhibWQ2YUc5MU1Rd3dDZ1lEVlFRSwpEQU5GVFZFeEREQUtCZ05WQkFzTUEwVk5VVEVTTUJBR0ExVUVBd3dKS2k1bGJYRjRMbWx2TVI0d0hBWUpLb1pJCmh2Y05BUWtCRmc5NmFHRnVaM2RvUUdWdGNYZ3VhVzh3SGhjTk1qQXdOakl3TURNd01qVXlXaGNOTkRrd01UQXgKTURNd01qVXlXakJqTVFzd0NRWURWUVFHRXdKRFRqRVpNQmNHQTFVRUNnd1FSVTFSSUZnZ1JYWmhiSFZoZEdsdgpiakVaTUJjR0ExVUVBd3dRUlUxUklGZ2dSWFpoYkhWaGRHbHZiakVlTUJ3R0NTcUdTSWIzRFFFSkFSWVBZMjl1CmRHRmpkRUJsYlhGNExtbHZNSUlCSWpBTkJna3Foa2lHOXcwQkFRRUZBQU9DQVE4QU1JSUJDZ0tDQVFFQXJ3KzMKMnc5QjdScjNNN0lPaU1jN09EM056djJLVXd0SzZPU1EwN1k3aWtESmgwanluV2N3NlFhbVRpUldNMkFsZThqcgowWEFtS2d3VVNJNDIrZjR3ODRuUHBBSDRrMUwwenVwYVIxMFZZS0lvd1pxWFZFdlN5VjhHMk43MDkxKzZKY29uCkRjYU5CcVpMUmUxRGlaWE1KbGhYbkRncTE0RlBBeGZmS2hDWGlDZ1l0bHVMRERMS3YrdzlCYVFHWlZqeGxGZTUKY3czMit6L3hIVTM2Nm5wSEJwYWZDYnhCdFdzTnZjaE1WdExCcXY5eVBtck1xZUJST3lvSmFJM25MNzh4RGdwZApjUm9ycW8rdVExSFdkY002SW5FRkVUNnB3a2V1QUY4L2pKUmxUMTJYR2daS0tnRlFUQ2taaTRodjdheXdrR0JFCkpydVBpZi93bEswWXVQSnU2UUlEQVFBQm80SFNNSUhQTUJFR0NTc0dBUVFCZzVvZEFRUUVEQUl4TURDQmxBWUoKS3dZQkJBR0RtaDBDQklHR0RJR0RaVzF4ZUY5aVlXTnJaVzVrWDNKbFpHbHpMR1Z0Y1hoZlltRmphMlZ1WkY5dAplWE54YkN4bGJYRjRYMkpoWTJ0bGJtUmZjR2R6Y1d3c1pXMXhlRjlpWVdOclpXNWtYMjF2Ym1kdkxHVnRjWGhmClltRmphMlZ1WkY5allYTnpZU3hsYlhGNFgySnlhV1JuWlY5cllXWnJZU3hsYlhGNFgySnlhV1JuWlY5eVlXSmkKYVhRd0VBWUpLd1lCQkFHRG1oMERCQU1NQVRFd0VRWUpLd1lCQkFHRG1oMEVCQVFNQWpFd01BMEdDU3FHU0liMwpEUUVCQlFVQUE0SUJBUURIVWU2K1AyVTRqTUQyM3U5NnZ4Q2VRcmhjL3JYV3ZwbVU1WEI4US9WR25KVG12M3lVCkVQeVRGS3RFWllWWDI5ejE2eG9pcFVFNmNybEhoRVRPZmV6WXNtOUswRHhGM2ZOaWxPTFJLa2c5VkVXY2I1aGoKaUwzYTJ0ZFo0c3EraC9aMWVsSVhENzFKSkJBSW1qcjZCbGpUSWRVQ2ZWdE52eGxFOE0wRC9yS1NuMmp3enNqSQpVclc4OFRITXRsejlzYjU2a21NM0pJT29JSm9lcDZ4TkVhaklCbm9DaFNHanRCWUZORnd6ZHdTVENvZFlrZ1B1CkppZnF4VEtTdXdBR1NscXhKVXdoaldHOHVsekwzL3BDQVlFd2xXbWQyK25zZm90UWRpQU5kYVBuZXo3bzB6MHMKRXVqT0NaTWJLOHFOZlNieW81MHE1aUlYaHoyWklHbCs0aGRwCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
```

Field Description:

| Field | Description |
| --- | --- |
| .spec.emqxTemplate.license.data | license content，base64 or raw |

### Probes

- readinessProbe

Periodically check the readiness of the EMQX container

Example:

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

Field Description:

| Field | Default | Description |
| --- | --- | --- |
| .spec.emqxTemplate.readinessProbe.httpGet.path |  | Path to access on the HTTP serve |
| .spec.emqxTemplate.readinessProbe.httpGet.port |  | Name or number of the port to access on the container |
| .spec.emqxTemplate.readinessProbe.initialDelaySeconds | 0 | The initialDelaySeconds field tells the kubelet that it should wait 10 seconds before performing the first probe |
| .spec.emqxTemplate.readinessProbe.periodSeconds | 10 | The periodSeconds field specifies that the kubelet should perform a liveness probe every 5 seconds.  |
| .spec.emqxTemplate.readinessProbe.failureThreshold | 3 | When a probe fails, Kubernetes will try failureThreshold times before giving up. |

- livenessProbe

Periodically check the liveness of the EMQX container

Example:

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

Field Description:

| Field | Default | Description |
| --- | --- | --- |
|  |  |  |
| .spec.emqxTemplate.livenessProbe.httpGet.path |  | Path to access on the HTTP serve |
| .spec.emqxTemplate.livenessProbe.httpGet.port |  | Name or number of the port to access on the container |
| .spec.emqxTemplate.livenessProbe.initialDelaySeconds | 0 | The initialDelaySeconds field tells the kubelet that it should wait 60 seconds before performing the first probe |
| .spec.emqxTemplate.livenessProbe.periodSeconds | 10 | The periodSeconds field specifies that the kubelet should perform a liveness probe every 30 seconds. |
| .spec.emqxTemplate.livenessProbe.failureThreshold | 3 | When a probe fails, Kubernetes will try failureThreshold times before giving up. |

- startupProbe

Check if the EMQX container started successfully

Example:

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

Field Description:

| Field | Default | Description |
| --- | --- | --- |
|  |  |  |
| .spec.emqxTemplate.startupProbe.httpGet.path |  | Path to access on the HTTP serve  |
| .spec.emqxTemplate.startupProbe.httpGet.port |  | Name or number of the port to access on the container |
| .spec.emqxTemplate.startupProbe.initialDelaySeconds | 0 | The initialDelaySeconds field tells the kubelet that it should wait 10 seconds before performing the first probe |
| .spec.emqxTemplate.startupProbe.periodSeconds | 10 | The periodSeconds field specifies that the kubelet should perform a liveness probe every 5 seconds. |
| .spec.emqxTemplate.startupProbe.failureThreshold | 3 | When a probe fails, Kubernetes will try failureThreshold times before giving up. |

Please refer to [Kubernetes Docs](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)

### Resource Configurations

Cpu and memory of EMQX pod configurations

Example:

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

Field Description:

| Field | Description |
| --- | --- |
| .spec.emqxTemplate.resources.requests | resource requests |
| .spec.emqxTemplate.resources.requests.limits | resource limits |

Please refer to [Kubernetes Docs](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)

### Service Template

EMQX service template configurations

Example:

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