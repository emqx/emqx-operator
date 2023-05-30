# Release Note 🍻

EMQX Operator 2.1.3 is released.

## Supported version

- `apps.emqx.io/v2alpha1`

  - `EMQX` at [5.0.14](https://www.emqx.com/en/changelogs/broker/5.0.14) and later
  - `EMQX Enterprise` at [5.0.0](https://www.emqx.com/en/changelogs/enterprise/5.0.0) and later

- `apps.emqx.io/v1beta4`

  - `EMQX` at [4.4.14](https://www.emqx.com/en/changelogs/broker/4.4.14) and later
  - `EMQX Enterprise` at [4.4.14](https://www.emqx.com/en/changelogs/enterprise/4.4.14) and later

## Features  🌈

- Add new CRD `apps.emqx.io/v1beta4 rebalance` to support the cluster load rebalancing, for more details, please refer to [Cluster Load Rebalancing](https://docs.emqx.com/en/emqx-operator/latest/tasks/configure-emqx-rebalance.html).

## Fixes 🛠

- `apps.emqx.io/v2alpha1`

  - Fixed an issue of the extra containers not working.

## How to install/upgrade EMQX Operator 2.1.3 💡

> Need make sure the [cert-manager](https://cert-manager.io) is ready

```
helm repo add emqx https://repos.emqx.io/charts
helm repo update
helm upgrade --install emqx-operator emqx/emqx-operator \
  --namespace emqx-operator-system \
  --create-namespace
```

If you have deployed `apps.emqx.io/v1beta3` resources, EMQX Operator will automatically convert them to `apps.emqx.io/v1beta4` resources, all without any manual intervention.

The resources of `apps.emqx.io/v2alpha1` will not receive any impact.

## Warning 🚨

`apps.emqx.io/v1beta3` will be dropped soon
