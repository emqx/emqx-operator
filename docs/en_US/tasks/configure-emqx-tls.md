# Enable TLS In EMQX

## Task Target

Customize TLS certificates via the `extraVolumes` and `extraVolumeMounts` fields.

## Create Secret Based On TLS Certificate

Secret is an object that contains a small amount of sensitive information such as passwords, tokens, or keys. For its documentation, please refer to: [Secret](https://kubernetes.io/docs/concepts/configuration/secret/#working-with-secrets). In this article, we use Secret to save TLS certificate information, so we need to create Secret based on TLS certificate before creating EMQX cluster.

+ Save the following as a YAML file and deploy it with the `kubectl apply` command

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

  > `ca.crt` indicates the content of the CA certificate, `tls.crt` indicates the content of the server certificate, and `tls.key` indicates the content of the server private key. In this example, the contents of the above three fields are omitted, please fill them with the contents of your own certificate.

## Configure EMQX Cluster

The following is the relevant configuration of EMQX Custom Resource. You can choose the corresponding APIVersion according to the version of EMQX you want to deploy. For the specific compatibility relationship, please refer to [EMQX Operator Compatibility](../index.md):

:::: tabs type:card
::: tab apps.emqx.io/v2beta1

`apps.emqx.io/v2beta1 EMQX` supports `.spec.coreTemplate.extraVolumes` and `.spec.coreTemplate.extraVolumeMounts` and `.spec.replicantTemplate.extraVolumes` and `.spec.replicantTemplate.extraVolumeMounts` fields to EMQX The cluster configures additional volumes and mount points. In this article, we can use these two fields to configure TLS certificates for the EMQX cluster.

There are many types of Volumes. For the description of Volumes, please refer to the document: [Volumes](https://kubernetes.io/docs/concepts/storage/volumes/#secret). In this article we are using the `secret` type.

+ Save the following as a YAML file and deploy it with the `kubectl apply` command

  ```yaml
  apiVersion: apps.emqx.io/v2beta1
  kind: EMQX
  metadata:
    name: emqx
  spec:
    image: emqx:5.1
    config:
      data: |
        listeners.ssl.default {
          bind = "0.0.0.0:8883"
          ssl_options {
            cacertfile = "/mounted/cert/ca.crt"
            certfile = "/mounted/cert/tls.crt"
            keyfile = "/mounted/cert/tls.key"
            gc_after_handshake = true
            hibernate_after = 5s
          }
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
          - name: emqx-tls
            secret:
              secretName: emqx-tls
        extraVolumeMounts:
          - name: emqx-tls
            mountPath: /mounted/cert
    dashboardServiceTemplate:
      spec:
        type: LoadBalancer
    listenersServiceTemplate:
      spec:
        type: LoadBalancer
  ```

  > The `.spec.coreTemplate.extraVolumes` field configures the volume type as: secret, and the name as: emqx-tls.

  > The `.spec.coreTemplate.extraVolumeMounts` field configures the directory where the TLS certificate is mounted to EMQX: `/mounted/cert`.

  > The `.spec.config.data` field configures the TLS listener certificate path. For more TLS listener configurations, please refer to the document: [Configuration Manual](https://www.emqx.io/docs/en/v5.1/configuration/configuration-manual.html#configuration-manual).

+ Wait for EMQX cluster to be ready, you can check the status of EMQX cluster through the `kubectl get` command, please make sure that `STATUS` is `Running`, this may take some time

  ```bash
  $ kubectl get emqx

  NAME   IMAGE      STATUS    AGE
  emqx   emqx:5.1   Running   10m
  ```

+ Obtain the External IP of EMQX cluster and access EMQX console

  EMQX Operator will create two EMQX Service resources, one is emqx-dashboard and the other is emqx-listeners, corresponding to EMQX console and EMQX listening port respectively.

   ```bash
   $ kubectl get svc emqx-dashboard -o json | jq '.status.loadBalancer.ingress[0].ip'

   192.168.1.200
   ```

   Access `http://192.168.1.200:18083` through a browser, and use the default username and password `admin/public` to login EMQX console.

:::
::: tab apps.emqx.io/v1beta4

`apps.emqx.io/v1beta4 EmqxEnterprise` supports configuring volumes and mount points for EMQX clusters through `.spec.template.spec.volumes` and `.spec.template.spec.emqxContainer.volumeMounts` fields. In this article, we can use these two fields to configure TLS certificates for the EMQX cluster.

There are many types of Volumes. For the description of Volumes, please refer to the document: [Volumes](https://kubernetes.io/docs/concepts/storage/volumes/). In this article we are using the `secret` type.

+ Save the following as a YAML file and deploy it with the `kubectl apply` command

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
            version: 4.4.14
          emqxConfig:
            listener.ssl.external.cacertfile: /mounted/cert/ca.crt
            listener.ssl.external.certfile: /mounted/cert/tls.crt
            listener.ssl.external.keyfile: /mounted/cert/tls.key
            listener.ssl.external: "0.0.0.0:8883"
            listener.ssl.external.gc_after_handshake: "true"
            listener.ssl.external.hibernate_after: 5s
          volumeMounts:
            - name: emqx-tls
              mountPath: /mounted/cert
        volumes:
          - name: emqx-tls
            secret:
              secretName: emqx-tls
  serviceTemplate:
    spec:
      type: LoadBalancer
  ```

  > The `.spec.template.spec.volumes` field configures the volume type as: secret, and the name as: emqx-tls.

  > The `.spec.template.spec.emqxContainer.volumeMounts` field configures the directory where the TLS certificate is mounted to EMQX: `/mounted/cert`.

  > The `.spec.template.spec.emqxContainer.emqxConfig` field configures the TLS listener certificate path. For more TLS listener configurations, please refer to the document: [tlsexternal](https://docs.emqx.com/en/enterprise/v4.4/configuration/configuration.html#tlsexternal).


+ Wait for EMQX cluster to be ready, you can check the status of EMQX cluster through the `kubectl get` command, please make sure that `STATUS` is `Running`, this may take some time

  ```bash
  $ kubectl get emqxenterprises
  NAME      STATUS   AGE
  emqx-ee   Running  8m33s
  ```

+ Obtain the External IP of EMQX cluster and access EMQX console

  ```bash
  $ kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip'

  192.168.1.200
  ```

  Access `http://192.168.1.200:18083` through a browser, and use the default username and password `admin/public` to login EMQX console.


:::
::::

## Verify TLS Connection Using MQTT X CLI

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

+ Subscribe to messages using MQTT X CLI

  ```bash
  mqttx sub -h ${external_ip} -p 8883 -t "hello" -l mqtts --insecure

  [10:00:25] › … Connecting...
  [10:00:25] › ✔ Connected
  [10:00:25] › … Subscribing to hello...
  [10:00:25] › ✔ Subscribed to hello
  ```

+ Create a new terminal window and publish a message using the MQTT X CLI

  ```bash
  mqttx pub -h ${external_ip} -p 8883 -t "hello" -m "hello world" -l mqtts --insecure

  [10:00:58] › … Connecting...
  [10:00:58] › ✔ Connected
  [10:00:58] › … Message Publishing...
  [10:00:58] › ✔ Message published
  ```

+ View messages received in the subscribed terminal window

  ```bash
  [10:00:58] › payload: hello world
  ```
