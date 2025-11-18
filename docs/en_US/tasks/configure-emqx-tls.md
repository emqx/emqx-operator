# Enable TLS In EMQX

## Objective

Customize TLS certificates using the `extraVolumes` and `extraVolumeMounts` fields.

## Create Secret Based On TLS Certificate

Secret is an object that contains a small amount of sensitive information such as passwords, tokens, or keys. In this article, we use secrets to store TLS certificate information, so we need to create one before creating the EMQX cluster.

For more information, please refer to the [Secret](https://kubernetes.io/docs/concepts/configuration/secret/#working-with-secrets) documentation.

- Save the following as a YAML file and deploy it using the `kubectl apply` command:

  ```yaml
  apiVersion: v1
  kind: Secret
  metadata:
    name: emqx-tls
  type: kubernetes.io/tls
  stringData:
    ca.crt: |
      -----BEGIN CERTIFICATE-----
      ...
      -----END CERTIFICATE-----
    tls.crt: |
      -----BEGIN CERTIFICATE-----
      ...
      -----END CERTIFICATE-----
    tls.key: |
      -----BEGIN RSA PRIVATE KEY-----
      ...
      -----END RSA PRIVATE KEY-----
  ```

  :::tip
  In this example, the contents of the above three fields are omitted. Please fill them with your own certificate contents.
  * `ca.crt` should contain the CA certificate.
  * `tls.crt` should contain the server certificate.
  * `tls.key` should contain the server private key.
  :::

## Configure EMQX Cluster

The EMQX CRD `apps.emqx.io/v2` provides the following fields to configure additional volumes and mount points for the EMQX cluster:
* `.spec.coreTemplate.extraVolumes`
* `.spec.coreTemplate.extraVolumeMounts`
* `.spec.replicantTemplate.extraVolumes`
* `.spec.replicantTemplate.extraVolumeMounts`

In this article, we will use these fields to provide TLS certificates to the EMQX cluster.

There are many types of Volumes. For information about Volumes, please refer to the [Volumes](https://kubernetes.io/docs/concepts/storage/volumes/#secret) documentation. Here we are using the `secret` volume type.

- Save the following as a YAML file and deploy it using `kubectl apply`:

  ```yaml
  apiVersion: apps.emqx.io/v2
  kind: EMQX
  metadata:
    name: emqx
  spec:
    image: emqx/emqx:6.0
    config:
      # Configure the TLS listener certificates mounted from the `emqx-tls` volume:
      data: |
        listeners.ssl.default {
          bind = "0.0.0.0:8883"
          ssl_options {
            cacertfile = "/mounted/cert/ca.crt"
            certfile = "/mounted/cert/tls.crt"
            keyfile = "/mounted/cert/tls.key"
            gc_after_handshake = true
            handshake_timeout = 5s
          }
        }
        license {
          key = "..."
        }
    coreTemplate:
      spec:
        extraVolumes:
          - name: emqx-tls
            secret:
              secretName: emqx-tls
        extraVolumeMounts:
          - name: emqx-tls
            mountPath: /mounted/cert
    replicantTemplate:
      spec:
        extraVolumes:
          # Create a `secret` volume type named `emqx-tls`:
          - name: emqx-tls
            secret:
              secretName: emqx-tls
        extraVolumeMounts:
          - name: emqx-tls
            # Directory where the TLS certificate is mounted to EMQX nodes:
            mountPath: /mounted/cert
    dashboardServiceTemplate:
      spec:
        type: LoadBalancer
    listenersServiceTemplate:
      spec:
        type: LoadBalancer
  ```

- Wait for the EMQX cluster to become ready.

  Check the status of the EMQX cluster using `kubectl get`, and make sure that `STATUS` is `Ready`. This may take a while.

  ```bash
  $ kubectl get emqx
  NAME   STATUS   AGE
  emqx   Ready    10m
  ```

## Verify TLS Connection using MQTTX

[MQTTX CLI](https://mqttx.app/cli) is an open-source MQTT 5.0 command-line client tool, designed to help developers quickly get started with MQTT services and applications.

- Obtain the external IP of the EMQX listeners service.

  ```bash
  external_ip=$(kubectl get svc emqx-listeners -o json | jq '.status.loadBalancer.ingress[0].ip')
  ```

- Subscribe to messages using MQTTX CLI.

  Connect to the TLS listener port 8883, using the `--insecure` flag to skip certificate verification.

  ```bash
  mqttx sub -h ${external_ip} -p 8883 -t "hello" -l mqtts --insecure
  [10:00:25] › … Connecting...
  [10:00:25] › ✔ Connected
  [10:00:25] › … Subscribing to hello...
  [10:00:25] › ✔ Subscribed to hello
  ```

- In a separate terminal window, publish a message.

  ```bash
  mqttx pub -h ${external_ip} -p 8883 -t "hello" -m "hello world" -l mqtts --insecure
  [10:00:58] › … Connecting...
  [10:00:58] › ✔ Connected
  [10:00:58] › … Message Publishing...
  [10:00:58] › ✔ Message published
  ```

- Observe the subscriber client receiving the message.

  This indicates that both the publisher and subscriber clients successfully communicate with the broker over a TLS connection.

  ```bash
  [10:00:58] › payload: hello world
  ```
