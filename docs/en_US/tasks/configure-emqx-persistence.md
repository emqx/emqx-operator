# Configure EMQX persistence

## Task target

- How to configure EMQX 4.x cluster persistence through the persistent field.
- How to configure EMQX 5.x cluster Core node persistence through the volumeClaimTemplates field.

## EMQX cluster persistence configuration

- Configure EMQX cluster

:::: tabs type:card
::: tab v2alpha1

In EMQX 5.0, the nodes in the EMQX cluster can be divided into two roles: core (Core) node and replication (Replicant) node. The Core node is responsible for all write operations in the cluster, and serves as the real data source of the EMQX database [Mria](https://github.com/emqx/mria) to store data such as routing tables, sessions, configurations, alarms, and Dashboard user information. The Replicant node is designed to be stateless and does not participate in the writing of data. Adding or deleting Replicant nodes will not change the redundancy of the cluster data. Therefore, in EMQX CRD, we only support the persistence of Core nodes.

EMQX CRD supports configuration of EMQX cluster Core node persistence through `.spec.coreTemplate.spec.volumeClaimTemplates` field. The semantics and configuration of `.spec.coreTemplate.spec.volumeClaimTemplates` field are consistent with `PersistentVolumeClaimSpec` of Kubernetes, and its configuration can refer to the document: [PersistentVolumeClaimSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#persistentvolumeclaimspec-v1-core).

When the user configures the `.spec.coreTemplate.spec.volumeClaimTemplates` field, EMQX Operator will create a fixed PVC (PersistentVolumeClaim) for each Core node in the EMQX cluster to represent the user's request for persistence. When a Pod is deleted, its corresponding PVC is not automatically cleared. When a Pod is rebuilt, it will automatically match the existing PVC. If you no longer want to use the data of the old cluster, you need to manually clean up the PVC.

PVC expresses the user's request for persistence, and what is responsible for storage is the persistent volume ([PersistentVolume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/), PV), PVC and PV are bound one-to-one through PVC Name. PV is a piece of storage in the cluster, which can be manually prepared according to requirements, or can be dynamically created using storage classes ([StorageClass](https://kubernetes.io/docs/concepts/storage/storage-classes/)) preparation. When a user is no longer using a PV resource, the PVC object can be manually deleted, allowing the PV resource to be recycled. Currently, there are two recycling strategies for PV: Retained (retained) and Deleted (deleted). For details of the recycling strategy, please refer to the document: [Reclaiming](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#reclaiming).

EMQX Operator uses PV to persist the data in the `/opt/emqx/data` directory of the Core node of the EMQX cluster. The data stored in the `/opt/emqx/data` directory of the EMQX Core node mainly includes: routing table, session, configuration, alarm, Dashboard user information and other data.

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
   name: emqx
spec:
   image: emqx/emqx:5.0.9
   imagePullPolicy: IfNotPresent
   coreTemplate:
     spec:
       volumeClaimTemplates:
         storageClassName: standard
         resources:
           requests:
             storage: 20Mi
         accessModes:
         - ReadWriteOnce
       replicas: 3
   replicantTemplate:
     spec:
       replicas: 0
   dashboardServiceTemplate:
     spec:
       type: NodePort
       ports:
         - name: "dashboard-listeners-http-bind"
           protocol: TCP
           port: 18083
           targetPort: 18083
           nodePort: 32016
```

**NOTE**: The storageClassName field indicates the name of the StorageClass. You can use the command `kubectl get storageclass` to get the StorageClass that already exists in the Kubernetes cluster, or you can create a StorageClass according to your own needs. The accessModes field indicates the access mode of the PV. Currently, By default the `ReadWriteOnce` mode is used. For more access modes, please refer to the document: [AccessModes](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#access-modes). The `.spec.dashboardServiceTemplate` field configures the way the EMQX cluster exposes services to the outside world: NodePort, and specifies that the nodePort corresponding to port 18083 of the EMQX Dashboard service is 32016 (the value range of nodePort is: 30000-32767).

:::
::: tab v1beta4

EMQX CRD supports configuring EMQX cluster persistence through the `.spec.persistent` field. The semantics and configuration of the `.spec.persistent` field are consistent with `PersistentVolumeClaimSpec` of Kubernetes, and its configuration can refer to the document: [PersistentVolumeClaimSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#persistentvolumeclaimspec-v1-core).

When the user configures the `.spec.persistent` field, EMQX Operator will create a fixed PVC (PersistentVolumeClaim) for each Pod in the EMQX cluster to represent the user's request for persistence. When a Pod is deleted, its corresponding PVC is not automatically cleared. When a Pod is rebuilt, it will automatically match the existing PVC. If you no longer want to use the data of the old cluster, you need to manually clean up the PVC.

PVC expresses the user's request for persistence, and what is responsible for storage is the persistent volume ([PersistentVolume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/), PV), PVC and PV are bound one-to-one through PVC Name. PV is a piece of storage in the cluster, which can be manually prepared according to requirements, or can be dynamically created using storage classes ([StorageClass](https://kubernetes.io/docs/concepts/storage/storage-classes/)) preparation. When a user is no longer using a PV resource, the PVC object can be manually deleted, allowing the PV resource to be recycled. Currently, there are two recycling strategies for PV: Retained (retained) and Deleted (deleted). For details of the recycling strategy, please refer to the document: [Reclaiming](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#reclaiming).

EMQX Operator uses PV to persist the data in the `/opt/emqx/data` directory of the EMQX node. The data stored in the `/opt/emqx/data` directory of the EMQX node mainly includes: loaded_plugins (loaded plug-in information), loaded_modules (loaded module information), mnesia database data (storing EMQX’s own operating data, such as alarm records, rules resources and rules created by the engine, Dashboard user information and other data).

```yaml
apiVersion: apps.emqx.io/v1beta4
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  persistent:
    storageClassName: standard
    resources:
      requests:
        storage: 20Mi
    accessModes:
    - ReadWriteOnce
  template:
    spec:
      emqxContainer:
        image: 
          repository: emqx/emqx-ee
          version: 4.4.8
  serviceTemplate:
    spec:
      type: NodePort
      ports:
        - name: "http-dashboard-18083"
          protocol: "TCP"
          port: 18083
          targetPort: 18083
          nodePort: 32016
```

**NOTE**: The storageClassName field indicates the name of the StorageClass. You can use the command `kubectl get storageclass` to get the StorageClass that already exists in the Kubernetes cluster, or you can create a StorageClass according to your own needs. The accessModes field indicates the access mode of the PV. Currently, By default the `ReadWriteOnce` mode is used. For more access modes, please refer to the document: [AccessModes](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#access-modes). The `.spec.serviceTemplate` field configures the way the EMQX cluster exposes services to the outside world: NodePort, and specifies that the nodePort corresponding to port 18083 of the EMQX Dashboard service is 32016 (the value range of nodePort is: 30000-32767).

:::
::: tab v1beta3

EMQX CRD supports configuring EMQX cluster persistence through the `.spec.persistent` field. The semantics and configuration of the `.spec.persistent` field are consistent with `PersistentVolumeClaimSpec` of Kubernetes, and its configuration can refer to the document: [PersistentVolumeClaimSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#persistentvolumeclaimspec-v1-core).

When the user configures the `.spec.persistent` field, EMQX Operator will create a fixed PVC (PersistentVolumeClaim) for each Pod in the EMQX cluster to represent the user's request for persistence. When a Pod is deleted, its corresponding PVC is not automatically cleared. When a Pod is rebuilt, it will automatically match the existing PVC. If you no longer want to use the data of the old cluster, you need to manually clean up the PVC.

PVC expresses the user's request for persistence, and what is responsible for storage is the persistent volume ([PersistentVolume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/), PV), PVC and PV are bound one-to-one through PVC Name. PV is a piece of storage in the cluster, which can be manually prepared according to requirements, or can be dynamically created using storage classes ([StorageClass](https://kubernetes.io/docs/concepts/storage/storage-classes/)) preparation. When a user is no longer using a PV resource, the PVC object can be manually deleted, allowing the PV resource to be recycled. Currently, there are two recycling strategies for PV: Retained (retained) and Deleted (deleted). For details of the recycling strategy, please refer to the document: [Reclaiming](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#reclaiming).

EMQX Operator uses PV to persist the data in the `/opt/emqx/data` directory of the EMQX node. The data stored in the `/opt/emqx/data` directory of the EMQX node mainly includes: loaded_plugins (loaded plug-in information), loaded_modules (loaded module information), mnesia database data (storing EMQX’s own operating data, such as alarm records, rules resources and rules created by the engine, Dashboard user information and other data).

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
   name: emqx-ee
spec:
   persistent:
     storageClassName: standard
     resources:
       requests:
         storage: 20Mi
     accessModes:
     - ReadWriteOnce
   emqxTemplate:
     image: emqx/emqx-ee:4.4.8
     serviceTemplate:
       spec:
         type: NodePort
         ports:
           - name: "http-dashboard-18083"
             protocol: "TCP"
             port: 18083
             targetPort: 18083
             nodePort: 32016
```

**NOTE**: The storageClassName field indicates the name of the StorageClass. You can use the command `kubectl get storageclass` to get the StorageClass that already exists in the Kubernetes cluster, or you can create a StorageClass according to your own needs. The accessModes field indicates the access mode of the PV. Currently, By default the `ReadWriteOnce` mode is used. For more access modes, please refer to the document: [AccessModes](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#access-modes). The `.spec.emqxTemplate.serviceTemplate` field configures the way the EMQX cluster exposes services to the outside world: NodePort, and specifies that the nodePort corresponding to port 18083 of the EMQX Dashboard service is 32016 (the value range of nodePort is: 30000-32767).

:::
::::

Save the above content as: emqx-persistent.yaml, execute the following command to deploy the EMQX cluster:

```
kubectl apply -f emqx-persistent.yaml
```

The output is similar to:

```
emqx.apps.emqx.io/emqx created
```

- Check whether the EMQX cluster is ready

:::: tabs type:card
::: tab v2alpha1

```
kubectl get emqx emqx -o json | jq ".status.emqxNodes"
```

The output is similar to:

```
[
   {
     "node": "emqx@emqx-core-0.emqx-headless.default.svc.cluster.local",
     "node_status": "running",
     "otp_release": "24.2.1-1/12.2.1",
     "role": "core",
     "version": "5.0.9"
   },
   {
     "node": "emqx@emqx-core-1.emqx-headless.default.svc.cluster.local",
     "node_status": "running",
     "otp_release": "24.2.1-1/12.2.1",
     "role": "core",
     "version": "5.0.9"
   },
   {
     "node": "emqx@emqx-core-2.emqx-headless.default.svc.cluster.local",
     "node_status": "running",
     "otp_release": "24.2.1-1/12.2.1",
     "role": "core",
     "version": "5.0.9"
   }
]
```

**NOTE**: `node` represents the unique identifier of the EMQX node in the cluster. `node_status` indicates the status of the EMQX node. `otp_release` indicates the version of Erlang used by EMQX. `role` represents the EMQX node role type. `version` indicates the EMQX version. EMQX Operator creates an EMQX cluster with three core nodes and three replicant nodes by default, so when the cluster is running normally, you can see information about three running core nodes and three replicant nodes. If you configure the `.spec.coreTemplate.spec.replicas` field, when the cluster is running normally, the number of running core nodes displayed in the output should be equal to the value of this replicas. If you configure the `.spec.replicantTemplate.spec.replicas` field, when the cluster is running normally, the number of running replicant nodes displayed in the output should be equal to the replicas value.

:::
::: tab v1beta4

```
kubectl get emqxenterprise emqx-ee -o json | jq ".status.emqxNodes"
```

The output is similar to:

```
[
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
   },
   {
     "node": "emqx-ee@emqx-ee-1.emqx-ee-headless.default.svc.cluster.local",
     "node_status": "Running",
     "otp_release": "24.1.5/12.1.5",
     "version": "4.4.8"
   }
]
```

**NOTE**: `node` represents the unique identifier of the EMQX node in the cluster. `node_status` indicates the status of the EMQX node. `otp_release` indicates the version of Erlang used by EMQX. `version` indicates the EMQX version. EMQX Operator will pull up the EMQX cluster with three nodes by default, so when the cluster is running normally, you can see the information of the three running nodes. If you configure the `.spec.replicas` field, when the cluster is running normally, the number of running nodes displayed in the output should be equal to the value of replicas.

:::
::: tab v1beta3

```
kubectl get emqxenterprise emqx-ee -o json | jq ".status.emqxNodes"
```

The output is similar to:

```
[
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
   },
   {
     "node": "emqx-ee@emqx-ee-1.emqx-ee-headless.default.svc.cluster.local",
     "node_status": "Running",
     "otp_release": "24.1.5/12.1.5",
     "version": "4.4.8"
   }
]
```

**NOTE**: `node` represents the unique identifier of the EMQX node in the cluster. `node_status` indicates the status of the EMQX node. `otp_release` indicates the version of Erlang used by EMQX. `version` indicates the EMQX version. EMQX Operator will pull up the EMQX cluster with three nodes by default, so when the cluster is running normally, you can see the information of the three running nodes. If you configure the `.spec.replicas` field, when the cluster is running normally, the number of running nodes displayed in the output should be equal to the value of replicas.

:::
::::

## Verify whether the EMQX cluster persistence is in effect

Verification scheme: 1) Create a test rule through the Dashboard in the old EMQX cluster; 2) Delete the old cluster; 3) Recreate the EMQX cluster, and check whether the previously created rule exists through the Dashboard.

