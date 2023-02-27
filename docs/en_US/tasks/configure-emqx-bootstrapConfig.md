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

**NOTE:** In the `.spec.bootstrapConfig` field, we have configured a TCP listener for the EMQX cluster. The name of this listener is: test, and the listening port is: 1884.

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
kubectl get emqx emqx -o json | jq ".status.emqxNodes"
```

The output is similar to:

```
[
   {
     "node": "emqx@emqx-core-0.emqx-headless.default.svc.cluster.local",
     "node_status": "running",
     "otp_release": "24.3.4.2-1/12.3.2.2",
     "role": "core",
     "version": "5.0.14"
   },
   {
     "node": "emqx@emqx-core-1.emqx-headless.default.svc.cluster.local",
     "node_status": "running",
     "otp_release": "24.3.4.2-1/12.3.2.2",
     "role": "core",
     "version": "5.0.14"
   },
   {
     "node": "emqx@emqx-core-2.emqx-headless.default.svc.cluster.local",
     "node_status": "running",
     "otp_release": "24.3.4.2-1/12.3.2.2",
     "role": "core",
     "version": "5.0.14"
   }
]
```

**NOTE:** node represents the unique identifier of the EMQX node in the cluster. node_status indicates the status of EMQX nodes. otp_release indicates the version of Erlang used by EMQX. role represents the EMQX node role type. version indicates the EMQX version. EMQX Operator creates an EMQX cluster with three core nodes and three replicant nodes by default, so when the cluster is running normally, you can see information about three running core nodes and three replicant nodes. If you configure the `.spec.coreTemplate.spec.replicas` field, when the cluster is running normally, the number of running core nodes displayed in the output should be equal to the value of this replicas. If you configure the `.spec.replicantTemplate.spec.replicas` field, when the cluster is running normally, the number of running replicant nodes displayed in the output should be equal to the replicas value.


## Verify whether the EMQX cluster configuration is valid

- View EMQX cluster listener information

```
kubectl exec -it emqx-core-0 -c emqx -- emqx_ctl listeners
```

The output is similar to:

```
tcp: default
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

**NOTE**: From the output results, we can see that the listener we configured named test has taken effect.