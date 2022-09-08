# 部署 EMQX 插件（以 `emqx_prometheus` 为例）

## 前置条件

+ 成功部署 EMQX Operator
+ 成功部署 EMQX 自定义资源

>[参考getting-started部署Operator及EMQX 实例](../getting-started/getting-started.md)

## 配置 `emqx_prometheus` 插件

```YAML
cat << "EOF" | kubectl apply -f -
apiVersion: apps.emqx.io/v1beta3
kind: EmqxPlugin
metadata:
  name: emqx-prometheus
  namespace: default
spec:
  selector:
   "foo": "bar"
  pluginName: emqx_prometheus
  config:
    "prometheus.push.gateway.server": "http://prometheus-pushgateway.prom.svc.cluster.local:9091"
EOF
```

## 检查 `emqx_prometheus` 是否创建成功

   ```bash
   kubectl get emqxplugins.apps.emqx.io | grep prometheus
   emqx-prometheus    2m37s
   ```

## 检查 EMQX 实例插件是否生效

   ```bash
   kubectl exec -it emqx-0 -- emqx_ctl plugins list | grep prometheus
   Plugin(emqx_prometheus, description=Prometheus for EMQ X, active=true)
   ```