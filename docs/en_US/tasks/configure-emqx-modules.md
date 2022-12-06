# Configure EMQX modules

## Task target
- Learn how to configure various functional modules of the EMQX cluster through the modules field

## EMQX module configuration
EMQX CRD supports configuring various functional modules of the EMQX cluster through the .spec.emqxTemplate.modules field. For the description of the modules field, please refer to: [modules field](https://github.com/emqx/emqx-operator/blob/2.0.2/docs/en_US/reference/v1beta3-reference.md#emqxenterprisetemplate), EMQX  module documentation can refer to: [module management](https://docs.emqx.com/en/enterprise/v4.4/modules/modules.html)

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:kubectl exec -it emqx-ee-0 -- emqx ctl modules list
  name: emqx-ee
spec:
  emqxTemplate:
    image: emqx/emqx-ee:4.4.8
    modules:
      - name: "internal_acl"
        enable: true
        configs:
          acl_rule_file: "/mounted/acl/acl.conf"
      - name: "retainer"
        enable: true
        configs:
          expiry_interval: 0
          max_payload_size: "1MB"
          max_retained_messages: 0
          storage_type: "ram"
```

Save the above file as emqx-modules.yaml and deploy the EMQX cluster

```
kubectl apply -f emqx-modules.yaml
```

After the EMQX cluster is ready, use the following command to check whether the configured modules are enabled

```
kubectl exec -it emqx-ee-0 -- emqx ctl modules list
```

The output is similar to:

```
Module(internal_acl, description = "Internal ACL File", enabled = true)
Module(retainer, description = "Set parameters such as enable status, storage location, and expiration date for MQTT retain messages.", enabled = true)
```