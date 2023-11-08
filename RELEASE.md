# Release Note 🍻

EMQX Operator 2.2.4 has been released.

## Supported version
+ apps.emqx.io/v2beta1

  + EMQX at 5.1.1 and later
  + EMQX Enterprise at 5.1.1 and later

+ apps.emqx.io/v1beta4

  + EMQX at 4.4.14 and later
  + EMQX Enterprise at 4.4.14 and later

## Enhancements ✨

+ `apps.emqx.io/v2beta1 EMQX`.

  + Support EMQX dashboard https port

  + The `.spec.bootstrapAPIKeys` can support k8s secret, the user can set EMQX's bootstrap API keys like this:

    ```yaml
    apiVersion: v1
    kind: Secret
    metadata:
      name: emqx-secret
    stringData:
      key: foo
      secret: bar
    ---
    apiVersion: apps.emqx.io/v2beta1
    kind: EMQX
    metadata:
      name: emqx
    spec:
      image: emqx:5.1
      bootstrapAPIKeys:
        - secretRef:
            key:
              secretName: emqx-secret
              secretKey: key
            secret:
              secretName: emqx-secret
              secretKey: secret
    ```

## Fixes 🛠

+ `apps.emqx.io/v2beta1 EMQX`.

  + When performing a blue-green upgrade, EMQX Operator should select the old version of StatefulSet for request API

## How to install/upgrade EMQX Operator 💡

> Need make sure the [cert-manager](https://cert-manager.io/) is ready

```
helm repo add emqx https://repos.emqx.io/charts
helm repo update
helm upgrade --install emqx-operator emqx/emqx-operator \
  --namespace emqx-operator-system \
  --create-namespace \
  --version 2.2.4
kubectl wait --for=condition=Ready pods -l "control-plane=controller-manager" -n emqx-operator-system
```

## Warning 🚨
`apps.emqx.io/v1beta3` and `apps.emqx.io/v2alpha1` will be dropped soon
