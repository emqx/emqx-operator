# 扩展 EMQX 集群

[EMQX 集群介绍](https://docs.emqx.io/en/broker/v4.3/advanced/cluster.html)

**Note**: 现在只支持k8s的节点发现策略

## 集群详细配置

[集群配置](https://docs.emqx.io/en/broker/v4.3/configuration/configuration.html)

## 集群配置例子

```
...
cluster:
    name: emqx
    k8s:
      apiserver: <https://xxxxxxxx>
      service_name: emqx
      address_type: dns
      suffix: pod.cluster.local
      app_name: emqx
      namespace: default
...

```

## 依赖

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
    </tr>
    <tr>
        <td>dns</td>
        <td>pod.cluster.local</td>
    </tr>
</table>