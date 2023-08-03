# Release Note 🍻

EMQX Operator 2.2.0 is released.

## Supported version
+ apps.emqx.io/v2beta1

  + EMQX at 5.1.1 and later
  + EMQX Enterprise at 5.1.1 and later

+ apps.emqx.io/v1beta4

  + EMQX at 4.4.14 and later
  + EMQX Enterprise at 4.4.14 and later

## Features 🌈

+ The `apps.emqx.io/v2alpha1 EMQX` upgrade to `apps.emqx.io/v2alpha2 EMQX`.

  + New configuration management, now can manage and update EMQX configuration through `apps.emqx.io/v2alpha2 EMQX`, for more details, please refer to [Change EMQX Configurations](https://docs.emqx.com/en/emqx-operatoe/latest/tasks/configure-emqx-config.html).

  + New upgrade strategy, now both EMQX 5 and EMQX Enterprise 5 can be upgraded through blue-green deployment. EMQX Enterprise Edition 5 also supports the feature of node evacuation, for more details, please refer to [Upgrade the EMQX cluster elegantly through blue-green deployment](https://docs.emqx.com/en/emqx-operatoe/latest/tasks/configure-emqx-blueGreenUpdate.html).


+ The `apps.emqx.io/v1beta4 rebalance` upgrade to `apps.emqx.io/v2alpha2 rebalance`, now `rebalance` can support both EMQX Enterprise 4 and EMQX Enterprise 5, for more details, please refer to [Cluster Load Rebalancing](https://docs.emqx.com/en/emqx-operator/latest/tasks/configure-emqx-rebalance.html).

## How to install/upgrade EMQX Operator 💡

> Need make sure the [cert-manager](https://cert-manager.io/) is ready

```
helm repo add emqx https://repos.emqx.io/charts
helm repo update
helm upgrade --install emqx-operator emqx/emqx-operator \
  --namespace emqx-operator-system \
  --create-namespace
kubectl wait --for=condition=Ready pods -l "control-plane=controller-manager" -n emqx-operator-system
```

## Warning 🚨
`apps.emqx.io/v1beta3` and `apps.emqx.io/v2alpha1` will be dropped soon
