## Release Note 🍻

EMQX Operator 1.2.3 is released.

### Features 🌈

- Users can add extra Containers to the pod, see https://github.com/emqx/emqx-operator/issues/252.

- When deploy EMQX Custom Resource, if exist PVCs, set `.spec.podManagementPolicy = "Parallel"`, or else, set `.spec.podManagementPolicy = "OrderedReady"`, this can avoid to some extent the problem of EMQX cluster brain cleavage.

- Add `username` and `password` to `.spec.emqxTemplate` for EMQX Custom Resource, users can use them to set up the dashboard and API authentication, and also, users will no longer be able to create and modify `emqx_management` and `emqx_dashboard` plugins by EMQX Plugin Custom Resource.

- If users didn't set `acl` and `modules` in `.spec.emqxTemplate`, the ConfigMap will not be created.

- New fields for `.status` in EMQX Custom Resource.

- Now we don't create `volume` and `volumeMount` for EMQX logs anymore