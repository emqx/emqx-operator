# Release Note 🍻

EMQX Operator 2.2.23 has been released.

## Supported version
+ apps.emqx.io/v2beta1

  + EMQX at 5.1.1 and later
  + EMQX Enterprise at 5.1.1 and later

+ apps.emqx.io/v1beta4

  + EMQX at 4.4.14 and later
  + EMQX Enterprise at 4.4.14 and later

## Other Changes ✨

+ `apps.emqx.io/v2beta1 EMQX`.

  + Delete `apps.emqx.io/v2beta1/EMQX/status.coreNodes[*].uptime` and `apps.emqx.io/v2beta1/EMQX/status.replicantNodes[*].uptime`, because EMQX operator will constantly updating them. This is unnecessary wear and tear.

## How to install/upgrade EMQX Operator 💡

> Need make sure the [cert-manager](https://cert-manager.io/) is ready

```
helm repo add emqx https://repos.emqx.io/charts
helm repo update
helm upgrade --install emqx-operator emqx/emqx-operator \
  --namespace emqx-operator-system \
  --create-namespace \
  --version 2.2.23
kubectl wait --for=condition=Ready pods -l "control-plane=controller-manager" -n emqx-operator-system
```

## Warning 🚨
`apps.emqx.io/v1beta3` and `apps.emqx.io/v2alpha1` will be dropped soon
