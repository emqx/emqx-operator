# Release Note 🍻

EMQX Operator 2.1.1 is released.

## Supported version

- `apps.emqx.io/v2alpha1`

  - `EMQX` at [5.0.14](https://www.emqx.com/en/changelogs/broker/5.0.14) and later
  - `EMQX Enterprise` at [5.0.0](https://www.emqx.com/en/changelogs/enterprise/5.0.0) and later

- `apps.emqx.io/v1beta4`

  - `EMQX` at [4.4.14](https://www.emqx.com/en/changelogs/broker/4.4.8) and later
  - `EMQX Enterprise` at [4.4.14](https://www.emqx.com/en/changelogs/enterprise/4.4.8) and later

## Features 🌈

Add new field bootstrap API keys in `apps.emqx.io/v1beta4` and `apps.emqx.io/v2alpha1`

Users can customize the keys and secrets required to request EMQX's API before EMQX is started, which helps with some of the operations tasks, until then, users must wait for EMQX to be ready and add them manually via the EMQX Dashboard.

## Fixes 🛠

- `apps.emqx.io/v2alpha1`

  - Fix an issue with EMQX pods not inheriting EMQX Custom Resource annotations

- `apps.emqx.io/v1beta4`

  - Fixed an issue where EMQX blue-green updating would not start in some cases
  - Fixed an issue where `.spec.persistence` did not work in some cases

## How to install/upgrade EMQX Operator 2.1.1 💡

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
