# emqx-operator

The EMQX Operator provides [Kubernetes](https://kubernetes.io/) native deployment and management of [EMQX](https://www.emqx.io/) including EMQX Broker and EMQX Enterprise. The purpose of this project is to simplify and automate the configuration of the EMQX cluster.

## Prerequisites

- Kubernetes 1.20+

## Installing the Chart

To install the chart with the release name `emqx-operator`:

```console
## Add the EMQX Helm repository
$ helm repo add emqx https://repos.emqx.io/charts
$ helm repo update

## Install the emqx-operator helm chart
$ helm install emqx-operator emqx/emqx-operator \
      --namespace emqx-operator-system \
      --create-namespace
```

> **Tip**: List all releases using `helm ls -A`

## Uninstalling the Chart

To uninstall/delete the `emqx-operator` deployment:

```console
$ helm delete emqx-operator -n emqx-operator-system
```

## Configuration

The following table lists the configurable parameters of the cert-manager chart and their default values.

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `image.repository` | Image repository | `emqx/emqx-operator-controller` |
| `image.tag` | Image tag | `{{RELEASE_VERSION}}` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `imagePullSecrets` | Image pull secrets| `[]` |
| `nameOverride` | Override chart name | `""` |
| `fullnameOverride` | Default fully qualified app name. | `""` |
| `replicaCount`  | Number of cert-manager replicas  | `1` |
| `serviceAccount.create` | If `true`, create a new service account | `true` |
| `serviceAccount.name` | Service account to be used. If not set and `serviceAccount.create` is `true`, a name is generated using the fullname template |  |
| `serviceAccount.annotations` | Annotations to add to the service account |  |
| `resources` | CPU/memory resource requests/limits | `{}` |
| `nodeSelector` | Node labels for pod assignment | `{}` |
| `affinity` | Node affinity for pod assignment | `{}` |
| `tolerations` | Node tolerations for pod assignment | `[]` |
| `webhooks.cert` | Manager TLS secret for operator webhook server | `{}` |
| `webhooks.cert.secretName` | TLS secret for certificates for the `${NAME}-webhook-service.${NAMESPACE}.svc` | `""` |
| `webhooks.cert.cert-manager.enable` | Using [cert manager](https://github.com/jetstack/cert-manager) for provisioning the certificates for the webhook server. You can follow [the cert manager documentation](https://cert-manager.io/docs/installation/) to install it. | `false` |
| `webhooks.conversion.patch` | add `.spec.conversion` fields in CRDs | `{}` |
| `webhooks.conversion.patch.image.tag` | Image tag | `` |
| `webhooks.conversion.patch.image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `webhooks.conversion.patch.imagePullSecrets` | Image pull secrets| `[]` |
| `webhooks.conversion.patch.serviceAccount.create` | If `true`, create a new service account | `true` |
| `webhooks.conversion.patch.serviceAccount.name` | Service account to be used. If not set and `serviceAccount.create` is `true`, a name is generated using the {fullname}-conversion-patch |  |
| `webhooks.conversion.patch.serviceAccount.annotations` | Annotations to add to the service account |  |
| `webhooks.conversion.patch.resources` | CPU/memory resource requests/limits | `{}` |
| `webhooks.conversion.patch.nodeSelector` | Node labels for pod assignment | `{}` |
| `webhooks.conversion.patch.affinity` | Node affinity for pod assignment | `{}` |
| `webhooks.conversion.patch.tolerations` | Node tolerations for pod assignment | `[]` |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`.

Alternatively, a YAML file that specifies the values for the above parameters can be provided while installing the chart. For example,

```console
$ helm install emqx-operator -f values.yaml .
```
> **Tip**: You can use the default [values.yaml](https://github.com/emqx/emqx-operator/tree/main/deploy/charts/emqx-operator/values.yaml)

## Contributing

This chart is maintained at [github.com/emqx/emqx-operator](https://github.com/emqx/emqx-operator/tree/main/deploy/charts/emqx-operator).
