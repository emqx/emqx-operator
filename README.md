# emqx-operator

EMQX Operator is a [Kubernetes](https://kubernetes.io/) operator for managing [EMQX](https://emqx.com/) clusters.

## Description

The operator conceptually consists of the following parts:
- EMQX CRD: Definition for a resource that resembles the EMQX cluster in the Kubernetes API.
- Rebalance CRD: Definition for a resource that orchestrates rebalancing of EMQX clusters.
- Controller manager: Kubernetes controller that manages various Kubernetes resources according to EMQX CRs specifications.

This operator supports:
* Management of EMQX clusters in both regular and core-replicant deployment modes.
* Management of [EMQX DS](https://docs.emqx.com/en/emqx/latest/durability/durability_introduction.html#sessions-and-durable-storage) replication, including automatic rebalancing.
* Rebalancing of MQTT sessions and connections across EMQX cluster nodes.

## Compatibility

This operator is compatible with the following EMQX releases:
- EMQX 5.9
- EMQX 5.10
- EMQX 6.x

## Installation

### Release Manifest

The simplest way to install the operator is to apply the release manifest.
```sh
kubectl apply --server-side=true -f https://github.com/emqx/emqx-operator/releases/download/v2.3.3/install.yaml
kubectl wait --for=condition=Ready pods -l "control-plane=controller-manager" --namespace emqx-operator-system
```

This will install both the CRDs, the controller manager and relevant resources into the cluster. The controller manager will be deployed in the `emqx-operator-system` namespace.

### Helm

You can also install the operator from the EMQX Helm chart repository.
```sh
helm repo add emqx https://repos.emqx.io/charts
helm repo update
helm install emqx-operator emqx/emqx-operator \
  --namespace emqx-operator-system \
  --create-namespace \
  --version 2.3.3 \
  --wait
```

This installs the same CRDs, controller manager and relevant resources into the `emqx-operator-system` namespace.

## Upgrading

### From 2.2.x

To upgrade from 2.2.x to 2.3.x, you need to patch the existing CRDs first to explicitly remove the conversion webhook.
```sh
kubectl patch crd emqxes.apps.emqx.io     --type=json -p='[{"op":"replace", "path":"/spec/conversion", "value":{"strategy":"None"}}]'
kubectl patch crd rebalances.apps.emqx.io --type=json -p='[{"op":"replace", "path":"/spec/conversion", "value":{"strategy":"None"}}]'
```

After patching the CRDs, delete the existing controller manager deployment.
```sh
kubectl delete --ignore-not-found clusterrole emqx-operator-manager-role
kubectl delete --ignore-not-found clusterrolebinding emqx-operator-manager-rolebinding
kubectl delete --ignore-not-found mutatingwebhookconfiguration emqx-operator-mutating-webhook-configuration
kubectl delete --ignore-not-found validatingwebhookconfiguration emqx-operator-validating-webhook-configuration
kubectl delete --ignore-not-found namespace emqx-operator-system
```

Optionally, delete legacy CRDs.
```sh
kubectl delete --ignore-not-found crd emqxbrokers.apps.emqx.io emqxenterprises.apps.emqx.io emqxplugins.apps.emqx.io
```

After that, you can upgrade the operator by following usual installation steps.

## Troubleshooting

Operator exposes limited number of events to the Kubernetes API.
```sh
kubectl get events --sort-by=.lastTimestamp
```

Alternatively, if EMQX resources fail to reach `Ready` status condition, consult the controller manager logs for more details.
```sh
kubectl logs -l "control-plane=controller-manager" --tail=-1 --namespace emqx-operator-system
```

For Helm installations, the chart defaults to production logging. Enable the `development` flag to turn debug logging on.
```sh
helm upgrade emqx-operator emqx/emqx-operator \
  --namespace emqx-operator-system \
  --reuse-values \
  --set development=true
```

## Development

### Prerequisites
- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.24+.
- Access to a Kubernetes v1.24+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `OPERATOR_IMAGE`:**

```sh
make docker-build docker-push OPERATOR_IMAGE=<some-registry>/emqx-operator:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**
```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `OPERATOR_IMAGE`:**
```sh
make deploy OPERATOR_IMAGE=<some-registry>/emqx-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

### To Uninstall
**Delete the CRDs from the cluster:**
```sh
make uninstall
```

**Undeploy the controller:**
```sh
make undeploy
```

## Contributing

Contributions are welcome! Please see the [CONTRIBUTING.md](CONTRIBUTING.md) file for more information.

Automatically generated files are kept in the repository for reference. Do not forget to update them when you make changes to the project.
```sh
make generate manifests
git add config/crd/bases/
git add api/**/zz_generated.deepcopy.go
```

More information can be found in the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
