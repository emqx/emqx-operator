## Release Note 🍻

### Features 🌈

- Support DNS Cluster for emqx-4.4.x
- The `EMQX` container can now be terminated more gracefully
  - Add `TerminationGracePeriodSeconds`
  - Add `preStop` command
  
### Notes 📗

- **Now we no longer support the creation of new v1beta1 resources,but existing v1beta1 resources are not affected**

### Fixes 🛠

- Fix `Telegraf` container run failed occasionally
- Fix `ACL` not work in emqx enterprise modules
- Fix can not use latest tag for emqx image