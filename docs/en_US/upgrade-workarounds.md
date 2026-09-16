# EMQX Upgrade Workarounds

Use these manual workarounds when an EMQX upgrade managed by the operator stalls with the symptoms described below. The operator does not apply them automatically.

## 6.2.x to 6.3.0: API authentication failure

If the upgrade stalls while both 6.2.x and 6.3.0 nodes are running, check the operator logs or the EMQX resource's events. Requests to a 6.2.x node may fail with:

```text
/api/v5/listeners: HTTP 401, response: {"code":"BAD_API_KEY_OR_SECRET","message":"Check api_key/api_secret"}
```

The 6.3.0 node has overwritten shared bootstrap API key records with records that the 6.2.x nodes cannot use.

Find a remaining running 6.2.x core pod:
```sh
kubectl get emqx <emqx-cr-name> -o jsonpath='{range .status.coreNodes[*]}{.podName}{"\t"}{.version}{"\t"}{.status}{"\n"}{end}'
```

Run this command in **one remaining 6.2.x core pod**:

```sh
kubectl exec <emqx-6.2-core-pod> -c emqx -- \
  emqx eval 'emqx_mgmt_auth:try_init_bootstrap_file().'
```

The command should return `ok`. It reloads the mounted bootstrap file and replaces the shared API key records, restoring API access for the older nodes. One core is sufficient because the records are shared across the cluster.

Verify that the operator resumes the rolling update and the EMQX resource returns to `Ready=True`:

```sh
kubectl wait emqx/emqx --for=condition=Ready --timeout=5m
```

All core and replicant pods should finish upgrading to 6.3.0. If another 6.3.0 node overwrites the records while 6.2.x cores remain and the same authentication failure stalls the upgrade again, repeat the command on one remaining 6.2.x core.

* Related upstream issue: [emqx#18851](https://github.com/emqx/emqx/issues/18851).
* Corresponding upgrade test procedure: [`workaround_bootstrap_auth.go`](../../test/e2e/upgrade/workaround_bootstrap_auth.go).
