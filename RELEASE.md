# Release Note 🍻

EMQX Operator 2.1.2 is released.

## Supported version

- `apps.emqx.io/v2alpha1`

  - `EMQX` at [5.0.14](https://www.emqx.com/en/changelogs/broker/5.0.14) and later
  - `EMQX Enterprise` at [5.0.0](https://www.emqx.com/en/changelogs/enterprise/5.0.0) and later

- `apps.emqx.io/v1beta4`

  - `EMQX` at [4.4.14](https://www.emqx.com/en/changelogs/broker/4.4.8) and later
  - `EMQX Enterprise` at [4.4.14](https://www.emqx.com/en/changelogs/enterprise/4.4.8) and later

## Fixes 🛠

- Both `apps.emqx.io/v1beta4` and `apps.emqx.io/v2alpha1`

  - Fixed an issue where EMQX Operator would frequently try to update statefulSet and deployment resources, even if there were no changes to the resources

- `apps.emqx.io/v2alpha1`

  - Fixed an issue where the EMQX replicant node would update before the EMQX Core node in some cases

## Enhancements 🚀

- More and better documents, please check [here](https://docs.emqx.com/en/emqx-operator)

- Add `additionalPrinterColumns` for `kind: EMQX` and `kind: EmqxEnterprise` and `kind: EmqxBroker`, now can get more friendly information when using `kubectl get emqx` or `kubectl get emqxenterprise` or `kubectl get emqxbroker`

- Add event filter for EMQX operator controller, reduced runtime memory consumption


## How to install/upgrade EMQX Operator 2.1.2 💡

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
