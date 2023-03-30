# Configure EMQX TLS certificate

## Task target 

- How to configure EMQX TLS certificate through extraVolumes and extraVolumeMounts fields.

## EMQX cluster TLS certificate configuration

- Create Secret based on TLS certificate

Secret is an object that contains a small amount of sensitive information such as passwords, tokens or keys. For its documentation, please refer to: [Secret](https://kubernetes.io/docs/concepts/configuration/secret/). In this article, we use Secret to save TLS certificate information, so we need to create Secret based on TLS certificate before creating EMQX cluster.

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

> `ca.crt` represents the content of the CA certificate, `tls.crt` represents the content of the server certificate, and `tls.key` represents the content of the server’s private key. In this example, the contents of the above three fields are omitted, please fill them with the contents of your own certificate.

Save the above file as: secret-tls.yaml, and execute the following command to create a secret:

```
kubectl apply -f secret-tls.yaml
```

The output is similar to:

```
secret/emqx-tls created
```

- Configure EMQX cluster

:::: tabs type:card
::: tab v2alpha1

EMQX CRD supports using `.spec.coreTemplate.extraVolumes` and `.spec.coreTemplate.extraVolumeMounts` and `.spec.replicantTemplate.extraVolumes` and `.spec.replicantTemplate.extraVolumeMounts` fields to configure additional volumes and mounts for the EMQX cluster point. In this article, we can use these two fields to configure TLS certificates for the EMQX cluster.

There are many types of Volumes. For the description of Volumes, please refer to the document: [Volumes](https://kubernetes.io/docs/concepts/storage/volumes/). In this article we are using the `secret` type.

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
   name: emqx
spec:
   image: emqx/emqx:5.0.14
   bootstrapConfig: |
     listeners.ssl.default {
       bind = "0.0.0.0:8883"
       ssl_options {
         cacertfile = "/mounted/cert/ca.crt"
         certfile = "/mounted/cert/tls.crt"
         keyfile = "/mounted/cert/tls.key"
       }
     }
   coreTemplate:
     spec:
       replicas: 3
       extraVolumes:
         - name: emqx-tls
           secret:
             secretName: emqx-tls
       extraVolumeMounts:
         - name: emqx-tls
           mountPath: /mounted/cert
   replicantTemplate:
     spec:
       replicas: 0
       extraVolumes:
         - name: emqx-tls
           secret:
             secretName: emqx-tls
       extraVolumeMounts:
         - name: emqx-tls
           mountPath: /mounted/cert
   dashboardServiceTemplate:
     spec:
       type: NodePort
       ports:
         - name: "dashboard-listeners-http-bind"
           protocol: TCP
           port: 18083
           targetPort: 18083
           nodePort: 32015
   listenersServiceTemplate:
     spec:
       type: NodePort
       ports:
         - name: "ssl-default"
           protocol: TCP
           port: 8883
           targetPort: 8883
           nodePort: 32016
```

> `.spec.coreTemplate.extraVolumes` field configures the volume type as: secret, and the name as: emqx-tls. `.spec.coreTemplate.extraVolumeMounts` field configures the directory where the TLS certificate is mounted to EMQX: `/mounted/cert`. The `.spec.bootstrapConfig` field configures the certificate path of the TLS listener. For more configurations of the TLS listener, please refer to the document: [ssllistener](https://www.emqx.io/docs/en/v5.0/admin/cfg.html#broker-mqtt-ssl-listener). The `.spec.listenersServiceTemplate` field configures the way the EMQX cluster exposes services to the outside world: NodePort, and specifies the nodePort corresponding to port 8883 of the EMQX ssl-default listener as 32016 (the value range of nodePort is: 30000-32767).

:::
::: tab v1beta4

EMQX CRD supports configuring volumes and mount points for EMQX clusters through `.spec.template.spec.volumes` and `.spec.template.spec.emqxContainer.volumeMounts` fields. In this article, we can use these two fields to configure TLS certificates for the EMQX cluster.

There are many types of Volumes. For the description of Volumes, please refer to the document: [Volumes](https://kubernetes.io/docs/concepts/storage/volumes/). In this article we are using the `secret` type.

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
        volumeMounts:
          - name: emqx-tls 
            mountPath: /mounted/cert
      volumes:
        - name: emqx-tls
          secret:
            secretName: emqx-tls
  serviceTemplate:
    spec:
      type: NodePort
      ports:
        - name: "mqtt-ssl-8883"
          protocol: "TCP"
          port: 8883
          targetPort: 8883
          nodePort: 32016
```

> The `.spec.template.spec.volumes` field configures the volume type as: secret, and the name as: emqx-tls. The `.spec.template.spec.emqxContainer.volumeMounts` field configures the directory where the TLS certificate is mounted to EMQX: `/mounted/cert`. The `.spec.template.spec.emqxContainer.emqxConfig` field configures the TLS listener certificate path. For more TLS listener configurations, please refer to the document: [tlsexternal](https://docs.emqx.com/en/enterprise/v4.4/configuration/configuration.html#tlsexternal). The `.spec.serviceTemplate` configuration field defines the way the EMQX cluster exposes services to the outside world: NodePort, and specifies the nodePort corresponding to port 8883 of the EMQX mqtt-ssl-8883 listener as 32016 (the value range of nodePort is: 30000-32767).

:::
::: tab v1beta3

EMQX CRD supports configuring additional volumes and mount points for the EMQX cluster through `.spec.emqxTemplate.extraVolumes` and `.spec.emqxTemplate.extraVolumeMounts` fields. In this article, we can use these two fields to configure TLS certificates for the EMQX cluster.

There are many types of Volumes. For the description of Volumes, please refer to the document: [Volumes](https://kubernetes.io/docs/concepts/storage/volumes/). In this article we are using the `secret` type.

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
   name: emqx-ee
spec:
   emqxTemplate:
     image: emqx/emqx-ee:4.4.8
     extraVolumes:
       - name: emqx-tls
         secret:
           secretName: emqx-tls
     extraVolumeMounts:
       - name: emqx-tls
         mountPath: /mounted/cert
     config:
       listener.ssl.external.cacertfile: /mounted/cert/ca.crt
       listener.ssl.external.certfile: /mounted/cert/tls.crt
       listener.ssl.external.keyfile: /mounted/cert/tls.key
       listener.ssl.external: "0.0.0.0:8883"
     serviceTemplate:
       spec:
         type: NodePort
         ports:
           - name: "mqtt-ssl-8883"
             protocol: "TCP"
             port: 8883
             targetPort: 8883
             nodePort: 32016
```

> `.spec.emqxTemplate.extraVolumes` field configures the volume type as: secret, and the name as: emqx-tls. The `.spec.emqxTemplate.extraVolumeMounts` field configures the directory where the TLS certificate is mounted to EMQX: `/mounted/cert`. The `.spec.emqxTemplate.config` field configures the certificate path of the TLS listener. For more configurations of the TLS listener, please refer to the document: [tlsexternal](https://docs.emqx.com/en/enterprise/v4.4/configuration/configuration.html#tlsexternal). The `.spec.emqxTemplate.serviceTemplate` configuration field defines the way the EMQX cluster exposes services to the outside world: NodePort, and specifies that the nodePort corresponding to port 8883 of the EMQX mqtt-ssl-8883 listener is 32016 (the value range of nodePort is: 30000-32767).

:::
::::

Save the above file as: emqx-tls.yaml, and execute the following command to deploy the EMQX cluster:

```bash
kubectl apply -f emqx-tls.yaml
```

The output is similar to:

```
emqx.apps.emqx.io/emqx created
```

- Check whether the EMQX cluster is ready

:::: tabs type:card
::: tab v2alpha1

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

:::
::: tab v1beta4

```bash
kubectl get emqxEnterprise emqx-ee -o json | jq '.status.conditions[] | select( .type == "Running" and .status == "True")'
```

The output is similar to:

```bash
{
  "lastTransitionTime": "2023-03-01T02:49:22Z",
  "lastUpdateTime": "2023-03-01T02:49:23Z",
  "message": "All resources are ready",
  "reason": "ClusterReady",
  "status": "True",
  "type": "Running"
}
```

:::
::: tab v1beta3

```bash
kubectl get emqxEnterprise emqx-ee -o json | jq '.status.conditions[] | select( .type == "Running" and .status == "True")'
```

The output is similar to:

```bash
{
  "lastTransitionTime": "2023-03-01T02:49:22Z",
  "lastUpdateTime": "2023-03-01T02:49:23Z",
  "message": "All resources are ready",
  "reason": "ClusterReady",
  "status": "True",
  "type": "Running"
}
```

:::
::::

## Verify that the TLS certificate is valid

- Use MQTT X to connect to the EMQX cluster to send messages

MQTT X is a fully open source MQTT 5.0 cross-platform desktop client. Supports quick creation of multiple simultaneous online MQTT client connections, convenient for testing MQTT/TCP, MQTT/TLS, MQTT/WebSocket connection, publishing, subscribing functions and other MQTT protocol features. For more documentation on using MQTT X, please refer to: [MQTT X](https://mqttx.app/docs). Next, we will use MQTT X to connect to the EMQX cluster to send and subscribe messages to verify whether the TLS certificate is valid.

Click the button to create a new connection on the MQTT X page, and configure the EMQX cluster node information and CA certificate path as shown in the figure. After configuring the connection information, click the connect button to connect to the EMQX cluster:

![](./assets/configure-tls/tls-connect.png)

Then click the Subscribe button to create a new subscription, as shown in the figure, MQTT X has successfully connected to the EMQX cluster and successfully created the subscription:

![](./assets/configure-tls/sub.png)

After successfully connecting to the EMQX cluster and creating a subscription, we can send messages to the EMQX cluster, as shown in the following figure:

![](./assets/configure-tls/tls-test.png)

As can be seen from the above figure, if the subscriber can normally receive the MQTT message sent by the client, it means that the TLS we configured is valid.
