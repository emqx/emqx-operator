## Release Note 🍻

### Features 🌈

- New Custom Resource Define: `EmqxPlugin`, for managing EMQX plugins and auto bind service port
  - After deploy EMQX Custom Resource, EMQX Operator will auto create default `EmqxPlugin` Custom Resource, and you can manage it.
  - You can create you own `EmqxPlugin` Custom Resource, and user-define them config, and EMQX Operator will load them to EMQX Custom Resource.
  - If the `EmqxPlugin` Custom Resource need listener port, EMQX Operator will bind the port to service.
  - If you delete some `EmqxPlugin` Custom Resource, EMQX Operator will unbind the port to service, and unload them to EMQX Custom Resource.

- New API version: `apps.emqx.io/v1beta3`, please check out reference docs
  - Now you can set up any EMQX configure via `.spec.emqxTemplate.config`, if you set some listener configure, EMQX Operator will auto bind the listening port to the service
  - Now we support set up `readinessProbe/livenessProbe/startupProbe` by `.spec.emqxTemplate` in EMQX Custom Resource
  - Now we support set up container `args` by `.spec.emqxTemplate ` in EMQX Custom Resource
  - Now we support set up `initContainers ` by `.spec` in EMQX Custom Resource
  - The format of our `.spec.emqxTemplate.acl` settings is now the same as the original emqx format, no additional conversions are needed

### Broken Change 🚫

- We not longer support API version: `apps.emqx.io/v1beta1`