- Create test rules through Dashboard

Open the browser, enter the `IP` of the host where the EMQX Pod is located and the port `32016` to log in to the EMQX cluster Dashboard (Dashboard default username: admin, default password: public), enter the Dashboard and click Data Integration → Rules to enter the creation rule page, we first click the Add Action button to add a response action for this rule, and then click Create to generate a rule, as shown in the following figure:

![](./assets/configure-emqx-persistent/emqx-core-action.png)

When our rule is successfully created, a rule record will appear on the page with the rule ID: emqx-persistent-test, as shown in the figure below:

![](./assets/configure-emqx-persistent/emqx-core-rule-old.png)

- Delete the old EMQX cluster

Execute the following command to delete the EMQX cluster:

```
kubectl delete -f emqx-persistent.yaml
```

**NOTE**: emqx-persistent.yaml is the YAML file used for the first deployment of the EMQX cluster in this article. This file does not need to be changed.

The output is similar to:

```
emqx.apps.emqx.io "emqx" deleted
```

Execute the following command to check whether the EMQX cluster is deleted:

```
kubectl get emqx emqx -o json | jq ".status.emqxNodes"
```

The output is similar to:

```
Error from server (NotFound): emqxes.apps.emqx.io "emqx" not found
```

- Recreate the EMQX cluster

