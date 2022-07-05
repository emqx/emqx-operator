## Release Note 🍻

EMQX Operator 1.2.1 is released.

### Features 🌈

- When updating EMQX Plugin Custom Resources, it will not restart Pods.

- EMQX Operator automatically binds ports to Service resources even if plugins or modules are enabled via EMQX Dashboard

### Fix 🛠

- We have added sidecar containers to reload the EMQX plugins when their configuration is updated

### Broken Change 🚫

- We no longer support the API version: `apps.emqx.io/v1beta2`