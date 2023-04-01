# Change EMQX Configuration In EMQX Custom Resource

## Task target

How to change EMQX configuration by `.spec.bootstrapConfig` in EMQX Custom Resource.

## Configure EMQX cluster

The main configuration file of EMQX is `emqx.conf`. Starting from version 5.0, EMQX adopts [HOCON](https://www.emqx.io/docs/en/v5.0/configuration/configuration.html#hocon-configuration-format) as the configuration file format.

EMQX CRD supports using the `.spec.bootstrapConfig` field to configure the EMQX cluster. For bootstrapConfig configuration, please refer to the document: [bootstrapConfig](https://www.emqx.io/docs/en/v5.0/admin/cfg.html). This field is only allowed to be configured when creating an EMQX cluster and does not support updating. **Note:** If you need to modify the cluster configuration after creating EMQX, please modify it through EMQX Dashboard.

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

Save the above content as `emqx.yaml` and execute the following command to deploy the EMQX cluster:

```bash
$ kubectl apply -f emqx.yaml

emqx.apps.emqx.io/emqx created
```

Check the status of the EMQX cluster and make sure that `STATUS` is `Running`, which may take some time to wait for the EMQX cluster to be ready.

```bash
$ kubectl get emqx emqx

NAME   IMAGE      STATUS    AGE
emqx   emqx:5.0   Running   10m
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

From the output results, we can see that the listener we configured named test has taken effect.
