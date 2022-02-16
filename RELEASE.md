## Release Note 🍻

🆕 Happy Valentine's Day!

### Features 🌈

- Now we can use helm to deploy, like this:

  ```
  $ helm repo add emqx https://repos.emqx.io/charts
  $ helm repo update
  $ helm install emqx-operator emqx/emqx-operator \
            --set installCRDs=true \
            --set cert-manager.enable=true \
            --namespace emqx-operator-system \
            --create-namespace
  ```

### Fixes 🛠

- Fix ".spec.storage" conversion failed for EmqxEnterprise in v1beta2 API version

- Fix an issue where the controller frequently performs updates when Custom Resource are not updated