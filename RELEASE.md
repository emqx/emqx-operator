# Release Note 🍻

EMQX Operator 2.2.1 is released.

## Supported version
+ apps.emqx.io/v2beta1

  + EMQX at 5.1.1 and later
  + EMQX Enterprise at 5.1.1 and later

+ apps.emqx.io/v1beta4

  + EMQX at 4.4.14 and later
  + EMQX Enterprise at 4.4.14 and later

## Enhancements ✨

+ `apps.emqx.io/v2beta1 EMQX`.

  + The window period when the service is unavailable during blue-green deployment has been canceled. Now, even during the blue-green release process, the EMQX service remains available.

  + Delete mutating webhook and validating webhook.

## Fixes 🛠

+ `apps.emqx.io/v2beta1 EMQX`.

  + Fix EMQX Operator controller will crash when getting EMQX listeners failed.

  + Fix always update statefulSet when set volume template in EMQX customer resource.

  + Fix nil pointer error caused by not finding statefulSet in certain situations.

  + Fix the issue where EMQX customer resource status is still `Ready` when deleting a Pod.

  + Fix the issue where the Pod cannot be ready when the EMQX custom resource has the labels from third-party settings.

## How to install/upgrade EMQX Operator 💡

> Need make sure the [cert-manager](https://cert-manager.io/) is ready

```
helm repo add emqx https://repos.emqx.io/charts
helm repo update
helm upgrade --install emqx-operator emqx/emqx-operator \
  --namespace emqx-operator-system \
  --create-namespace \
  --version 2.2.1
kubectl wait --for=condition=Ready pods -l "control-plane=controller-manager" -n emqx-operator-system
```

## Warning 🚨
`apps.emqx.io/v1beta3` and `apps.emqx.io/v2alpha1` will be dropped soon
