# Configure EMQX cluster

## Task target

- How to configure the EMQX cluster using the bootstrapConfig field.

## Configure EMQX cluster

The main configuration file of EMQX is emqx.conf. Starting from version 5.0, EMQX adopts [HOCON](https://www.emqx.io/docs/en/v5.0/configuration/configuration.html#hocon-configuration-format) as the configuration file format.

EMQX CRD supports using the `.spec.bootstrapConfig` field to configure the EMQX cluster. For bootstrapConfig configuration, please refer to the document: [bootstrapConfig](https://www.emqx.io/docs/en/v5.0/admin/cfg.html). This field is only allowed to be configured when creating an EMQX cluster, and does not support updating. **Note:** If you need to modify the cluster configuration after creating EMQX, please modify it through EMQX Dashboard.

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
   name: emqx
spec:
   image: emqx/emqx:5.0.14
   imagePullPolicy: IfNotPresent
   bootstrapConfig: |
    listeners.tcp.test {
       bind = "0.0.0.0:1884"
       max_connections = 1024000
     }
   coreTemplate:
     spec:
       replicas: 3
   replicantTemplate:
     spec:
       replicas: 0
```

> In the `.spec.bootstrapConfig` field, we have configured a TCP listener for the EMQX cluster. The name of this listener is: test, and the listening port is: 1884.

Save the above content as: emqx-bootstrapConfig.yaml, and execute the following command to deploy the EMQX cluster:

```bash
kubectl apply -f emqx-bootstrapConfig.yaml
```

The output is similar to:

```
emqx.apps.emqx.io/emqx created
```

- Check whether the EMQX cluster is ready

```bash
kubectl get emqx emqx -o json | jq '.status.conditions[] | select( .type == "Running" and .status == "True")'
```

The output is similar to:

```bash
{
   "lastTransitionTime": "2023-02-10T02:46:36Z",
   "lastUpdateTime": "2023-02-07T06:46:36Z",
   "message": "Cluster is running",
   "reason": "ClusterRunning",
   "status": "True",
   "type": "Running"
}
```

## Verify whether the EMQX cluster configuration is valid

- View EMQX cluster listener information

```bash
kubectl exec -it emqx-core-0 -c emqx -- emqx_ctl listeners
```

The output is similar to:

```bash
tcp:default
   listen_on: 0.0.0.0:1883
   acceptors: 16
   proxy_protocol : false
   running: true
   current_conn: 0
   max_conns : 1024000
tcp:test
   listen_on: 0.0.0.0:1884
   acceptors: 16
   proxy_protocol : false
   running: true
   current_conn: 0
   max_conns : 1024000
```

> From the output results, we can see that the listener we configured named test has taken effect.
