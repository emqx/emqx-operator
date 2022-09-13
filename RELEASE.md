## Release Note 🍻

EMQX Operator 2.0.1 is released.

### Supported version

- `apps.emqx.io/v1beta3`

  - `EMQX` at [4.4.8](https://www.emqx.com/en/changelogs/enterprise/4.4.8) and later
  - `EMQX Enterprise` at [4.4.8](https://www.emqx.com/en/changelogs/broker/4.4.8) and later

- `apps.emqx.io/v2alpha1`

  - `EMQX` at [5.0.6](https://www.emqx.com/en/changelogs/broker/5.0.6) and later

### Fixes 🛠

- `apps.emqx.io/v2alpha1`

  - Fix EMQX Custom Resource can not update at runtime by `kubectl apply`
  - Fix EMQX Custom Resource status error when updating the image

- Helm Chart

  - Fix `podAnnotations` not working
  - Delete hardcoded namespace and service name for Helm Template

### Other changes

- `apps.emqx.io/v2alpha1`

  - Set minimum is 1 for EMQX Core node
  - Add default readiness probe and default liveness probe for EMQX Replicant node