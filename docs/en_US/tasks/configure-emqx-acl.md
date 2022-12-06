# Configure EMQX publish/subscribe ACL

## Task target
- Learn how to configure EMQX cluster publish/subscribe ACL through the acl field

## EMQX publish/subscribe ACL configuration

EMQX CRD supports configuring EMQX publish/subscribe ACL through the `.spec.emqxTemplate.acl` field. For the specific description of the acl field, please refer to: [acl field](https://github.com/emqx/emqx-operator/blob/2.0.2/docs/en_US/reference/v1beta3-reference.md#emqxenterprisetemplate) , EMQX publish/subscribe ACL documentation You can refer to: [Publish/Subscribe ACL](https://docs.emqx.com/en/enterprise/v4.4/advanced/acl.html) 

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  emqxTemplate:
    image: emqx/emqx-ee:4.4.8
    acl:
    # Deny "all users" from subscribing to "$SYS/#" "#" topic
    - "{deny, all, subscribe, ["$SYS/#", {eq, "#"}]}."
```

Save the above content as emqx-acl.yaml and deploy the EMQX cluster

```
kubectl apply -f emqx-acl.yaml
```

- Check whether the EMQX cluster is running normally

```
kubectl get pods -l apps.emqx.io/instance=emqx-ee
```

The output is similar to:

```
NAME        READY   STATUS    RESTARTS   AGE
emqx-ee-0   2/2     Running   0          48m
emqx-ee-1   2/2     Running   0          48m
emqx-ee-2   2/2     Running   0          48m
```