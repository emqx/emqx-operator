## Release Note 🍻

EMQX Operator 1.2.8 is released.

### Supported EMQX version

- EMQX 4.4.8 and later

- EMQX Enterprise 4.4.8 and later

### Features 🌈

- Add `.spec.emqxTemplate.registry` for EMQX Enterprise and EMQX Broker Custom Resource, the user can customize registry will used for EMQX owner image, like `${registry}/emqx/emqx` and `${registry}/emqx/emqx-operator-reloader`, but it will not be use other images, like sidecar container or else.

### Fixes 🛠

- Fix `.status.condition[].lastTransitionTime` is not accurate
