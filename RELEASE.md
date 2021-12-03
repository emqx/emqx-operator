## Release Note 🍻

### Features 🌈

- Custom Resource Definition for **EMQ X Broker** and **EMQ X Enterprise**
- Support EMQ X Broker 4.3.x version and EMQ X Enterprise 4.3.x version
- Readability configuration about `acl`，`modules` and `plugins` in `YAML`
- Support persistence for data and logs in EMQ X Cluster
- Support the strategy of node discovery `k8s`
- Support EMQ X metrics monitoring with `Prometheus`
- Scaling EMQ X cluster without disconnection
- Dynamic storage provisioning with pvc template
- Resources restrictions with k8s requests and limits
- Node selector

### Breaking Changes 💡

- Bump APIversion to `apps.emqx.io/v1beta1`
- Add the `CRD` of `EMQ X Enterprise`
- Remove configurations:`cluster`, `loadedPluginConf`,`loadedModulesConf`,`aclConf`
