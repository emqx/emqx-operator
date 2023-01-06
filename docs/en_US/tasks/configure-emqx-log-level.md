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
   image: emqx/emqx:5.0.9
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

**NOTE:** The `.spec.bootstrapConfig` field configures the EMQX cluster log level to `debug`.

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
           version: 4.4.8
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

**NOTE:** The `.spec.template.spec.emqxContainer.emqxConfig` field configures the EMQX cluster log level to `debug`.

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
     image: emqx/emqx-ee:4.4.8
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

**NOTE:** The `.spec.emqxTemplate.config` field configures the log level of the EMQX cluster to `debug`.

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
kubectl get emqx emqx -o json | jq ".status.emqxNodes"
```

The output is similar to:

```
[
   {
     "node": "emqx@emqx-core-0.emqx-headless.default.svc.cluster.local",
     "node_status": "running",
     "otp_release": "24.2.1-1/12.2.1",
     "role": "core",
     "version": "5.0.9"
   },
   {
     "node": "emqx@emqx-core-1.emqx-headless.default.svc.cluster.local",
     "node_status": "running",
     "otp_release": "24.2.1-1/12.2.1",
     "role": "core",
     "version": "5.0.9"
   },
   {
     "node": "emqx@emqx-core-2.emqx-headless.default.svc.cluster.local",
     "node_status": "running",
     "otp_release": "24.2.1-1/12.2.1",
     "role": "core",
     "version": "5.0.9"
   }
]
```

**NOTE:** node represents the unique identifier of the EMQX node in the cluster. node_status indicates the status of EMQX nodes. otp_release indicates the version of Erlang used by EMQX. role represents the EMQX node role type. version indicates the EMQX version. EMQX Operator creates an EMQX cluster with three core nodes and three replicant nodes by default, so when the cluster is running normally, you can see information about three running core nodes and three replicant nodes. If you configure the `.spec.coreTemplate.spec.replicas` field, when the cluster is running normally, the number of running core nodes displayed in the output should be equal to the value of this replicas. If you configure the `.spec.replicantTemplate.spec.replicas` field, when the cluster is running normally, the number of running replicant nodes displayed in the output should be equal to the replicas value.

:::
::: tab v1beta4

```bash
kubectl get emqxenterprise emqx-ee -o json | jq ".status.emqxNodes"
```
The output is similar to:

```
[
   {
     "node": "emqx-ee@emqx-ee-0.emqx-ee-headless.default.svc.cluster.local",
     "node_status": "Running",
     "otp_release": "24.1.5/12.1.5",
     "version": "4.4.8"
   },
   {
     "node": "emqx-ee@emqx-ee-1.emqx-ee-headless.default.svc.cluster.local",
     "node_status": "Running",
     "otp_release": "24.1.5/12.1.5",
     "version": "4.4.8"
   },
   {
     "node": "emqx-ee@emqx-ee-2.emqx-ee-headless.default.svc.cluster.local",
     "node_status": "Running",
     "otp_release": "24.1.5/12.1.5",
     "version": "4.4.8"
   }
]
```

**NOTE:** node represents the unique identifier of the EMQX node in the cluster. node_status indicates the status of EMQX nodes. otp_release indicates the version of Erlang used by EMQX. version indicates the EMQX version. EMQX Operator will pull up the EMQX cluster with three nodes by default, so when the cluster is running normally, you can see the information of the three running nodes. If you configure the `.spec.replicas` field, when the cluster is running normally, the number of running nodes displayed in the output should be equal to the value of replicas.

:::
::: tab v1beta3

```bash
kubectl get emqxenterprise emqx-ee -o json | jq ".status.emqxNodes"
```

The output is similar to:

```
[
   {
     "node": "emqx-ee@emqx-ee-0.emqx-ee-headless.default.svc.cluster.local",
     "node_status": "Running",
     "otp_release": "24.1.5/12.1.5",
     "version": "4.4.8"
   },
   {
     "node": "emqx-ee@emqx-ee-1.emqx-ee-headless.default.svc.cluster.local",
     "node_status": "Running",
     "otp_release": "24.1.5/12.1.5",
     "version": "4.4.8"
   },
   {
     "node": "emqx-ee@emqx-ee-2.emqx-ee-headless.default.svc.cluster.local",
     "node_status": "Running",
     "otp_release": "24.1.5/12.1.5",
     "version": "4.4.8"
   }
]
```

**NOTE:** node represents the unique identifier of the EMQX node in the cluster. node_status indicates the status of EMQX nodes. otp_release indicates the version of Erlang used by EMQX. version indicates the EMQX version. EMQX Operator will pull up the EMQX cluster with three nodes by default, so when the cluster is running normally, you can see the information of the three running nodes. If you configure the `.spec.replicas` field, when the cluster is running normally, the number of running nodes displayed in the output should be equal to the value of replicas.

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