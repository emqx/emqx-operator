# 配置 EMQX 持久化

## 任务目标
- 学习如何通过 persistent 字段配置 EMQX 集群持久化

## EMQX 持久化配置
EMQX CRD 支持通过来配置 `.spec.persistent` 字段来配置 EMQX 集群持久化， persistent 字段的具体描述可以参考：[persistent 字段](https://github.com/emqx/emqx-operator/blob/1.2.8/docs/en_US/reference/v1beta3-reference.md#servicetemplate)，persistent 的语义及配置与 Kubernetes 的 PersistentVolumeClaimSpec 一致，PersistentVolumeClaimSpec 使用可以参考文档：[PVC 使用文档](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#persistentvolumeclaimspec-v1-core)

``` yaml
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
将上述内容保存为 emqx-persistent.yaml 并部署 EMQX 集群

```
kubectl apply -f emqx-persistent.yaml
```

- 获取为 EMQX 创建的 pvc  信息：

```
kubectl get pvc -l apps.emqx.io/instance=emqx-ee
```

输出类似于：

```
NAME                     STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
emqx-ee-data-emqx-ee-0   Bound    pvc-b00dffe9-56de-4947-b64b-7e14c5aae4e7   20Mi       RWO            nfs-client     54s
emqx-ee-data-emqx-ee-1   Bound    pvc-67d1f049-1f4f-4340-b73d-809fbae5b252   20Mi       RWO            nfs-client     54s
emqx-ee-data-emqx-ee-2   Bound    pvc-f3e3dfa4-03d6-47ce-85da-83d0d25bc4b0   20Mi       RWO            nfs-client     54s
```