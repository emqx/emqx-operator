# Deploy EMQX Cluster in k8s with restricted access

Here we are assuming k8s cluster does not have access to the internet, and the user does not have permissions to create and/or use `ClusterRole`.

+ Both `emqx-operator` and `emqx` are installed in the same namespace
+ Cert manager may be available cluster-wide or in the same namespace as `emqx-operator`
+ The `emqx-operator` is configured to use a private docker registry, and the `emqx` is configured to use a custom `securityContext`

## Task Target

- Push necessary images to a private docker registry
- Override default parameters of `cert-manager` to use private registry
- Manually install EMQX Operator CRDs
- Override default parameters of `emqx-operator` to use private registry, single namespace, custom `securityContext`, and disabled webhook
- Use custom `securityContext` for EMQX

## Push necessary docker images to a private docker registry

```bash
export CERT_MANAGER_VERSION='v1.16.2'
export EMQX_OPERATOR_VERSION='2.2.26'
export EMQX_VERSION='5.10.0'
export REGISTRY='my.private.registry'

CERT_MANAGER_IMAGES=(
    "cert-manager-controller"
    "cert-manager-cainjector"
    "cert-manager-webhook"
    "cert-manager-acmesolver"
    "cert-manager-startupapicheck"
)

pull_retag_push() {
    local source=$1
    local target=$2
    docker pull "$source"
    docker tag "$source" "$target"
    docker push "$target"
}

for img in "${CERT_MANAGER_IMAGES[@]}"; do
    pull_retag_push "quay.io/jetstack/$img:$CERT_MANAGER_VERSION" "$REGISTRY/jetstack/$img:$CERT_MANAGER_VERSION"
done

pull_retag_push "emqx/emqx-enterprise:$EMQX_VERSION" "$REGISTRY/emqx/emqx-enterprise:$EMQX_VERSION"
pull_retag_push "emqx/emqx-operator-controller:$EMQX_OPERATOR_VERSION" "$REGISTRY/emqx/emqx-operator-controller:$EMQX_OPERATOR_VERSION"
```

## Deploy cert-manager

Skip this step if cert-manager is installed in the cluster.

Update namespace name if required.

```bash
helm repo add jetstack https://charts.jetstack.io
helm repo update
helm upgrade --install cert-manager jetstack/cert-manager \
   --namespace emqx \
   --create-namespace \
   --set crds.enabled=true \
   --set image.repository=$REGISTRY/jetstack/cert-manager-controller \
   --set image.tag=$CERT_MANAGER_VERSION \
   --set webhook.image.repository=$REGISTRY/jetstack/cert-manager-webhook \
   --set webhook.image.tag=$CERT_MANAGER_VERSION \
   --set cainjector.image.repository=$REGISTRY/jetstack/cert-manager-cainjector \
   --set cainjector.image.tag=$CERT_MANAGER_VERSION \
   --set acmesolver.image.repository=$REGISTRY/jetstack/cert-manager-acmesolver \
   --set acmesolver.image.tag=$CERT_MANAGER_VERSION \
   --set startupapicheck.image.repository=$REGISTRY/jetstack/cert-manager-startupapicheck \
   --set startupapicheck.image.tag=$CERT_MANAGER_VERSION
```

## Deploy EMQX Operator

### Deploy CRDs manually from release assets

```bash
kubectl -n emqx apply -f https://github.com/emqx/emqx-operator/releases/download/$EMQX_OPERATOR_VERSION/crds.yaml
```

### Deploy emqx-operator

If cert-manager is installed cluster-wide already, add `--set cert-manager.enable=false`.

In this example `podSecurityContext` and `containerSecurityContext` contain default values, override as necessary.

```bash
helm repo add emqx https://repos.emqx.io/charts
helm repo update
helm upgrade --install emqx-operator emqx/emqx-operator \
  --namespace emqx \
  --create-namespace \
  --set singleNamespace=true \
  --set webhook.enabled=false \
  --set crds.enabled=false \
  --set-json='podSecurityContext={"runAsNonRoot":true}' \
  --set-json='containerSecurityContext={"allowPrivilegeEscalation":false}' \
  --set image.repository=$REGISTRY/emqx/emqx-operator-controller \
  --set image.tag=$EMQX_OPERATOR_VERSION
```

### Ensure emqx-operator is up and running

```bash
kubectl -n emqx wait --for=condition=Ready pods -l "control-plane=controller-manager"
```

## Configure EMQX Cluster

+ Save the following content as a YAML file and deploy it with the `kubectl apply` command

  ```yaml
  apiVersion: apps.emqx.io/v2beta1
  kind: EMQX
  metadata:
    name: emqx
    namespace: emqx
  spec:
    image: ${REGISTRY}/emqx/emqx-enterprise:${EMQX_VERSION}
    config:
      data: |
        license {
          key = "..."
        }
  ```

+ Wait for the EMQX cluster to be ready, you can check the status of EMQX cluster through `kubectl get` command, please make sure `STATUS` is `Running`, this may take some time

  ```bash
  $ kubectl get emqx emqx
  NAME   IMAGE                                             STATUS    AGE
  emqx   my.private.registry/emqx/emqx-enterprise:5.10.0   Running   10m
  ```
