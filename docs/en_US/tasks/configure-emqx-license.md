# Manage EMQX Enterprise License

## Objective

- Configure EMQX Enterprise license.
- Update EMQX Enterprise license.

## Configure License

You can apply for a EMQX Enterprise license for free on the EMQX official website: [Apply for EMQX Enterprise License](https://www.emqx.com/en/apply-licenses/emqx).

## Configure EMQX Cluster

EMQX CRD `apps.emqx.io/v2` supports configuring the EMQX cluster license through the `.spec.config.data` field. Refer to the [Configuration Manual](https://docs.emqx.com/en/enterprise/v6.0.0/hocon/) for complete configuration reference.

+ Save the following as a YAML file and deploy it using `kubectl apply`.

  ```yaml
  apiVersion: apps.emqx.io/v2
  kind: EMQX
  metadata:
    name: emqx-ee
  spec:
    config:
      data: |
        license {
          key = "..."
        }
    image: emqx/emqx:6.0
    dashboardServiceTemplate:
      spec:
        type: LoadBalancer
  ```

  ::: tip
  The `license.key` in the `.spec.config.data` field represents the license content. In this example, the license content is omitted. Please fill it in with your own license key.
  :::

+ Wait for the EMQX cluster to become ready.

  Check the status of the EMQX cluster with `kubectl get` and ensure that `STATUS` is `Ready`. This may take some time.

  ```bash
  $ kubectl get emqx emqx-ee
  NAME   STATUS   AGE
  emqx   Ready    10m
  ```

## Update License

+ View the license information.

  ```bash
  $ pod_name="$(kubectl get pods -l 'apps.emqx.io/instance=emqx-ee,apps.emqx.io/db-role=core' -o json | jq --raw-output '.items[0].metadata.name')"
  $ kubectl exec -it ${pod_name} -c emqx -- emqx ctl license info
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

  The output shows basic license information, including the applicant's information, the maximum number of connections supported by the license, and the expiration time.

+ Modify the EMQX CR to update the license.

  ```bash
  $ kubectl edit emqx emqx-ee
  ...
  spec:
    image: emqx/emqx:6.0
    config:
      data: |
        license {
          key = "${new_license_key}"
        }
  ...
  ```

+ Verify that the license has been updated.

  ```bash
  $ pod_name="$(kubectl get pods -l 'apps.emqx.io/instance=emqx-ee,apps.emqx.io/db-role=core' -o json | jq --raw-output '.items[0].metadata.name')"
  $ kubectl exec -it ${pod_name} -c emqx -- emqx ctl license info
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

  This output indicates that the EMQX Enterprise license has been updated successfully. Keep in mind that license update may take time, so you may need to retry the command.
