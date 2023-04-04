# Deploy EMQX on Google Kubernetes Engine

## Overview

This guide will walk you through the process of deploying EMQ X, an open-source MQTT broker, on Google Kubernetes Engine (GKE). By following these steps, you'll learn how to create and configure an EMQ X deployment on GKE.

## Prerequisites

+ A Google Kubernetes Engine (GKE) cluster, for more information, please check [Creating an autopilot cluster](https://cloud.google.com/kubernetes-engine/docs/how-to/creating-an-autopilot-cluster)
+ MQTTX CLI, A user-friendly MQTT 5.0 command line tool, download it [here](https://mqttx.app/cli)


## Deploy EMQX on GKE

### Deploying EMQX Operator

To install EMQX Operator, please check [Quick Start](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/getting-started/getting-started.md)

::: warning
To install `cert-manager`, consult the official documentation:

- [GKE Autopilot](https://cert-manager.io/docs/installation/compatibility/#gke-autopilot)
- [Private GKE Cluster](https://cert-manager.io/docs/installation/compatibility/#gke)

Remember to install CRDs when running `helm` with the `--set installCRDs=true` flag.

More information can be found at [cert-manager](https://cert-manager.io).
:::

### Enable EMQX Cluster Persistence

Google Kubernetes Engine (GKE) now supports a new storage class designed specifically for EMQX message broker deployments. This GKE storage class enhances high-availability and performance, ensuring seamless MQTT message handling and efficient resource utilization in IoT and edge computing environments.
```Shell
kubectl get sc
```
Outputs:
```Shell
NAME                        PROVISIONER                    RECLAIMPOLICY   VOLUMEBINDINGMODE      ALLOWVOLUMEEXPANSION   AGE
standard                    kubernetes.io/gce-pd           Delete          Immediate              true                   47h
...
```

Here, we pick up `standard` as storage class in the guide

For more storage classes, please check [gcp docs](https://cloud.google.com/kubernetes-engine/docs/concepts/persistent-volumes#storageclasses)


### Deploying EMQX Cluster

Below are the relevant configurations for EMQX Custom Resource. Choose the corresponding APIVersion based on the EMQX version you want to deploy. For specific compatibility relationships, please check [EMQX Operator Compatibility](../README.md):

:::: tabs type:card
::: tab v2alpha1

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
  name: emqx
spec:
  image: "emqx:5.0"
  coreTemplate:
    spec:
      volumeClaimTemplates:
        storageClassName: standard
        resources:
          requests:
            storage: 20Mi
        accessModes:
        - ReadWriteOnce
  dashboardServiceTemplate:
    spec:
      type: LoadBalancer
  listenersServiceTemplate:
    spec:
      type: LoadBalancer
```


:::
::: tab v1beta4

```yaml
apiVersion: apps.emqx.io/v1beta4
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  persistent:
    metadata:
      name: emqx-ee
    spec:
      storageClassName: standard
      resources:
        requests:
          storage: 20Mi
      accessModes:
        - ReadWriteOnce
  template:
    spec:
      emqxContainer:
        image:
          repository: emqx/emqx-ee
          version: 4.4.15
  serviceTemplate:
    spec:
      type: LoadBalancer
```

:::
::::


### Veirty the deployment

:::: tabs type:card
::: tab v2alpha1

- Retrieve the listener IP address of the load balancer:
```Shell
lb_listener_ip=$(kubectl get svc emqx-listeners -o json | jq '.status.loadBalancer.ingress[0].ip')
```

- Retrieve the dashboard IP address of the load balancer:
```Shell
lb_dashboard_ip=$(kubectl get svc emqx-dashboard -o json | jq '.status.loadBalancer.ingress[0].ip')
```

- connect, publish, and subscribe using MQTTX CLI
```Shell
mqttx conn -h ${lb_listener_ip} -p 1883
mqttx sub -t 'hello' -h ${lb_listener_ip} -p 1883
mqttx pub -t 'hello' -h ${lb_listener_ip} -p 1883 -m 'from MQTTX CLI'
```

- Access the EMQX dashboard
```Shell
http://${lb_dashboard_ip}:18083
```

:::
::: tab v1beta4

- Retrieve the load balancer's IP address:
```Shell
lb_ip=$(kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip')
```


- connect, publish, and subscribe using MQTTX CLI
```Shell
mqttx conn -h ${lb_ip} -p 1883
mqttx sub -t 'hello' -h ${lb_ip} -p 1883
mqttx pub -t 'hello' -h ${lb_ip} -p 1883 -m 'from MQTTX CLI'
```

- Access the EMQX dashboard
```Shell
http://${lb_ip}:18083
```

:::
::::

## Handing LoadBalancer TLS offloading

Since Google LoadBalancer doesn't support TCP certificates, please check [discussion](https://github.com/emqx/emqx-operator/discussions/312) to address TCP certificate offloading issues.


## Conclusion

This tutorial has provided you with the necessary knowledge to successfully deploy an EMQ X instance on Google Kubernetes Engine (GKE).
