# Getting Started

**Note**: EMQX Operator Controller requires Kubernetes v1.20.0 and up.

## Background

This article was deployed using minikube v1.20.0

## Deployment Operator Controller

This project can be run inside a kubernetes cluster

### Prepare

We using [cert manager](https://github.com/jetstack/cert-manager) for provisioning the certificates for the webhook server. You can follow [the cert manager documentation](https://cert-manager.io/docs/installation/) to install it.

### Default static install

> You don’t require any tweaking of the EMQX Operator Controller install parameters.

The default static configuration can be installed as follows:

```shell
$ curl -f -L "https://github.com/emqx/emqx-operator/releases/download/1.1.6/emqx-operator-controller.yaml" | kubectl apply -f -
```

### Installing with Helm

1.  Add the EMQX Helm repository:
   ```
   $ helm repo add emqx https://repos.emqx.io/charts
   $ helm repo update
   ```
2. Install EMQX Operator Controller by helm
   ```
   $ helm install emqx-operator emqx/emqx-operator \
      --set installCRDs=true \
      --namespace emqx-operator-system \
      --create-namespace
   ```
**Note**:
If you want to collect metrics about `EMQX Operator` by `Prometheus`,first of all you need to deploy [Prometheus Operator](https://github.com/prometheus-operator/prometheus-operator)

```bash
$ cat << EOF kubectl apply -f -
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  labels:
    control-plane: controller-manager
  name: emqx-operator-controller-manager-metrics-monitor
  namespace: emqx-operator-system
spec:
  endpoints:
  - bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
    path: /metrics
    port: https
    scheme: https
    tlsConfig:
      insecureSkipVerify: true
  selector:
    matchLabels:
      control-plane: controller-manager
EOF
```

### Check EMQX Operator Controller status

   ```shell
   $ kubectl get pods -l "control-plane=controller-manager" -n emqx-operator-system
   NAME                                                READY   STATUS    RESTARTS   AGE
   emqx-operator-controller-manager-68b866c8bf-kd4g6   1/1     Running   0          15s
   ```

## Deploy the EMQX Broker

1. Deploy EMQX Custom Resource
   ```
   cat << "EOF" | kubectl apply -f -
   apiVersion: apps.emqx.io/v1beta2
   kind: EmqxBroker
   metadata:
     name: emqx
   spec:
     image: emqx/emqx:4.4.0
   EOF
   ```

2. Check EMQX status
   ```bash
   $ kubectl get pods
   NAME              READY   STATUS    RESTARTS   AGE
   emqx-0   1/1     Running   0          22m
   emqx-1   1/1     Running   0          22m
   emqx-2   1/1     Running   0          22m

   $ kubectl exec -it emqx-0 -- emqx_ctl status
   Node 'emqx@emqx-0.emqx.default.svc.cluster.local' 4.3.11 is started

   $ kubectl exec -it emqx-0 -- emqx_ctl cluster status
   Cluster status: #{running_nodes =>
                         ['emqx@emqx-0.emqx.default.svc.cluster.local',
                          'emqx@emqx-1.emqx.default.svc.cluster.local',
                          'emqx@emqx-2.emqx.default.svc.cluster.local'],
                     stopped_nodes => []}
   ```