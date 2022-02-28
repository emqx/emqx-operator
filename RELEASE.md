## Release Note 🍻

### Features 🌈

- Add telegraf sidecar container, sending metrics and events from `emqx_prometheus` plugin and emqx logs.
- Add `.spec.imagePullSecret` for Custom Resource
- Add `.spec.emqxTemplate.listener.certificate` for Custom Resource
- Add `.spec.emqxTemplate.listener.labels` for Custom Resource
- Add `.spec.emqxTemplate.listener.annotations` for Custom Resource
- Now update `.spec.license` in EmqxEnterprise does not require restart pods

### Fixes 🛠

- Fix update service failed in k8s 1.21
- Fix `.spec.listener.nodePort` not work