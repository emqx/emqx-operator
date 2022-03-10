## Release Note 🍻

### Features 🌈

- For `EMQX 4.4`, a DNS cluster is used by default, no additional `serviceAccount` needs to be created, `EMQX 4.3` still uses the k8s APIServer cluster
- The `EMQX` container can now be terminated more gracefully
  - Add `TerminationGracePeriodSeconds`
  - Add `preStop` command
  
### Notes 📗

- **Now we no longer support the creation of new v1beta1 resources,but existing v1beta1 resources are not affected**

### Fixes 🛠

- Fix `Telegraf` container run failed occasionally
- Fix `ACL` not work in emqx enterprise modules
- Fix can not use latest tag for emqx image