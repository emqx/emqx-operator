# Change EMQX Configuration

## Objective

Change the EMQX configuration using the `.spec.config.data` field in the EMQX Custom Resource.

## Configure EMQX Cluster

The EMQX CRD `apps.emqx.io/v2` supports configuring the EMQX cluster through the `.spec.config.data` field. Refer to the [Configuration Manual](https://docs.emqx.com/en/enterprise/v6.0.0/hocon/) for the complete configuration reference.

EMQX uses [HOCON](https://docs.emqx.com/en/emqx/latest/configuration/configuration.html#hocon-configuration-format) as the configuration file format.

- Save the following as a YAML file and deploy it using `kubectl apply`:

   ```yaml
   apiVersion: apps.emqx.io/v2
   kind: EMQX
   metadata:
      name: emqx
   spec:
      image: emqx/emqx:6.0
      imagePullPolicy: IfNotPresent
      config:
         # Configure a TCP listener named `test` listening on port 1884:
         data: |
            listeners.tcp.test {
               bind = "0.0.0.0:1884"
               max_connections = 1024000
            }
            license {
              key = "..."
            }
      listenersServiceTemplate:
         spec:
            type: LoadBalancer
      dashboardServiceTemplate:
         spec:
            type: LoadBalancer
   ```

   ::: tip
   The content of the `.spec.config.data` field is supplied as [`base.hocon` configuration file](https://docs.emqx.com/en/emqx/latest/configuration/configuration.html#base-configuration-file) to the EMQX container.
   :::

- Wait for the EMQX cluster to become ready.

  Check the status of the EMQX cluster using `kubectl get`, and make sure that `STATUS` is `Ready`. This may take some time.

   ```bash
   $ kubectl get emqx emqx
   NAME   STATUS   AGE
   emqx   Ready    10m
   ```

## Verify Configuration

- View the EMQX listeners status.

   ```bash
   $ kubectl exec -it emqx-core-0 -c emqx -- emqx ctl listeners
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

   Here we can see that the new listener on port 1884 is running.
