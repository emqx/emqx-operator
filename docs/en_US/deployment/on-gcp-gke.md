# Set Up EMQX on Google Kubernetes Engine

## Overview

This guide will walk you through the process of deploying EMQ X, an open-source MQTT broker, on Google Kubernetes Engine (GKE). By following these steps, you'll learn how to create and configure an EMQ X deployment on GKE.

## Prerequisites

+ A Google Kubernetes Engine (GKE) cluster, for more information, see [Creating an autopilot cluster](https://cloud.google.com/kubernetes-engine/docs/how-to/creating-an-autopilot-cluster)

+ MQTTX CLI, A user-friendly MQTT 5.0 command line tool, download it [here](https://mqttx.app/cli)

## DeployEMQX on GKE

### Set Up Cert Manager

To install `cert-manager`, consult the official documentation:

- [GKE Autopilot](https://cert-manager.io/docs/installation/compatibility/#gke-autopilot)
- [Private GKE Cluster](https://cert-manager.io/docs/installation/compatibility/#gke)

Remember to install CRDs when running `helm` with the `--set installCRDs=true` flag.

> More information can be found at [cert-manager](https://cert-manager.io).


### Enable EMQX Cluster Persistence

1. Connect to your GKE cluster using the command-line tool, such as Cloud Shell or a local terminal.

2. Create a YAML file that defines your StorageClass. The following is an example:
```yaml 
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: gce-pd
provisioner: kubernetes.io/gce-pd
parameters:
  type: pd-standard
```
In this example, the provisioner is set to "kubernetes.io/gce-pd", which means that the StorageClass will use Google Compute Engine Persistent Disk as the storage backend. The "parameters" section specifies the type of disk to use (pd-standard, which is the default).

3. Apply the YAML file using the kubectl apply command:
```yaml
kubectl apply -f my-storage-class.yaml
```

This will create the StorageClass in your GKE cluster. You can verify that the StorageClass has been created by running the following command:
```yaml
kubectl get storageclass
```

This will list all of the StorageClasses in your cluster, including the one you just created. You can use this StorageClass to provision persistent volumes for your applications in the cluster.


### Deploying EMQX Operator

To install `emqx-operator`, refer to the official [docs](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/getting-started/getting-started.md)

> **_NOTE_** The cert-manager installation was done in the previous step

### Deploying EMQX Cluster  

Below are the relevant configurations for EMQX Custom Resource. Choose the corresponding APIVersion based on the EMQX version you want to deploy. For specific compatibility relationships, See [EMQX Operator Compatibility](../README.md):

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
      podSecurityContext:
        runAsUser: 1000
        runAsGroup: 1000
        fsGroup: 1000
        fsGroupChangePolicy: Always
        supplementalGroups:
          - 1000
      volumeClaimTemplates:
        storageClassName: gce-pd
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
      storageClassName: gce-pd
      resources:
        requests:
          storage: 20Mi
      accessModes:
        - ReadWriteOnce
  template:
    spec:
      podSecurityContext:
        runAsUser: 1000
        runAsGroup: 1000
        fsGroup: 1000
        fsGroupChangePolicy: Always
        supplementalGroups:
          - 1000
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

- Retrieve the LoadBalancer's IP address:
```Shell
kubectl get svc | grep emqx
```

- connect, publish, and subscribe using MQTTX CLI
```Shell
mqttx conn -h ${load_balancer_ip} -p 1883 -u 'admin' -P 'public'
mqttx sub -t 'hello' -h ${load_balancer_ip} -p 1883
mqttx pub -t 'hello' -h ${load_balancer_ip} -p 1883 -m 'from MQTTX CLI'
```

- Access the EMQX dashboard
```Shell
http://${load-balancer-ip}:18083
```


## Handing LoadBalancer TLS offloading

Since Google LoadBalancer doesn't support TCP certificates, please refer to this [discussion](https://github.com/emqx/emqx-operator/discussions/312) to address TCP certificate offloading issues.


## Conclusion

This tutorial has provided you with the necessary knowledge to successfully deploy an EMQ X instance on Google Kubernetes Engine (GKE).
