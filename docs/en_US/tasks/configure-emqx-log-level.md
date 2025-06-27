# Change EMQX Log Level

## Task Target

Modify the log level of EMQX cluster.

## Configure EMQX Cluster

The following is the relevant configuration of EMQX Custom Resource. You can choose the corresponding APIVersion according to the version of EMQX you want to deploy. For the specific compatibility relationship, please refer to [EMQX Operator Compatibility](../index.md):

:::: tabs type:card
::: tab apps.emqx.io/v2beta1

`apps.emqx.io/v2beta1 EMQX` supports configuration of EMQX cluster log level through `.spec.config.data`. The configuration of config.data can refer to the document: [Configuration Manual](https://www.emqx.io/docs/en/v5.1/configuration/configuration-manual.html#configuration-manual).

> This field is only allowed to be configured when creating an EMQX cluster, and does not support updating. If you need to modify the cluster log level after creating EMQX, please modify it through EMQX Dashboard.

+ Save the following content as a YAML file and deploy it with the kubectl apply command

  ```yaml
  apiVersion: apps.emqx.io/v2beta1
  kind: EMQX
  metadata:
    name: emqx
  spec:
    image: emqx/emqx-enterprise:5.10
    config:
      data: |
        log.console.level = debug
        license {
          key = "..."
        }
    dashboardServiceTemplate:
      spec:
        type: LoadBalancer
    listenersServiceTemplate:
      spec:
        type: LoadBalancer
  ```

  > The `.spec.config.data` field configures the EMQX cluster log level to `debug`.

+ Wait for the EMQX cluster to be ready, you can check the status of the EMQX cluster through the kubectl get command, please make sure that `STATUS` is Running, this may take some time

  ```bash
  $ kubectl get emqx
  NAME   IMAGE                         STATUS    AGE
  emqx   emqx/emqx-enterprise:5.10.0   Running   10m
  ```

+ EMQX Operator will create two EMQX Service resources, one is emqx-dashboard and the other is emqx-listeners, corresponding to EMQX console and EMQX listening port respectively.

  ```bash
  $ kubectl get svc emqx-dashboard -o json | jq '.status.loadBalancer.ingress[0].ip'

  192.168.1.200
  ```

  Access `http://192.168.1.200:18083` through a browser, and use the default username and password `admin/public` to login EMQX console.

:::
::: tab apps.emqx.io/v1beta4

`apps.emqx.io/v1beta4 EmqxEnterprise` supports configuring the log level of EMQX cluster through `.spec.template.spec.emqxContainer.emqxConfig` field. For the specific description of the emqxConfig field, please refer to: [emqxConfig](../reference/v1beta4-reference.md#emqxtemplatespec).

+ Save the following content as a YAML file and deploy it with the `kubectl apply` command

  ```yaml
  apiVersion: apps.emqx.io/v1beta4
  kind: EmqxEnterprise
  metadata:
    name: emqx-ee
  spec:
    template:
      spec:
        emqxContainer:
          image:
            repository: emqx/emqx-ee
            version: 4.4.30
          emqxConfig:
            log.level: debug
    serviceTemplate:
      spec:
        type: LoadBalancer
  ```

  > The `.spec.template.spec.emqxContainer.emqxConfig` field configures the EMQX cluster log level to `debug`.

+ Wait for the EMQX cluster to be ready, you can check the status of the EMQX cluster through the `kubectl get` command, please make sure that `STATUS` is `Running`, this may take some time

  ```bash
  $ kubectl get emqxenterprises
  NAME      STATUS   AGE
  emqx-ee   Running  8m33s
  ```

+ Obtain the External IP of EMQX cluster and access EMQX console

  ```bash
  $ kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip'

  198.18.3.10
  ```
  Access `http://192.168.1.200:18083` through a browser, and use the default username and password `admin/public` to login EMQX console.

:::
::::

## Verify Log Level

[MQTT X CLI](https://mqttx.app/cli) is an open source MQTT 5.0 command line client tool, designed to help developers to more Quickly develop and debug MQTT services and applications.

+ Obtain the External IP of EMQX cluster

  :::: tabs type:card
  ::: tab apps.emqx.io/v2beta1

  ```bash
  external_ip=$(kubectl get svc emqx-listeners -o json | jq '.status.loadBalancer.ingress[0].ip')
  ```
  :::
  ::: tab apps.emqx.io/v1beta4

  ```bash
  external_ip=$(kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip')
  ```
  :::
  ::::

+ Use MQTT X CLI to connect to EMQX cluster

  ```bash
  $ mqttx conn -h ${external_ip} -p 1883

  [4/17/2023] [5:17:31 PM] › … Connecting...
  [4/17/2023] [5:17:31 PM] › ✔ Connected
  ```

+ Use the command line to view EMQX cluster log information

  ```bash
  $ kubectl logs emqx-core-0 -c emqx
  ```

  You can get a print similar to the following, which means that EMQX has received a CONNECT message from the client and replied a CONNACK message to the client:

  ```bash
  2023-04-17T09:11:35.993031+00:00 [debug] msg: mqtt_packet_received, mfa: emqx_channel:handle_in/2, line: 360, peername: 218.190.230.144:59457, clientid: mqttx_322680d9, packet: CONNECT(Q0, R0, D0, ClientId=mqttx_322680d9, ProtoName=MQTT, ProtoVsn=5, CleanStart=true, KeepAlive=30, Username=undefined, Password=), tag: MQTT
  2023-04-17T09:11:35.997066+00:00 [debug] msg: mqtt_packet_sent, mfa: emqx_connection:serialize_and_inc_stats_fun/1, line: 872, peername: 218.190.230.144:59457, clientid: mqttx_322680d9, packet: CONNACK(Q0, R0, D0, AckFlags=0, ReasonCode=0), tag: MQTT
  ```
