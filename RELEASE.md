# Release Note 🍻

EMQX Operator 2.2.19 has been released.

## Supported version
+ apps.emqx.io/v2beta1

  + EMQX at 5.1.1 and later
  + EMQX Enterprise at 5.1.1 and later

+ apps.emqx.io/v1beta4

  + EMQX at 4.4.14 and later
  + EMQX Enterprise at 4.4.14 and later

## Fixes 🛠

+ `apps.emqx.io/v2beta1 EMQX`.

  + Fix the issue that the EMQX operator can not update the K8s service port when the user changes EMQX's listener port by EMQX dashboard.

  + Fix the issue when the EMQX customer resources are updated with changes that affect both the statefulSet and the EMQX config, the statefulSet is updated last and this blocks the update process if the statefulSet changes are referenced in EMQX config. check: https://github.com/emqx/emqx-operator/issues/1027

## Other Changes ✨

+ Update `sigs.k8s.io/controller-runtime` version to 0.17

## How to install/upgrade EMQX Operator 💡

> Need make sure the [cert-manager](https://cert-manager.io/) is ready

```
helm repo add emqx https://repos.emqx.io/charts
helm repo update
helm upgrade --install emqx-operator emqx/emqx-operator \
  --namespace emqx-operator-system \
  --create-namespace \
  --version 2.2.19
kubectl wait --for=condition=Ready pods -l "control-plane=controller-manager" -n emqx-operator-system
```

## Warning 🚨
`apps.emqx.io/v1beta3` and `apps.emqx.io/v2alpha1` will be dropped soon
