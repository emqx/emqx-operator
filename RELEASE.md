## Release Note 🍻

EMQX Operator 1.2.6 is released.

### Supported EMQX version

- EMQX 4.4.8 and later

- EMQX Enterprise 4.4.8 and later

### Features 🌈

- Add `.spec.emqxTemplate.license.secretName` for EMQX Enterprise Custom Resource, the user can create the EMQX license as a Kubernetes secret resource and use in MQX Enterprise Custom Resource

- After the user updates the license, the EMQX Operator completes the runtime update via the EMQX API

### Fixes 🛠

- Now it's not possible to update `.spec.persistent` in the EMQX Custom Resource runtime

- Now it does not create `loaded_plugins` configMap for EMQX Custom Resource, this is to fix the `erofs` error in EMQX
