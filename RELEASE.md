## Release Note 🍻

Now we remove the v1beta1 in release-1.2.0 and add the v1beta3.

### Features 🌈

- Remove `.spec.emqxTemplate.env` and add the `.spec.env` in v1beta3
- Change acl type to []string
- Add new condition for init plugin
- Supply more logical plugin loaded
- Update service ports by emqx config
- Adjust emqx configuration（spec.emqxTemplate.env->spec.emqxTemplate.config) (#226)
- Update emqx plugin status
- Add new CRD for emqx plugin
- Add emqx client (#216)
- Add probe for `.spec.emqxTemplate`
- Add args for `.spec.emqxTemplate`
- Add initContainers for `.spec`
- Update kubebuilder marker and yaml file
- Update api version for controllers
- Add v1beta3 and delete v1beta1