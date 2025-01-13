# Release Note 🍻

EMQX Operator 2.2.27 has been released.

## Supported version
+ apps.emqx.io/v2beta1

  + EMQX at 5.1.1 and later
  + EMQX Enterprise at 5.1.1 and later

+ apps.emqx.io/v1beta4

  + EMQX at 4.4.14 and later
  + EMQX Enterprise at 4.4.14 and later

## Enhancements 🚀

+ EMQX operator helm chart can support the `podSecurityContext` and `containerSecurityContext` configuration, which can be used to configure the security context of the operator pod.

+ EMQX operator helm chart can disable web hooks by `webhook.enabled: false` in the `values.yaml` file, it will disable the web hooks of the operator, if you have any `apps.emqx.io/v2alpha1` or `apps.emqx.io/v1beta3` resources, please make sure the web hooks are enabled, otherwise the resources can not convert to the `apps.emqx.io/v2beta1` and `apps.emqx.io/v1beta4` resources.

## How to install/upgrade EMQX Operator 💡

> Need make sure the [cert-manager](https://cert-manager.io/) is ready

```
helm repo add emqx https://repos.emqx.io/charts
helm repo update
helm upgrade --install emqx-operator emqx/emqx-operator \
  --namespace emqx-operator-system \
  --create-namespace \
  --version 2.2.27
kubectl wait --for=condition=Ready pods -l "control-plane=controller-manager" -n emqx-operator-system
```

## Warning 🚨
`apps.emqx.io/v1beta3` and `apps.emqx.io/v2alpha1` will be dropped soon
