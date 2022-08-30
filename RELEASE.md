## Release Note 🍻

EMQX Operator 2.0.0-alpha.1 is released.

### Supported EMQX version

- EMQX 5.0.6 and later

#### Features 🌈

New API version: `apps.emqx.io/v2alpha1` and Kind: `EMQX`, support for EMQX 5 milestone versions, please check out reference docs

- New stateless node: EMQX Replicant, use `Deployment`

- New HOCON configuration style, in line with the format of EMQX 5

- Fully automated EMQX node upgrade management

### Broken Change 🚫

Must uninstall EMQX Operator 1 before install this release
