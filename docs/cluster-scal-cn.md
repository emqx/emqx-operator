# Emqx 集群扩容

> [emqx分布式集群简介](https://docs.emqx.cn/enterprise/v4.3/getting-started/cluster.html)

**`Operator`目前仅支持基于`k8s`节点发现与自动集群**

## 配置明细

[`cluster` 配置明细](https://docs.emqx.cn/enterprise/v4.3/configuration/configuration.html#cluster)

## `Operator cluster yaml` 配置示例

```yaml
cluster:
    name: emqx
    k8s:   
      apiserver: https://xxxxxxxx
      service_name: emqx
      address_type: dns
      suffix: pod.cluster.local
      app_name: emqx
      namespace: default
```

## 配置依赖简述

<table>
    <tr>
        <th>
        cluster.discovery
        </th>
        <th>
        cluster.k8s.address_type    
        </th>
        <th>
        cluster.k8s.suffix
        </th>
    </tr>
    <tr>
        <td rowspan="3">k8s</td>
        <td>hostname</td>
        <td>svc.cluster.local</td>
    </tr>
    <tr>
        <td>ip</td>
        <td>空</td>
    </tr>
    <tr>
        <td>dns</td>
        <td>pod.cluster.local</td>
    </tr>
</table>
