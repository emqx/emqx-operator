# Scale EMQX cluster

> [Introduction to EMQX cluster](https://docs.emqx.io/en/broker/v4.3/advanced/cluster.html)

**Note**: Now only supports the strategy of node discovery"*k8s*".

## Configuration Details

[`cluster` details](https://docs.emqx.io/en/broker/v4.3/configuration/configuration.html)

## Example

```yaml
...
cluster:
    name: emqx
    k8s:
      apiserver: https://xxxxxxxx
      service_name: emqx
      address_type: dns
      suffix: pod.cluster.local
      app_name: emqx
      namespace: default
...
```

## Dependency details

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

