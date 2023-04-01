# 通过 EMQX Operator 修改 EMQX 配置

## 任务目标

如何使用 bootstrapConfig 字段修改 EMQX 节点配置。

## 配置 EMQX 集群

EMQX 主配置文件为 emqx.conf，从 5.0 版本开始，EMQX 采用 [HOCON](https://www.emqx.io/docs/zh/v5.0/configuration/configuration.html#hocon-%E9%85%8D%E7%BD%AE%E6%A0%BC%E5%BC%8F) 作为配置文件格式。

EMQX CRD 支持使用 `.spec.bootstrapConfig` 字段配置 EMQX 集群，bootstrapConfig 配置可以参考文档：[bootstrapConfig](https://www.emqx.io/docs/zh/v5.0/admin/cfg.html)。这个字段只允许在创建 EMQX 集群的时候配置，不支持更新。

:::tip 

如果在创建 EMQX 之后需要修改集群配置，请通过 EMQX Dashboard 进行修改。
:::

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
```

> 在 `.spec.bootstrapConfig` 字段里面，我们为 EMQX 集群配置了一个 TCP listener，这个 listener 名称为：test，监听的端口为：1884。

将上述内容保存为：emqx-bootstrapConfig.yaml，并执行如下命令部署 EMQX 集群：

```bash
$ kubectl apply -f emqx-bootstrapConfig.yaml

emqx.apps.emqx.io/emqx created
```

检查 EMQX 集群状态，请确保 `STATUS` 为 `Running`，这可能需要一些时间等待 EMQX 集群准备就绪。

```
$ kubectl get emqx

NAME   IMAGE      STATUS    AGE
emqx   emqx:5.0   Running   2m55s
```

## 验证配置更改

查看 EMQX 集群 listener 信息 

```bash
kubectl exec -it emqx-core-0 -c emqx -- emqx_ctl listeners 
```

输出类似于：

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

从输出结果可以看到我们配置的名称为 test 的 listener 已经生效。
