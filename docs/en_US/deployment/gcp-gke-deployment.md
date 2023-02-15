# This article mainly introduces how to deploy EMQX cluster on Google Cloud Platform GKE with EMQX Operator. The main contents including:

## Terminology explanation

EMQX: The most scalable open-source MQTT broker for IoT, For details: [EMQX docs](https://github.com/emqx/emqx)

EMQX Operator: A Kubernetes Operator for EMQX, For details: [EMQX Operator docs](https://github.com/emqx/emqx-operator)

GKE: Google Kubernetes Engine, For details: [GKE docs](https://cloud.google.com/kubernetes-engine/)

## Create GKE Cluster

Log in Google Cloud GKE console and enter the page for creating GKE Cluster. For details: [Create GKE cluster](https://cloud.google.com/kubernetes-engine/docs/how-to/creating-an-autopilot-cluster)

## Access Kubernetes cluster

For details: [kubeconfig](https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-access-for-kubectl)

## Cert manager configurations

To install `cert-manager` read the official docs:

- [GKE Autopilot](https://cert-manager.io/docs/installation/compatibility/#gke-autopilot)
- [Private GKE Cluster](https://cert-manager.io/docs/installation/compatibility/#gke)

Don't forget to install CRDs when running `helm` using `--set installCRDs=true`.

> More info at [cert-manager](https://cert-manager.io).

## EMQX Operator

Deploy it as usual.

## EMQX cluster creation

[Operator installation reference](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/getting-started/getting-started.md)

After Operator is installed, deploy EMQX cluster in GKE as usual.

### Troubleshoot: Context deadline exceeded when calling "mutating.apps.emqx.io" webhook

Before deploying your EMQX CRD, you need to make sure yout VPC doesn't have any firewall rules that denies traffic to the EMQX webhook.
Open port 443 to allow receiveing ingress traffic.

## Persistence

To enable persistence in GKE using `volumeClaimTemplates` you must configure a **Pod/Container Security Contexts**.

As reference, this config should work as it is:

```yaml
spec:
  coreTemplate:
    spec:
      podSecurityContext:
        runAsUser: 1000
        runAsGroup: 1000
        fsGroup: 1000
        fsGroupChangePolicy: Always
        supplementalGroups:
          - 1000
      containerSecurityContext:
        runAsNonRoot: true
        runAsUser: 1000
        runAsGroup: 1000
```
