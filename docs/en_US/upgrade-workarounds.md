# EMQX Upgrade Workarounds

Certain EMQX version upgrades may get stuck during a rolling update performed
by the operator. This document lists known affected version pairs and the exact
steps to unblock the upgrade.

## 6.1.0 to 6.2.0

### Preparation

We advise to set up a single EMQX configuration override via `coreTemplate`
before triggering the upgrade. This helps the Durable Storage to recover
availability quickly during the rolling update:

```yaml
apiVersion: apps.emqx.io/v2
kind: EMQX
...
spec:
  image: emqx/emqx:6.1.0
  coreTemplate:
    spec:
      args: ["/opt/emqx/bin/emqx", "foreground",
              "-ra", "machine_upgrade_strategy", "quorum"]
```

Without this, in rare circumstances the availability will be lost until
core nodes are restarted, either manually or automatically.

### Symptom

After the operator patches the core StatefulSet image to 6.2.0 and the new
pods start:

* `emqx ctl ds info` shows one or more databases with shard transitions that
  never finish (the transition count stays non-zero indefinitely).
* The operator logs contain repeated errors like
  `failed to update DB <name> replica set`.
* The EMQX CR never transitions back to `Ready=True`.

The root cause is that `emqx_mq_message_db` and `emqx_mq_state_storage`
databases are not automatically opened on new 6.2.0 nodes when they join a
cluster that was originally created on 6.1.0.

### Manual Remediation

Exec into **every new core pod** (the pods running the 6.2.0 image) and open
the missing databases:

```bash
$ kubectl exec <pod> -- emqx eval 'emqx_mq_message_db:open()'
ok
$ kubectl exec <pod> -- emqx eval 'emqx_mq_state_storage:open_db()'
ok
```

Repeat for each core pod that was recreated during the rolling upgrade. In a
two-node core cluster both pods will be recreated, so both need the commands.

Each command is idempotent, re-run freely if needed.

### Verification

After running the commands on all affected pods:

```bash
kubectl exec service/emqx-listeners -- emqx ctl ds info
```

All databases should show zero in-progress transitions. The operator will then
be able to update the DS replica sets and the EMQX CR will become `Ready=True`.
