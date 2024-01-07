# Release Note 🍻

EMQX Operator 2.2.7 has been released.

## Supported version
+ apps.emqx.io/v2beta1

  + EMQX at 5.1.1 and later
  + EMQX Enterprise at 5.1.1 and later

+ apps.emqx.io/v1beta4

  + EMQX at 4.4.14 and later
  + EMQX Enterprise at 4.4.14 and later

## Enhancements ✨

+ `apps.emqx.io/v2beta1 EMQX`.

  + Sometimes the updated statefulSet / replicaSet will not be ready, because the EMQX node can not be started. Then we will roll back EMQX CR spec, the EMQX operator controller will create a new statefulSet / replicaSet. But the new statefulSet / replicaSet will be the same as the previous one, so we didn't need to create it, just change the EMQX status. 

## How to install/upgrade EMQX Operator 💡

> Need make sure the [cert-manager](https://cert-manager.io/) is ready

```
helm repo add emqx https://repos.emqx.io/charts
helm repo update
helm upgrade --install emqx-operator emqx/emqx-operator \
  --namespace emqx-operator-system \
  --create-namespace \
  --version 2.2.7
kubectl wait --for=condition=Ready pods -l "control-plane=controller-manager" -n emqx-operator-system
```

## Warning 🚨
`apps.emqx.io/v1beta3` and `apps.emqx.io/v2alpha1` will be dropped soon
