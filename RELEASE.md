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