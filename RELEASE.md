## Release Note 🍻

EMQX Operator 2.1.0 is released.

### Supported version

- `apps.emqx.io/v1beta4`

  - `EMQX` at [4.4.14](https://www.emqx.com/en/changelogs/broker/4.4.8) and later
  - `EMQX Enterprise` at [4.4.14](https://www.emqx.com/en/changelogs/enterprise/4.4.8) and later

- `apps.emqx.io/v2alpha1`

  - `EMQX` at [5.0.14](https://www.emqx.com/en/changelogs/broker/5.0.14) and later

### Features 🌈

New API version: `apps.emqx.io/v1beta4`, support for EMQX 4.4 and EMQX Enterprise 4.4, please check out reference docs

- Fully compatible with  `apps.emqx.io/v1beta3`

- The new blue-green updating feature ensures smooth migration of client connections during EMQX cluster upgrades, reducing peak server pressure, For more info please check: https://github.com/emqx/emqx-operator/blob/2.1.0/docs/en_US/tasks/configure-emqx-blueGreenUpdate.md

- EMQX Operator now uses the EMQX Bootstrap user to access the EMQX API, no longer the EMQX Dashboard user

### How to upgrade EMQX Operator 2.1 from EMQX Operator 2.0 💡

Take Helm Chart as an example:

```
helm upgrade --install emqx-operator emqx/emqx-operator \
  --namespace emqx-operator-system \
  --create-namespace
```

If you have deployed `apps.emqx.io/v1beta3` resources, EMQX Operator will automatically convert them to `apps.emqx.io/v1beta4` resources, all without any manual intervention.

The resources of `apps.emqx.io/v2alpha1` will not receive any impact.

### Warning 🚨	

`apps.emqx.io/v1beta3` will be dropped in EMQX Operator 2.1.1