Execute the following command to recreate the EMQX cluster:

```
kubectl apply -f emqx-persistent.yaml
```

The output is similar to:

```
emqx.apps.emqx.io/emqx created
```

Next, execute the following command to check whether the EMQX cluster is ready:

```
kubectl get emqx emqx -o json | jq ".status.emqxNodes"
```

The output is similar to:

```
[
   {
     "node": "emqx@emqx-core-0.emqx-headless.default.svc.cluster.local",
     "node_status": "running",
     "otp_release": "24.2.1-1/12.2.1",
     "role": "core",
     "version": "5.0.9"
   },
   {
     "node": "emqx@emqx-core-1.emqx-headless.default.svc.cluster.local",
     "node_status": "running",
     "otp_release": "24.2.1-1/12.2.1",
     "role": "core",
     "version": "5.0.9"
   },
   {
     "node": "emqx@emqx-core-2.emqx-headless.default.svc.cluster.local",
     "node_status": "running",
     "otp_release": "24.2.1-1/12.2.1",
     "role": "core",
     "version": "5.0.9"
   }
]
```

Finally, visit the EMQX Dashboard through the browser to check whether the previously created rules exist, as shown in the following figure:

![](./assets/configure-emqx-persistent/emqx-core-rule-new.png)

It can be seen from the figure that the rule emqx-persistent-test created in the old cluster still exists in the new cluster, which means that the persistence we configured is in effect.