## Release Note 🍻

🆕 Happy New Year!

### Features 🌈

- New APIVersion: apps.emqx.io/v1beta2
- Restart pods when update `configmap` and `secret`
- Retain PVC when deleting EMQ X Custom Resource
- The emqx operator controller automatically creates the `serviceAccount`, `role` and `roleBinding` resources when the resource specified by `.spec.serviceAccountName` does not exist
- `.spec.serviceAccountName` is no longer required in EMQ X Custom Resource
- `.spec.replicas` is no longer required in EMQ X Custom Resource
- Add extra volumes mount possibility
- Add readiness probe for Statefulset

### Fixes 🛠

- Fix annotations for EMQ X Custom Resource not working
