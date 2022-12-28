## Release Note 🍻

EMQX Operator 2.0.3 is released.

### Supported version

- `apps.emqx.io/v1beta3`

  - `EMQX` at [4.4.8](https://www.emqx.com/en/changelogs/enterprise/4.4.8) and later
  - `EMQX Enterprise` at [4.4.8](https://www.emqx.com/en/changelogs/broker/4.4.8) and later

- `apps.emqx.io/v2alpha1`

  - `EMQX` at [5.0.8](https://www.emqx.com/en/changelogs/broker/5.0.8) and later

### Fixes 🛠

- Fixes a conflict when a user defines a port in the service template that duplicates an EMQX listener

- Fixes an issue with service resources not having annotation from Custom Resource resources

- Fixed incorrect DNS Name of Certificate resource in Cert Manger from EMQX Operator Helm Chart 

### Other changes

- In EMQX Operator Helm Chart, add extra selectors to avoid conflicts with other stuff in the same namespace

- Add More document
