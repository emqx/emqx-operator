## Release Note 🍻

EMQX Operator 2.0.2 is released.

### Supported version

- `apps.emqx.io/v1beta3`

  - `EMQX` at [4.4.8](https://www.emqx.com/en/changelogs/enterprise/4.4.8) and later
  - `EMQX Enterprise` at [4.4.8](https://www.emqx.com/en/changelogs/broker/4.4.8) and later

- `apps.emqx.io/v2alpha1`

  - `EMQX` at [5.0.8](https://www.emqx.com/en/changelogs/broker/5.0.8) and later

### Features 🌈

  - `apps.emqx.io/v1beta3`

    - Add `.spec.emqxTemplate.registry` for EMQX Enterprise and EMQX Broker Custom Resource, the user can customize the registry that will be used for EMQX owner image, like `${registry}/emqx/emqx` and `${registry}/emqx/emqx-operator-reloader`, but it will not be used by other images, like sidecar container or else.

### Fixes 🛠

- `apps.emqx.io/v1beta3`

  - Fix that adds nodePort to the ports of the headless service when `.spec.emqxTemplate.serviceTemplate.spec.type` is set to nodePort

  - Fix the service can not be updated when EMQX Custom Resources's replicas equal 1

  - Fix `.status.condition[].lastTransitionTime` is not accurate

- `apps.emqx.io/v2alpha1`

  - Fix the crash of the EMQX Operator when the EMQX Custom Resources's StatefulSet is not created or not found

  - Fix the error when the user-defined EMQX Custom Resources' template name

  - Fix the EMQX Custom Resources's StatefulSet always update even if the user does not change the EMQX Custom Resources

  - Fix can not update EMQX Custom Resources when not set `node.cookie` in  `.spec.bootstrapConfig`

### Other changes

  - In `apps.emqx.io/v2alpha.1`, change the default value of `.spec.coreTemplate.spec.replicas` to 1

  - In `apps.emqx.io/v2alpha.1`, the template metadata can not be updated when EMQX Custom Resources running