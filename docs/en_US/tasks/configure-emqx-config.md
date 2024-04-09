# Change EMQX Configurations

## Task Target

Change EMQX configuration by `config.data` in EMQX Custom Resource.

## Configure EMQX Cluster

The main configuration file of EMQX is `/etc/emqx.conf`. Starting from version 5.0, EMQX adopts [HOCON](https://www.emqx.io/docs/en/v5.1/configuration/configuration.html#hocon-configuration-format) as the configuration file format.

`apps.emqx.io/v2beta1 EMQX` supports configuring EMQX cluster through `.spec.config.data` field. For config.data configuration, please refer to the document: [Configuration Manual](https://www.emqx.io/docs/en/v5.1/configuration/configuration-manual.html#configuration-manual).

+ Save the following content as a YAML file and deploy it with the `kubectl apply` command

   ```yaml
   apiVersion: apps.emqx.io/v2beta1
   kind: EMQX
   metadata:
      name: emqx
   spec:
      image: emqx:5
      imagePullPolicy: IfNotPresent
      config:
         data: |
            listeners.tcp.test {
               bind = "0.0.0.0:1884"
               max_connections = 1024000
            }
      listenersServiceTemplate:
         spec:
            type: LoadBalancer
      dashboardServiceTemplate:
         spec:
            type: LoadBalancer
   ```

   > In the `.spec.config.data` field, we have configured a TCP listener for the EMQX cluster. The name of this listener is: test, and the listening port is: 1884.

+ Wait for the EMQX cluster to be ready, you can check the status of EMQX cluster through `kubectl get` command, please make sure `STATUS` is `Running`, this may take some time

   ```bash
   $ kubectl get emqx emqx
   NAME   IMAGE      STATUS    AGE
   emqx   emqx:5.1   Running   10m
   ```

+ Obtain the Dashboard External IP of EMQX cluster and access EMQX console

  EMQX Operator will create two EMQX Service resources, one is emqx-dashboard and the other is emqx-listeners, corresponding to EMQX console and EMQX listening port respectively.

  ```bash
  $ kubectl get svc emqx-dashboard -o json | jq '.status.loadBalancer.ingress[0].ip'

  192.168.1.200
  ```

  Access `http://192.168.1.200:18083` through a browser, and use the default username and password `admin/public` to login EMQX console.

## Verify Configuration

+ View EMQX cluster listener information

   ```bash
   $ kubectl exec -it emqx-core-0 -c emqx -- emqx_ctl listeners
   ```

   You can get a print similar to the following, which means that the listener named `test` configured by us has taken effect.

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
