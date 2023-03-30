# Configure EMQX log level

## Task target

- How to configure the log level of EMQX cluster.

## Configure EMQX cluster

:::: tabs type:card
::: tab v2alpha1

EMQX CRD supports the use of `.spec.bootstrapConfig` to configure the log level of the EMQX cluster. The configuration of bootstrapConfig can refer to the document: [bootstrapConfig](https://www.emqx.io/docs/en/v5.0/admin/cfg.html). This field is only allowed to be configured when creating an EMQX cluster, and does not support updating. **NOTE:** If you need to modify the cluster log level after creating EMQX, please modify it through EMQX Dashboard.

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
   name: emqx
spec:
   image: emqx/emqx:5.0.14
   imagePullPolicy: IfNotPresent
   bootstrapConfig: |
     log {
        console_handler {
           level = debug
         }
     }
   coreTemplate:
     spec:
       replicas: 3
   replicantTemplate:
     spec:
       replicas: 0
   listenersServiceTemplate:
     spec:
       type: NodePort
       ports:
         - name: "tcp-default"
           protocol: TCP
           port: 1883
           targetPort: 1883
           nodePort: 32010
```

> The `.spec.bootstrapConfig` field configures the EMQX cluster log level to `debug`.

:::
::: tab v1beta4

The corresponding CRD of EMQX Enterprise Edition in EMQX Operator is EmqxEnterprise, and EmqxEnterprise supports configuring the log level of EMQX cluster through `.spec.template.spec.emqxContainer.emqxConfig` field. For the specific description of the emqxConfig field, please refer to: [emqxConfig](https://github.com/emqx/emqx-operator/blob/main-2.1/docs/en_US/reference/v1beta4-reference.md#emqxtemplatespec).

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
           log.level: debug
   serviceTemplate:
     spec:
       type: NodePort
       ports:
         - name: "mqtt-tcp-1883"
           protocol: "TCP"
           port: 1883
           targetPort: 1883
           nodePort: 32010
```

> The `.spec.template.spec.emqxContainer.emqxConfig` field configures the EMQX cluster log level to `debug`.

:::
::: tab v1beta3

The corresponding CRD of EMQX Enterprise Edition in EMQX Operator is EmqxEnterprise, and EmqxEnterprise supports configuring the cluster log level through `.spec.emqxTemplate.config` field. The description of the config field can refer to the document: [config](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/reference/v1beta3-reference.md#emqxenterprisetemplate)

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
   name: emqx-ee
spec:
   emqxTemplate:
     image: emqx/emqx-ee:4.4.14
     config:
       log.level: debug
     serviceTemplate:
       spec:
         type: NodePort
         ports:
           - name: "mqtt-tcp-1883"
             protocol: "TCP"
             port: 1883
             targetPort: 1883
             nodePort: 32010
```

> The `.spec.emqxTemplate.config` field configures the log level of the EMQX cluster to `debug`.

:::
::::

Save the above content as: emqx-log-level.yaml, and execute the following command to deploy the EMQX cluster:

```bash
kubectl apply -f emqx-log-level.yaml
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

## Verify whether the EMQX cluster log level configuration is effective

- Use MQTT X to connect to the EMQX cluster to send messages

Click the button to create a new connection on the MQTT X page, and configure the EMQX cluster node information as shown in the figure. After configuring the connection information, click the connect button to connect to the EMQX cluster:

![](./assets/configure-log-level/mqtt-connected.png)

Then click the Subscribe button to create a new subscription, as shown in the figure, MQTT X has successfully connected to the EMQX cluster and successfully created the subscription:

![](./assets/configure-log-level/mqtt-sub.png)

After successfully connecting to the EMQX cluster and creating a subscription, we can send messages to the EMQX cluster, as shown in the following figure:

![](./assets/configure-log-level/mqtt-pub.png)

- Use the command line to view EMQX cluster log information

```bash
kubectl logs emqx-core-0 -c emqx
```

The output is shown in the figure below:

![](./assets/configure-log-level/emqx-debug-log.png)

From the figure, you can see the debug log information of connecting to the EMQX cluster using MQTT just now and sending messages.
