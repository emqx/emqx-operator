# Configure EMQX log level

## Task target

- Learn how to configure the EMQX log level through the env field
- Learn how to configure EMQX log levels through the config field

## EMQX log level configuration

EMQX CRD supports configuring the EMQX cluster log level through the `.spec.env` field. For the specific description of the env field, please refer to: [env field description](https://github.com/emqx/emqx-operator/blob/1.2.8/docs/en_US/reference/v1beta3-reference.md#emqxenterprise) also supports configuring the log level through the `.spec.emqxTemplate.config `field. For the specific description of the config field, please refer to: [config field description](https://github.com/emqx/emqx-operator/blob/1.2.8/docs/en_US/reference/v1beta3-reference.md#emqxenterprisetemplate), there is essentially no difference between the two methods, and eventually the log level will be configured for the environment variable EMQX. For the configuration of the EMQX environment variable, please refer to: [Configuration from environment variable](https://www.emqx.io/docs/en/v4/configuration/configuration.html)

### Configure the EMQX cluster log level  through the env field

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  env:
    - name: EMQX_LOG__LEVEL
      value: debug
  emqxTemplate:
    image: emqx/emqx-ee:4.4.8
```

Save the above content as: emqx-log.yaml and deploy the EMQX cluster

```
kubectl apply -f emqx-log.yaml
```

- Check whether the EMQX cluster is running normally

```
kubectl get pods  -l  apps.emqx.io/instance=emqx-ee
```

The output is similar to:

```
NAME        READY   STATUS    RESTARTS   AGE
emqx-ee-0   2/2     Running   0          48m
emqx-ee-1   2/2     Running   0          48m
emqx-ee-2   2/2     Running   0          48m

```
kubectl logs emqx-ee-0
```

### Configure the EMQX log level through the config field

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  emqxTemplate:
    image: emqx/emqx-ee:4.4.8
  config:
    log.level: debug
```

Save the above content as: emqx-config.yaml and deploy the EMQX cluster

```
kubectl apply -f emqx-config.yaml
```

- Check whether the EMQX cluster is running normally

```
kubectl get pods  -l  apps.emqx.io/instance=emqx-ee
```

The output is similar to:

```
NAME        READY   STATUS    RESTARTS   AGE
emqx-ee-0   2/2     Running   0          48m
emqx-ee-1   2/2     Running   0          48m
emqx-ee-2   2/2     Running   0          48m
```

- View EMQX cluster log information

```
kubectl logs emqx-ee-0
```
