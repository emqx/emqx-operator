# Install EMQX plugin (eg. install `emqx_prometheus` plugin)

## Requirements

+ Install EMQX Operator successfully
+ Install EMQX CR successfully

>[Refer to getting-started](../getting-started/getting-started.md)

## Configure `emqx_prometheus` plugin

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

## Check status for `emqx_prometheus` plugin

   ```bash
   kubectl get emqxplugins.apps.emqx.io | grep prometheus
   emqx-prometheus    2m37s
   ```

## Check status in EMQX instance

   ```bash
   kubectl exec -it emqx-0 -- emqx_ctl plugins list | grep prometheus
   Plugin(emqx_prometheus, description=Prometheus for EMQ X, active=true)
   ```