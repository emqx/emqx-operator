# Configure EMQX Enterprise Edition License

## Task target
 
- How to configure EMQX Enterprise Edition License.
- How to update the EMQX Enterprise Edition License.

## Configure EMQX Enterprise Edition License

EMQX Enterprise Edition License can be applied for free on EMQ official website: [Apply for EMQX Enterprise Edition License](https://www.emqx.com/en/apply-licenses/emqx).

:::: tabs type:card
::: tab v2alpha1

- Configure EMQX cluster

The corresponding CRD of EMQX Enterprise Edition in EMQX Operator is EMQX. EMQX CRD supports using `.spec.bootstrapConfig` to configure EMQX cluster license. For bootstrapConfig configuration, please refer to the document: [bootstrapConfig](https://www.emqx.io/docs/en/v5.0/admin/cfg.html). This field is only allowed to be configured when creating an EMQX cluster, and does not support updating. **Note:** After creating an EMQX cluster, if you need to update the license, please update it through the EMQX Dashboard.

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
   name: emqx-ee
spec:
   bootstrapConfig: |
     license {
       key = "..."
     }
   image: emqx/emqx-enterprise:5.0.0
   dashboardServiceTemplate:
     metadata:
       name: emqx-dashboard
     spec:
       type: NodePort
       selector:
         apps.emqx.io/db-role: core
       ports:
         - name: "dashboard-listeners-http-bind"
           protocol: TCP
           port: 18083
           targetPort: 18083
           nodePort: 32012
```

>  `license.key` in the `bootstrapConfig` field indicates the content of the license. In this example, the content of the license is omitted, please fill it in by the user.

:::
::: tab v1beta4

- Create Secret based on License file

A Secret is an object that contains a small amount of sensitive information such as a password, token, or key. For more detailed documentation on Secret, please refer to: [Secret](https://kubernetes.io/docs/concepts/configuration/secret/). EMQX Operator supports using Secret to mount License information, so we need to create a Secret based on the License before creating an EMQX cluster.

```
kubectl create secret generic test --from-file=emqx.lic=/path/to/license/file
```

> `/path/to/license/file` indicates the path of the EMQX Enterprise Edition License file, which can be an absolute path or a relative path. For more details on using kubectl to create a Secret, please refer to the document: [Using kubectl to create a secret](https://kubernetes.io/docs/tasks/configmap-secret/managing-secret-using-kubectl/).

The output is similar to:

```
secret/test created
```

- Configure EMQX cluster

The corresponding CRD of EMQX Enterprise Edition in EMQX Operator is EmqxEnterprise. EmqxEnterprise supports configuring EMQX Enterprise Edition License through `.spec.license.secretName` field. For the specific description of the secretName field, please refer to: [secretName](https://github.com/emqx/emqx-operator/blob/main-2.1/docs/en_US/reference/v1beta4-reference.md#emqxlicense).

```yaml
apiVersion: apps.emqx.io/v1beta4
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  license:
    secretName: test
  template:
    spec:
      emqxContainer:
        image:
          repository: emqx/emqx-ee
          version: 4.4.14
```

> `secretName` indicates the name of the Secret created in the previous step.

:::
::: tab v1beta3

- Create Secret based on License file

A Secret is an object that contains a small amount of sensitive information such as a password, token, or key. For more detailed documentation on Secret, please refer to: [Secret](https://kubernetes.i/docs/concepts/configuration/secret/). EMQX Operator supports using Secret to mount License information, so we need to create a Secret based on the License before creating an EMQX cluster.

```bash
kubectl create secret generic test --from-file=emqx.lic=/path/to/license/file
```

> `/path/to/license/file` indicates the path of the EMQX Enterprise Edition License file, which can be an absolute path or a relative path. For more details on using kubectl to create a Secret, please refer to the document: [Using kubectl to create a secret](https://kubernetes.io/docs/tasks/configmap-secret/managing-secret-using-kubectl/).

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
     image: emqx/emqx-ee:4.4.14
     license:
       secretName: test
```

>  `secretName` indicates the name of the Secret created in the previous step.

:::
::::

Save the above content as: emqx-license.yaml, and execute the following command to deploy the EMQX Enterprise Edition cluster.

```bash
kubectl apply -f emqx-license.yaml
```

The output is similar to:

```
emqx.apps.emqx.io/emqx-ee created
```

- Check whether the EMQX Enterprise Edition cluster is ready

:::: tabs type:card
::: tab v2alpha1

```bash
kubectl get emqx emqx-ee -o json | jq '.status.conditions[] | select( .type == "Running" and .status == "True")'

```

The output is similar to:

```bash
{
   "lastTransitionTime": "2023-02-10T02:46:36Z",
   "lastUpdateTime": "2023-02-07T06:46:36Z",
   "message": "Cluster is running",
   "reason": "ClusterRunning",
   "status": "True",
   "type": "Running"
}
```

:::
::: tab v1beta4

```bash
kubectl get emqxEnterprise emqx-ee -o json | jq '.status.conditions[] | select( .type == "Running" and .status == "True")'
```
The output is similar to:

```bash
{
  "lastTransitionTime": "2023-03-01T02:49:22Z",
  "lastUpdateTime": "2023-03-01T02:49:23Z",
  "message": "All resources are ready",
  "reason": "ClusterReady",
  "status": "True",
  "type": "Running"
}
```

:::
::: tab v1beta3

```bash
kubectl get emqxEnterprise emqx-ee -o json | jq '.status.conditions[] | select( .type == "Running" and .status == "True")'
```

The output is similar to:

```bash
{
  "lastTransitionTime": "2023-03-01T02:49:22Z",
  "lastUpdateTime": "2023-03-01T02:49:23Z",
  "message": "All resources are ready",
  "reason": "ClusterReady",
  "status": "True",
  "type": "Running"
}
```

:::
::::

- Check the EMQX Enterprise Edition License information

```bash
kubectl exec -it emqx-ee-core-0 -c emqx -- emqx_ctl license info
```

The output is similar to:

```bash
customer        : EMQ
email           : cloudnative@emqx.io
deployment      : deployment-6159820
max_connections : 10000
start_at        : 2023-02-16
expiry_at       : 2023-05-17
type            : trial
customer_type   : 0
expiry          : false

```

> From the output results, you can see the basic information of the license we applied for, including applicant information, the maximum number of connections supported by the license, and the expiration time of the license.

## Update EMQX Enterprise Edition License

:::: tabs type:card
::: tab v2alpha1

- Update License through EMQX Dashboard

Open the browser, enter the host `IP` and port `32012` where the EMQX Pod is located, log in to the EMQX cluster Dashboard (Dashboard default user name: admin, default password: public), enter the Dashboard click Overview and pull down the page to the bottom to see The current license information of the cluster, as shown in the following figure:

![](./assets/configure-emqx-license/emqx-dashboard-license.png)

Then click the Update License button to upload the latest License Key content, as shown in the following figure:

![](./assets/configure-emqx-license/emqx-license-upload.png)

Finally, click the Save button to save the update. The following picture shows the updated License information:

![](./assets/configure-emqx-license/emqx-license-update.png)

As can be seen from the above figure, the content of the license has been updated, which means that the license has been updated successfully.

:::
::: tab v1beta4

- Update EMQX Enterprise Edition License Secret

```bash
kubectl create secret generic test --from-file=emqx.lic=/path/to/license/file --dry-run -o yaml | kubectl apply -f -
```

The output is similar to:

```
secret/test configured
```

- Check whether the EMQX cluster license has been updated

```bash
kubectl exec -it emqx-ee-0 -c emqx -- emqx_ctl license info
```

The output is similar to:

```bash
customer                 : cloudnative
email                    : cloudnative@emqx.io
max_connections          : 100000
original_max_connections : 100000
issued_at                : 2022-11-21 02:49:35
expiry_at                : 2022-12-01 02:49:35
vendor                   : EMQ Technologies Co., Ltd.
version                  : 4.4.14
type                     : official
customer_type            : 2
expiry                   : false
```

> If the certificate information is not updated, you can wait for a while, the update of the license will be delayed. From the above output results, we can see that the content of the License has been updated, which means that the EMQX Enterprise Edition License has been updated successfully.

:::
::: tab v1beta3

- Update EMQX Enterprise Edition License Secret

```bash
kubectl create secret generic test --from-file=emqx.lic=/path/to/license/file --dry-run -o yaml | kubectl apply -f -
```

The output is similar to:

```
secret/test configured
```

- Check whether the EMQX cluster license has been updated

```bash
kubectl exec -it emqx-ee-0 -c emqx -- emqx_ctl license info
```

The output is similar to:

```bash
customer                 : cloudnative
email                    : cloudnative@emqx.io
max_connections          : 100000
original_max_connections : 100000
issued_at                : 2022-11-21 02:49:35
expiry_at                : 2022-12-01 02:49:35
vendor                   : EMQ Technologies Co., Ltd.
version                  : 4.4.14
type                     : official
customer_type            : 2
expiry                   : false
```

> If the certificate information is not updated, you can wait for a while, the update of the license will be delayed. From the above output results, we can see that the content of the License has been updated, which means that the EMQX Enterprise Edition License has been updated successfully.

:::
::::