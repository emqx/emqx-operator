# Configure Prometheus to monitor EMQX cluster

## Task target

- How to monitor EMQX cluster through Prometheus.

## Deploy Prometheus

Prometheus deployment documentation can refer to: [Prometheus](https://github.com/prometheus-operator/prometheus-operator)

## Deploy EMQX cluster

:::: tabs type:card
::: tab v2alpha1

EMQX supports exposing indicators through the http interface. For all statistical indicators under the cluster, please refer to the document: [HTTP API](https://www.emqx.io/docs/en/v5.0/observability/prometheus.html)

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
   name: emqx
spec:
   image: emqx/emqx:5.0.14
   imagePullPolicy: IfNotPresent
   coreTemplate:
     spec:
       replicas: 3
       ports:
         - name: http-dashboard
           containerPort: 18083
   replicantTemplate:
     spec:
       replicas: 1
       ports:
         - name: http-dashboard
           containerPort: 18083
```

:::
::: tab v1beta4

EMQX supports exposing indicators through the http interface. For all statistical indicators under the cluster, you can refer to the document: [HTTP API](https://www.emqx.io/docs/en/v4.4/advanced/http-api.html#%E7%BB%9F%E8%AE%A1%E6%8C%87%E6%A0%87)

```yaml
apiVersion: apps.emqx.io/v1beta4
kind: EmqxEnterprise
metadata:
   name: emqx-ee
spec:
   replicas: 3
   template:
     spec:
       emqxContainer:
         image:
           repository: emqx/emqx-ee
           version: 4.4.14
         ports:
           - name: http-management
             containerPort: 8081
```

:::
::: tab v1beta3

EMQX supports exposing indicators through the http interface. For all statistical indicators under the cluster, you can refer to the document: [HTTP API](https://www.emqx.io/docs/en/v4.4/advanced/http-api.html#%E7%BB%9F%E8%AE%A1%E6%8C%87%E6%A0%87)

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
   name: emqx-ee
spec:
   replicas: 3
   emqxTemplate:
       image: emqx/emqx-ee:4.4.14
```

:::
::::

Save the above content as: emqx.yaml and deploy the EMQX cluster

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

## Configure Prometheus Monitor

:::: tabs type:card
::: tab v2alpha1

A PodMonitor Custom Resource Definition (CRD) allows to declaratively define how a dynamic set of services should be monitored. Use label selection to define which services are selected to monitor with the desired configuration, and its documentation can be referred to: [PodMonitor](https://github.com/prometheus-operator/prometheus-operator/blob/main/Documentation/design.md#podmonitor)

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PodMonitor
metadata:
   name: emqx
   namespace: default
   labels:
     app.kubernetes.io/name: emqx
spec:
   jobLabel: emqx-scraping
   namespaceSelector:
     matchNames:
     - default
   podMetricsEndpoints:
   - basicAuth:
       password:
         key: password
         name: emqx-basic-auth
       username:
         key: username
         name: emqx-basic-auth
     interval: 10s
     params:
       type:
       -prometheus
     path: /api/v5/prometheus/stats
     port: http-dashboard
     scheme: http
   selector:
     matchLabels:
       apps.emqx.io/instance: emqx
```

> `path` indicates the path of the indicator collection interface. In EMQX 5, the path is: `/api/v5/prometheus/stats`. `selector.matchLabels` indicates the label of the matching Pod: `apps.emqx.io/instance: emqx`.

:::
::: tab v1beta4

A PodMonitor Custom Resource Definition (CRD) allows to declaratively define how a dynamic set of services should be monitored. Use label selection to define which services are selected to monitor with the desired configuration, and its documentation can refer to: [PodMonitor](https://github.com/prometheus-operator/prometheus-operator/blob/main/Documentation/design.md#podmonitor)

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PodMonitor
metadata:
   name: emqx
   namespace: default
   labels:
     app.kubernetes.io/name: emqx
spec:
   jobLabel: emqx-scraping
   namespaceSelector:
     matchNames:
     - default
   podMetricsEndpoints:
   - basicAuth:
       password:
         key: password
         name: emqx-basic-auth
       username:
         key: username
         name: emqx-basic-auth
     interval: 10s
     params:
       type:
       -prometheus
     path: /api/v4/emqx_prometheus
     port: http-management
     scheme: http
   selector:
     matchLabels:
       apps.emqx.io/instance: emqx-ee
```

> `path` indicates the path of the indicator collection interface. In EMQX 4, the path is: `/api/v4/emqx_prometheus`. `selector.matchLabels` indicates the label of the matching Pod: `apps.emqx.io/instance: emqx-ee`.

:::
::: tab v1beta3

A ServiceMonitor Custom Resource Definition (CRD) allows to declaratively define how a dynamic set of services should be monitored. Use label selection to define which services are selected to monitor with the desired configuration, and its documentation can be referred to: [ServiceMonitor](https://github.com/prometheus-operator/prometheus-operator/blob/main/Documentation/design.md#servicemonitor)

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
   name: emqx
   namespace: default
   labels:
     app.kubernetes.io/name: emqx
spec:
   jobLabel: emqx-scraping
   namespaceSelector:
     matchNames:
     - default
   ServiceMetricsEndpoints:
   - basicAuth:
       password:
         key: password
         name: emqx-basic-auth
       username:
         key: username
         name: emqx-basic-auth
     interval: 10s
     params:
       type:
       -prometheus
     path: /api/v4/emqx_prometheus
     port: http-management-8081
     scheme: http
   selector:
     matchLabels:
       apps.emqx.io/instance: emqx-ee
```

> `path` indicates the path of the indicator collection interface. In EMQX 4, the path is: `/api/v4/emqx_prometheus`. `selector.matchLabels` means matching the label of Service: `apps.emqx.io/instance: emqx-ee`.

:::
::::

Save the above content as: monitor.yaml and execute the following command:

```bash
kubectl apply -f monitor.yaml
```

Use basicAuth to provide Monitor with password and account information for accessing EMQX interface

```yaml
apiVersion: v1
kind: Secret
metadata:
   name: emqx-basic-auth
   namespace: default
type: kubernetes.io/basic-auth
stringData:
   username: admin
   password: public
```

Save the above content as: secret.yaml and create Secret

```bash
kubectl apply -f secret.yaml
```

## Visit Prometheus to view the indicators of EMQX cluster

Open the Prometheus interface, switch to the Graph page, and enter emqx to display as shown in the following figure:

![](./assets/configure-emqx-prometheus/emqx-prometheus-metrics.png)

Switch to the Status → Targets page, the following figure is displayed, and you can see all monitored EMQX Pod information in the cluster:

![](./assets/configure-emqx-prometheus/emqx-prometheus-target.png)