# Configure EMQX License

## Task target

- Learn how to configure EMQX License through the data field
- Learn how to configure EMQX License through the secretName field
- Learn how to update EMQX License

## EMQX License configuration

EMQX CRD supports configuring the EMQX cluster license through the `.spec.emqxTemplate.license` field. For the specific description of the license field, please refer to: [License reference document](https://github.com/emqx/emqx-operator/blob/1.2.8/docs/en_US/reference/v1beta3-reference.md#license), EMQX License can be applied for on the EMQ official website: [Apply for EMQX Enterprise Edition License for free](https://www.emqx.com/en/apply-licenses/emqx)

### Configure the License through the data field

- Fill the Base64 encoded result of License content into the data field

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  emqxTemplate:
    image: emqx/emqx-ee:4.4.8
    license:
      data:
```
Save the above content as: emqx-license.yaml and deploy the EMQX cluster

```
kubectl apply -f emqx-license.yaml
```

- After the EMQX cluster is ready, check the license information of the EMQX cluster

```
kubectl exec -it emqx-ee-0 -- emqx_ctl license info
```

The output is similar to:

```
Defaulted container "emqx" out of: emqx, reloader
customer                 : EMQ X Evaluation
email                    : contact@emqx.io
max_connections          : 10
original_max_connections : 10
issued_at                : 2020-06-20 03:02:52
expiry_at                : 2049-01-01 03:02:52
vendor                   : EMQ Technologies Co., Ltd.
version                  : 4.4.8
type                     : official
customer_type            : 10
expiry                   : false
```

- Update the new License content to the data field and update the EMQX cluster, execute the following command to check whether the EMQX cluster License is updated

```
kubectl exec -it emqx-ee-0 -- emqx_ctl license info
```

The output is similar to:

``` 
Defaulted container "emqx" out of: emqx, reloader
customer                 : raoxiaoli
email                    : xiaoli.rao@emqx.io
max_connections          : 100000
original_max_connections : 100000
issued_at                : 2022-11-21 02:49:35
expiry_at                : 2022-12-01 02:49:35
vendor                   : EMQ Technologies Co., Ltd.
version                  : 4.4.8
type                     : official
customer_type            : 2
expiry                   : false
```

**Remarks**: If the certificate information is not updated, you can wait for a while. The update of the license depends on the reloader container, and there will be some delay.

### Configure License through the secretName field

- Create secret based on License file

```
kubectl create secret generic test --from-file=emqx.lic=license.lic
```

- Set the secretName field to the secret name created in the previous step: test

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  emqxTemplate:
    image: emqx/emqx-ee:4.4.8
    license:
      secretName: test
```

Save the above content as: emqx-license.yaml and deploy the EMQX cluster

```
kubectl apply -f emqx-license.yaml
```

- View EMQX cluster license information

```
kubectl exec -it emqx-ee-0 -- emqx_ctl license info 
```

The output is similar to:

```
Defaulted container "emqx" out of: emqx, reloader
customer                 : EMQ X Evaluation
email                    : contact@emqx.io
max_connections          : 10
original_max_connections : 10
issued_at                : 2020-06-20 03:02:52
expiry_at                : 2049-01-01 03:02:52
vendor                   : EMQ Technologies Co., Ltd.
version                  : 4.4.8
type                     : official
customer_type            : 10
expiry                   : false
```

- Update the secret with the new License, where new.license.lic is the name of the new License file

```
kubectl create secret generic test --from-file=emqx.lic=new.license.lic --dry-run -o yaml | kubectl apply  -f - 
```

- Check whether the EMQX cluster license has been updated

```
kubectl exec -it emqx-ee-0 -- emqx_ctl license info 
```

The output is similar to:

```
efaulted container "emqx" out of: emqx, reloader
customer                 : raoxiaoli
email                    : xiaoli.rao@emqx.io
max_connections          : 100000
original_max_connections : 100000
issued_at                : 2022-11-21 02:49:35
expiry_at                : 2022-12-01 02:49:35
vendor                   : EMQ Technologies Co., Ltd.
version                  : 4.4.8
type                     : official
customer_type            : 2
expiry                   : false
```

**Remarks**: If the certificate information is not updated, you can wait for a while. The update of the license depends on the reloader container, and there will be some delay.
