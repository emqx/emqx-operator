# Configure EMQX Enterprise Edition License

## Task target
 
- How to use the secretName field to configure the EMQX Enterprise Edition License.
- How to update the EMQX Enterprise Edition License.

## Use the secretName field to configure the License

- Create Secret based on License file

A Secret is an object that contains a small amount of sensitive information such as a password, token, or key. For more detailed documentation on Secret, please refer to: [Secret](https://kubernetes.io/docs/concepts/configuration/secret/). EMQX Operator supports using Secret to mount License information, so we need to create a Secret based on the License before creating an EMQX cluster.

EMQX Enterprise Edition License can be applied for free on EMQ official website: [Apply for EMQX Enterprise Edition License](https://www.emqx.com/en/apply-licenses/emqx).

```
kubectl create secret generic test --from-file=emqx.lic=/path/to/license/file
```

**NOTE**: `/path/to/license/file` indicates the path of the EMQX Enterprise Edition License file, which can be an absolute path or a relative path. For more details on using kubectl to create a Secret, please refer to the document: [Using kubectl to create a secret](https://kubernetes.io/docs/tasks/configmap-secret/managing-secret-using-kubectl/).

The output is similar to:

```
secret/test created
```

- Configure EMQX cluster

The corresponding CRD of EMQX Enterprise Edition in EMQX Operator is EmqxEnterprise. EmqxEnterprise supports configuring the EMQX Enterprise Edition License through `.spec.emqxTemplate.license.secretName` field. For the specific description of the secretName field, please refer to: [secretName](https://github.com/emqx/emqx-operator/blob/2.0.2/docs/en_US/reference/v1beta3-reference.md#license).

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

**NOTE**: `secretName` indicates the name of the Secret created in the previous step.

Save the above content as: emqx-license.yaml, and execute the following command to deploy the EMQX Enterprise Edition cluster.

```
kubectl apply -f emqx-license.yaml
```

The output is similar to:

```
emqxenterprise.apps.emqx.io/emqx-ee created
```

- Check whether the EMQX Enterprise Edition cluster is ready

```
kubectl get emqxenterprise emqx-ee -o json | jq ".status.emqxNodes"
```

The output is similar to:

```
[
   {
     "node": "emqx-ee@emqx-ee-1.emqx-ee-headless.default.svc.cluster.local",
     "node_status": "Running",
     "otp_release": "24.1.5/12.1.5",
     "version": "4.4.8"
   },
   {
     "node": "emqx-ee@emqx-ee-0.emqx-ee-headless.default.svc.cluster.local",
     "node_status": "Running",
     "otp_release": "24.1.5/12.1.5",
     "version": "4.4.8"
   },
   {
     "node": "emqx-ee@emqx-ee-2.emqx-ee-headless.default.svc.cluster.local",
     "node_status": "Running",
     "otp_release": "24.1.5/12.1.5",
     "version": "4.4.8"
   }
]
```

**NOTE**: `node` represents the unique identifier of the EMQX node in the cluster. `node_status` indicates the status of the EMQX node. `otp_release` indicates the version of Erlang used by EMQX. `version` indicates the EMQX version. EMQX Operator will pull up the EMQX cluster with three nodes by default, so when the cluster is running normally, you can see the information of the three running nodes. If you configure the `.spec.replicas` field, when the cluster is running normally, the number of running nodes displayed in the output should be equal to the value of replicas.

- Check the EMQX Enterprise Edition License information

```
kubectl exec -it emqx-ee-0 -c emqx -- emqx_ctl license info
```

The output is similar to:

```
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

**NOTE**: From the output results, you can see the basic information of the license we applied for, including applicant information, the maximum number of connections supported by the license, and the expiration time of the license.

## Update EMQX Enterprise Edition License

- Update EMQX Enterprise Edition License Secret

```
kubectl create secret generic test --from-file=emqx.lic=/path/to/license/file --dry-run -o yaml | kubectl apply -f -
```

The output is similar to:

```
secret/test configured
```

- Check whether the EMQX cluster license has been updated

```
kubectl exec -it emqx-ee-0 -c emqx -- emqx_ctl license info
```

The output is similar to:

```
customer                 : cloudnative
email                    : cloudnative@emqx.io
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

**NOTE**: If the certificate information is not updated, you can wait for a while, the update of the license will be delayed. From the above output results, we can see that the content of the License has been updated, which means that the EMQX Enterprise Edition License has been updated successfully.