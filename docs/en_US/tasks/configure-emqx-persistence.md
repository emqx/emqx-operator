# Configure EMQX persistence

## Task target
- Learn how to configure EMQX cluster persistence through the persistent field

## EMQX persistent configuration
EMQX CRD supports configuring EMQX cluster persistence by configuring the `.spec.persistent` field. For the specific description of the persistent field, please refer to: [persistent field](https://github.com/emqx/emqx-operator/blob/1.2.8/docs/en_US/reference/v1beta3-reference.md#servicetemplate), The semantics and configuration of persistent are consistent with Kubernetes' PersistentVolumeClaimSpec. PersistentVolumeClaimSpec can be used by referring to the document: [PVC documentation](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#persistentvolumeclaimspec-v1-core)

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  persistent:
      storageClassName: nfs-client
      resources:
        requests:
          storage: 20Mi
      accessModes:
      - ReadWriteOnce
  emqxTemplate:
    image: emqx/emqx-ee:4.4.8
```
Save the above content as emqx-persistent.yaml and deploy the EMQX cluster

```
kubectl apply -f emqx-persistent.yaml
```

Obtain pvc information created for EMQX:

```
kubectl get pvc -l apps.emqx.io/instance=emqx-ee
```

The output is similar to:

```
NAME                     STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
emqx-ee-data-emqx-ee-0   Bound    pvc-b00dffe9-56de-4947-b64b-7e14c5aae4e7   20Mi       RWO            nfs-client     54s
emqx-ee-data-emqx-ee-1   Bound    pvc-67d1f049-1f4f-4340-b73d-809fbae5b252   20Mi       RWO            nfs-client     54s
emqx-ee-data-emqx-ee-2   Bound    pvc-f3e3dfa4-03d6-47ce-85da-83d0d25bc4b0   20Mi       RWO            nfs-client     54s
 ```

