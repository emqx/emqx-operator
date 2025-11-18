# Change EMQX Log Level

## Objective

Modify the log level in the EMQX cluster.

## Configure EMQX Cluster

EMQX CRD `apps.emqx.io/v2` supports configuring the log level of the EMQX cluster through `.spec.config.data`. Refer to the [Configuration Manual](https://docs.emqx.com/en/enterprise/v6.0.0/hocon/) for complete configuration reference.

+ Save the following content as a YAML file and deploy it using `kubectl apply`:

  ```yaml
  apiVersion: apps.emqx.io/v2
  kind: EMQX
  metadata:
    name: emqx
  spec:
    image: emqx/emqx:6.0
    config:
      # Enable debug logging:
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

+ Wait for the EMQX cluster to become ready.

  Check the status of the EMQX cluster with `kubectl get` and ensure that `STATUS` is `Ready`. This may take some time.

  ```bash
  $ kubectl get emqx
  NAME   STATUS   AGE
  emqx   Ready    10m
  ```

## Verify Log Level

+ Obtain the External IP of the EMQX cluster.

  ```bash
  external_ip=$(kubectl get svc emqx-listeners -o json | jq '.status.loadBalancer.ingress[0].ip')
  ```

+ Use MQTTX CLI to connect to the EMQX cluster.

  [MQTTX CLI](https://mqttx.app/cli) is an open source MQTT 5.0 command line client tool, designed to help developers start using MQTT services and applications more quickly.

  ```bash
  $ mqttx conn -h ${external_ip} -p 1883
  [4/17/2023] [5:17:31 PM] › … Connecting...
  [4/17/2023] [5:17:31 PM] › ✔ Connected
  ```

+ View EMQX container logs.

  ```bash
  $ kubectl logs emqx-core-0 -c emqx
  ...
  2023-04-17T09:11:35.993031+00:00 [debug] msg: mqtt_packet_received, mfa: emqx_channel:handle_in/2, line: 360, peername: 218.190.230.144:59457, clientid: mqttx_322680d9, packet: CONNECT(Q0, R0, D0, ClientId=mqttx_322680d9, ProtoName=MQTT, ProtoVsn=5, CleanStart=true, KeepAlive=30, Username=undefined, Password=), tag: MQTT
  2023-04-17T09:11:35.997066+00:00 [debug] msg: mqtt_packet_sent, mfa: emqx_connection:serialize_and_inc_stats_fun/1, line: 872, peername: 218.190.230.144:59457, clientid: mqttx_322680d9, packet: CONNACK(Q0, R0, D0, AckFlags=0, ReasonCode=0), tag: MQTT
  ```
