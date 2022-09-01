## Release Note 🍻

EMQX Operator 2.0.0 is released.

### Supported version

- `apps.emqx.io/v1beta3`

  - `EMQX` at [4.4.8](https://www.emqx.com/en/changelogs/enterprise/4.4.8)
  - `EMQX Enterprise` at [4.4.8](https://www.emqx.com/en/changelogs/broker/4.4.8)

- `apps.emqx.io/v2alpha1`

  - `EMQX` at [5.0.6](https://www.emqx.com/en/changelogs/broker/5.0.6)

#### Features 🌈

New API version: `apps.emqx.io/v2alpha1` and Kind: `EMQX`, support for EMQX 5 milestone versions, please check out reference docs

- New stateless node: EMQX Replicant, use `Deployment`

- New HOCON configuration style, in line with the format of EMQX 5

- Fully automated EMQX node upgrade management

### Broken Change 🚫

Must uninstall EMQX Operator 1 before installing this release
