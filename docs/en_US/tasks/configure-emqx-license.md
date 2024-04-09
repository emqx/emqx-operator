# License Configuration (EMQX Enterprise)

## Task Target

- Configure EMQX Enterprise License.
- Update EMQX Enterprise License.

## Configure License

EMQX Enterprise License can be applied for free on EMQ official website: [Apply for EMQX Enterprise License](https://www.emqx.com/en/apply-licenses/emqx).

The following is the relevant configuration of EMQX Custom Resource. You can choose the corresponding APIVersion according to the version of EMQX you want to deploy. For the specific compatibility relationship, please refer to [EMQX Operator Compatibility](../index.md):

## Configure EMQX Cluster

:::: tabs type:card
::: tab apps.emqx.io/v2beta1

  `apps.emqx.io/v2beta1 EMQX` supports configuring EMQX cluster license through `.spec.config.data`. For config.data configuration, please refer to the document: [Configuration Manual](https://www.emqx.io/docs/en/v5.1/configuration/configuration-manual.html#configuration-manual). This field is only allowed to be configured when creating an EMQX cluster, and does not support updating.

  > After the EMQX cluster is created, if the license needs to be updated, please update it through the EMQX Dashboard.

+ Save the following content as a YAML file and deploy it via the `kubectl apply` command

  ```yaml
  apiVersion: apps.emqx.io/v2beta1
  kind: EMQX
  metadata:
    name: emqx-ee
  spec:
    config:
      data: |
        license {
          key = "..."
        }
    image: emqx/emqx-enterprise:5.6
    dashboardServiceTemplate:
      spec:
        type: LoadBalancer
  ```

  > The `license.key` in the `config.data` field represents the Licesne content. In this example, the License content is omitted, please fill it in by the user.

+ Wait for the EMQX cluster to be ready, you can check the status of the EMQX cluster through `kubectl get` command, please make sure `STATUS` is `Running`, this may take some time

  ```bash
  $ kubectl get emqx emqx-ee
  NAME   IMAGE                        STATUS    AGE
  emqx   emqx/emqx-enterprise:5.1.0   Running   10m
  ```

+ Obtain the Dashboard External IP of EMQX cluster and access EMQX console

  EMQX Operator will create two EMQX Service resources, one is emqx-dashboard and the other is emqx-listeners, corresponding to EMQX console and EMQX listening port respectively.

  ```bash
  $ kubectl get svc emqx-ee-dashboard -o json | jq '.status.loadBalancer.ingress[0].ip'

  192.168.1.200
  ```

  Access `http://192.168.1.200:18083` through a browser, and use the default username and password `admin/public` to login EMQX console.

:::
::: tab apps.emqx.io/v1beta4

+ Create Secret based on License file

  A Secret is an object that contains a small amount of sensitive information such as a password, token, or key. For more detailed documentation on Secret, please refer to: [Secret](https://kubernetes.io/docs/concepts/configuration/secret/). EMQX Operator supports using Secret to mount License information, so we need to create a Secret based on the License before creating an EMQX cluster.

  ```bash
  $ kubectl create secret generic ${your_license_name} --from-file=emqx.lic=${/path/to/license/file}
  ```

  > `${your_license_name}` represents the name of the created Secret.

  > `${/path/to/license/file}` represents the path of the EMQX Enterprise Edition License file, which can be an absolute path or a relative path. For more details on using kubectl to create a Secret, please refer to the document: [Using kubectl to create a secret](https://kubernetes.io/docs/tasks/configmap-secret/managing-secret-using-kubectl/).

+ Save the following content as a YAML file and deploy it via the `kubectl apply` command

  `apps.emqx.io/v1beta4 EmqxEnterprise` supports configuring EMQX Enterprise License through `.spec.license` field. For more information, please refer to: [license](../reference/v1beta4-reference.md#emqxlicense).

  ```yaml
  apiVersion: apps.emqx.io/v1beta4
  kind: EmqxEnterprise
  metadata:
    name: emqx-ee
  spec:
    license:
      secretName: ${your_license_name}
    template:
      spec:
        emqxContainer:
          image:
            repository: emqx/emqx-ee
            version: 4.4.14
    serviceTemplate:
      spec:
        type: LoadBalancer
  ```

  > `secretName` represents the name of the Secret created in the previous step.

+ Wait for the EMQX cluster to be ready, you can check the status of the EMQX cluster through `kubectl get` command, please make sure `STATUS` is `Running`, this may take some time

  ```bash
  $ kubectl get emqxenterprises
  NAME      STATUS   AGE
  emqx-ee   Running  8m33s
  ```

+ Obtain the External IP of EMQX cluster and access EMQX console

  ```bash
  $ kubectl get svc emqx-ee -o json | jq '.status.loadBalancer.ingress[0].ip'

  192.168.1.200
  ```
  Access `http://192.168.1.200:18083` through a browser, and use the default username and password `admin/public` to login EMQX console.

:::
::::

## Update License

:::: tabs type:card
::: tab apps.emqx.io/v2beta1

+ View License information
  ```bash
  $ pod_name="$(kubectl get pods -l 'apps.emqx.io/instance=emqx-ee,apps.emqx.io/db-role=core' -o json | jq --raw-output '.items[0].metadata.name')"
  $ kubectl exec -it ${pod_name} -c emqx -- emqx_ctl license info
  ```

  The following output can be obtained. From the output, we can see the basic information of the license we applied for, including applicant's information, maximum connection supported by the license, and expiration time of the license.
  ```bash
  customer        : Evaluation
  email           : contact@emqx.io
  deployment      : default
  max_connections : 100
  start_at        : 2023-01-09
  expiry_at       : 2028-01-08
  type            : trial
  customer_type   : 10
  expiry          : false
  ```

+ Modify EMQX custom resources to update the License.
  ```bash
  $ kubectl edit emqx emqx-ee
  ...
  spec:
    image: emqx/emqx-enterprise:5.6
    config:
      data: |
        license {
          key = "${new_license_key}"
        }
  ...
  ```

  + Check if the EMQX cluster license has been updated.
  ```bash
  $ pod_name="$(kubectl get pods -l 'apps.emqx.io/instance=emqx-ee,apps.emqx.io/db-role=core' -o json | jq --raw-output '.items[0].metadata.name')"
  $ kubectl exec -it ${pod_name} -c emqx -- emqx_ctl license info
  ```

  It can be seen from the "max_connections" field that the content of the License has been updated, indicating that the EMQX Enterprise Edition License update is successful. If the certificate information is not updated, you can wait for a while as there may be some delay in updating the License.
  ```bash
  customer        : Evaluation
  email           : contact@emqx.io
  deployment      : default
  max_connections : 100000
  start_at        : 2023-01-09
  expiry_at       : 2028-01-08
  type            : trial
  customer_type   : 10
  expiry          : false
  ```
:::
::: tab apps.emqx.io/v1beta4
+ View License Information

  ```bash
  $ kubectl exec -it emqx-ee-core-0 -c emqx -- emqx_ctl license info
  ```

  The following output can be obtained. From the output results, we can see the basic information of the license we applied for, including the applicant's information, the maximum number of connections supported by the license, and the expiration time of the license.

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

+ Update EMQX Enterprise License Secret

  ```bash
  $ kubectl create secret generic ${your_license_name} --from-file=emqx.lic=${/path/to/license/file} --dry-run -o yaml | kubectl apply -f -
  ```

+ Check whether the EMQX cluster license has been updated

  ```bash
  $ kubectl exec -it emqx-ee-0 -c emqx -- emqx_ctl license info
  ```

  You can get information similar to the following. From the `max_connections` field, you can see that the content of the License has been updated, which means that the EMQX Enterprise Edition License has been updated successfully. If the certificate information is not updated, you can wait for a while, the update of the license will be delayed.

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
:::
::::
