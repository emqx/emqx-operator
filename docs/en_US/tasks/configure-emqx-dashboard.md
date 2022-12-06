# Configure EMQX Dashboard account

## Task target
- Learn how to configure EMQX Dashboard account through username and password fields

## EMQX Dashboard configuration
EMQX CRD supports configuring EMQX cluster Dashboard account through `.spec.emqxTemplate.username` and `.spec.emqxTemplate.passowrd` fields

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  emqxTemplate:
    username: test
    password: test
    image: emqx/emqx-ee:4.4.8
```

Save the content of the above file as: emqx-dashboard.yaml and deploy the EMQX cluster

```
kubectl apply -f emqx-dashboard.yaml
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

- Use port forwarding to access EMQX cluster Dashboard

```
kubectl port-forward  service/emqx-ee 32010:18083
```

Note: After the cluster is ready, you can use the configured username and password to log in to EMQX Dashboard
