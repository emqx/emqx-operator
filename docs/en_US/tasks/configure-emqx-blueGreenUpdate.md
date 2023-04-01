# Configure Blue-Green Upgrade (EMQX Enterprise)

## Task target

How to use the blueGreenUpdate field to configure the blue-green upgrade of EMQX Enterprise.

## Why Need Blue-Green Upgrade

EMQX provides a long-term connection service. In Kubernetes, the existing upgrade strategy requires restarting the EMQX service except for hot upgrade. This upgrade strategy will cause disconnection of the device. If the device has a reconnection mechanism, a large number of devices will appear Simultaneously requesting connections, which triggers an avalanche, eventually causing a large number of clients to be temporarily unserviced. Therefore, EMQX Operator implements a blue-green upgrade based on the Node Evacuation function of EMQX Enterprise to solve the above problems.

The EMQX node evacuation function is used to evacuate all connections in the node, and manually/automatically move client connections and sessions to other nodes in the cluster or other clusters. For a detailed introduction to EMQX node evacuation, please refer to the document: [Node Evacuation](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#evacuation).

:::tip

The node evacuation function is only available in EMQX Enterprise 4.4.12.

:::

## How to Use Blue-Green Upgrade

The corresponding CRD of EMQX Enterprise in EMQX Operator is EmqxEnterprise. EmqxEnterprise supports configuring the blue-green upgrade of EMQX Enterprise through the `.spec.blueGreenUpdate` field. For the specific description of the blueGreenUpdate field, please refer to [blueGreenUpdate](https://github.com/emqx/emqx-operator/blob/main-2.1/docs/en_US/reference/v1beta4-reference.md#evacuationstrategy).

```yaml
apiVersion: apps.emqx.io/v1beta4
kind: EmqxEnterprise
metadata:
   name: emqx-ee
spec:
   replicas: 3
   blueGreenUpdate:
    initialDelaySeconds: 5
    evacuationStrategy:
      waitTakeover: 5
      connEvictRate: 10
      sessEvictRate: 10
   template:
     spec:
       emqxContainer:
         image:
           repository: emqx/emqx-ee
           version: 4.4.14
         ports:
           - name: "http-dashboard"
             containerPort: 18083
```

> `waitTakeover` indicates the waiting time (unit is second) before the current node starts session evacuation. `connEvictRate` indicates the client disconnection rate of the current node (unit: count/second). `sessEvictRate` indicates the current node client session evacuation rate (unit: count/second). The `.spec.license.stringData` field is filled with the content of the license certificate. In this article, the content of this field is omitted. Please fill it with the content of your own certificate.

Save the above content as `emqx.yaml`, execute the following command to deploy the EMQX Enterprise cluster:

```bash
kubectl apply -f emqx.yaml
```

The output is similar to:

```
emqxenterprise.apps.emqx.io/emqx-ee created
```

- Check whether the EMQX Enterprise cluster is ready

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

## Use MQTT X CLI to Connect EMQX Cluster

MQTT X CLI is an open-source MQTT 5.0 CLI Client that supports automatic reconnection, and it is also a pure command-line mode MQTT X. Designed to help develop and debug MQTT services and applications faster without using a graphical interface. For documentation about MQTT X CLI, please refer to [MQTTX CLI](https://mqttx.app/docs/cli).

Execute the following command to connect to the EMQX cluster:

```bash
mqttx bench conn -h $host -p $port -c 3000
```

> `-h` indicates the IP of the host where the EMQX Pod is located. `-p` means nodePort port. `-c` indicates the number of connections to create. When deploying the EMQX cluster, this article usesThe NodePort pattern exposes services. If the service is exposed by LoadBalancer, `-h` should be the IP of LoadBalancer, and `-p` should be the EMQX MQTT service port.

The output is similar to:

```bash
[10:05:21 AM] › ℹ Start the connect benchmarking, connections: 3000, req interval: 10ms
✔ success [3000/3000] - Connected
[10:06:13 AM] › ℹ Done, total time: 31.113s
```

## Trigger EMQX Operator to Perform Blue-Green Upgrade

Modifying any content of the `.spec.template` field of the EmqxEnterprise object will trigger EMQX Operator to perform a blue-green upgrade. In this article, we modify the EMQX Container Name to trigger the upgrade, and users can modify it according to actual needs.

```bash
kubectl patch EmqxEnterprise emqx-ee --type='merge' -p '{"spec": {"template": {"spec": {"emqxContainer": {"name": "emqx-ee-a"}} }}}'
```

The output is similar to:

```
emqxenterprise.apps.emqx.io/emqx-ee patched
```

- Check the status of the blue-green upgrade

```bash
kubectl get emqxEnterprise emqx-ee -o json | jq ".status.blueGreenUpdateStatus.evacuationsStatus"
```

The output is similar to:

```bash
[
  {
    "connection_eviction_rate": 10,
    "node": "emqx-ee@emqx-ee-54fc496fb4-2.emqx-ee-headless.default.svc.cluster.local",
    "session_eviction_rate": 10,
    "session_goal": 0,
    "connection_goal": 22,
    "session_recipients": [
      "emqx-ee@emqx-ee-5d87d4c6bd-2.emqx-ee-headless.default.svc.cluster.local",
      "emqx-ee@emqx-ee-5d87d4c6bd-1.emqx-ee-headless.default.svc.cluster.local",
      "emqx-ee@emqx-ee-5d87d4c6bd-0.emqx-ee-headless.default.svc.cluster.local"
    ],
    "state": "waiting_takeover",
    "stats": {
      "current_connected": 0,
      "current_sessions": 0,
      "initial_connected": 33,
      "initial_sessions": 0
    }
  }
]
```

> `connection_eviction_rate` indicates the rate of node evacuation(unit: count/second). `node` indicates the node currently being evacuated. `session_eviction_rate` indicates the rate of node session evacuation(unit: count/second). `session_recipients` represents the list of recipients for session evacuation. `state` indicates the node evacuation phase. `stats` indicates the statistical indicators of the evacuated node, including the current number of connections (current_connected), the number of current sessions (current_sessions), the number of initial connections (initial_connected), and the number of initial sessions (initial_sessions).

### Use Prometheus to Monitor Client Connections During Upgrade

Use a browser to access the Prometheus web service, click Graph, enter `emqx_connections_count` in the search box, and click Execute, as shown in the following figure:

![](./assets/configure-emqx-blueGreenUpdate/prometheus.png)

It can be seen from the figure that there are two EMQX clusters, old and new, and each cluster has three EMQX nodes. After starting the blue-green upgrade, the connection of each node of the old cluster is disconnected at the configured rate and migrated to the nodes of the new cluster. Finally, all connections in the old cluster are completely migrated to the new cluster, which means the blue-green upgrade is complete.
