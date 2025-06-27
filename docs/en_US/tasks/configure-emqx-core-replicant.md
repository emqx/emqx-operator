# Enable Core + Replicant Cluster (EMQX 5.x)

## Task Target

- Configure EMQX cluster Core node through `coreTemplate` field.
- Configure EMQX cluster Replicant node through `replicantTemplate` field.

## Core Nodes And Replicant Nodes

:::tip
Just EMQX Enterprise Edition supports Core + Replicant cluster.
:::

In EMQX 5.0, the nodes in the EMQX cluster can be divided into two roles: core (Core) node and replication (Replicant) node. The Core node is responsible for all write operations in the cluster, which is consistent with the behavior of the nodes in the EMQX 4.x cluster, and serves as the real data source of the EMQX database [Mria](https://github.com/emqx/mria) to store the routing table, Data such as sessions, configurations, alarms, and Dashboard user information. The Replicant node is designed to be stateless and does not participate in the writing of data. Adding or deleting Replicant nodes will not change the redundancy of the cluster data. For more information about the EMQX 5.0 architecture, please refer to the document: [EMQX 5.0 Architecture](https://docs.emqx.com/en/enterprise/v5.0/deploy/cluster/mria-introduction.html), the topological structure of the Core node and the Replicant node is shown in the following figure:

  <div style="text-align:center">
  <img src="./assets/configure-core-replicant/mria-core-repliant.png" style="zoom:30%;" />
  </div>

:::tip
There must be at least one Core node in the EMQX cluster. For the purpose of high availability, EMQX Operator recommends that the EMQX cluster have at least three Core nodes.
:::

## Configure EMQX Cluster

`apps.emqx.io/v2beta1 EMQX` supports configuring the Core node of the EMQX cluster through the `.spec.coreTemplate` field, and configuring the Replicant node of the EMQX cluster using the `.spec.replicantTemplate` field. For more information, please refer to: [API Reference](../reference/v2beta1-reference.md#emqxspec).

+ Save the following content as a YAML file and deploy it with the `kubectl apply` command

  ```yaml
  apiVersion: apps.emqx.io/v2beta1
  kind: EMQX
  metadata:
    name: emqx
  spec:
    image: emqx/emqx-enterprise:5.10
    config:
      data: |
        license {
          key = "..."
        }
    coreTemplate:
      spec:
        replicas: 2
        resources:
          requests:
            cpu: 250m
            memory: 512Mi
    replicantTemplate:
      spec:
        replicas: 3
        resources:
          requests:
            cpu: 250m
            memory: 1Gi
    dashboardServiceTemplate:
      spec:
        type: LoadBalancer
  ```

  > In the YAML above, we declared that this is an EMQX cluster consisting of two Core nodes and three Replicant nodes. Core nodes require a minimum of 512Mi of memory, and Replicant nodes require a minimum of 1Gi of memory. You can adjust according to the actual business load. In actual business, the Replicant node will accept all client requests, so the resources required by the Replicant node will be higher.

+ Wait for the EMQX cluster to be ready, you can check the status of EMQX cluster through `kubectl get` command, please make sure `STATUS` is `Running`, this may take some time

  ```bash
  $ kubectl get emqx emqx
  NAME   IMAGE                         STATUS    AGE
  emqx   emqx/emqx-enterprise:5.10.0   Running   10m
  ```

+ Obtain the Dashboard External IP of EMQX cluster and access EMQX console

  EMQX Operator will create two EMQX Service resources, one is emqx-dashboard and the other is emqx-listeners, corresponding to EMQX console and EMQX listening port respectively.

  ```bash
  $ kubectl get svc emqx-dashboard -o json | jq '.status.loadBalancer.ingress[0].ip'

  192.168.1.200
  ```

  Access `http://192.168.1.200:18083` through a browser, and use the default username and password `admin/public` to login EMQX console.

## Verify EMQX Cluster

  Information about all the nodes in the cluster can be obtained by checking the `.status` of the EMQX custom resources.

  ```bash
  $ kubectl get emqx emqx -o json | jq .status.coreNodes
  [
    {
      "node": "emqx@emqx-core-0.emqx-headless.default.svc.cluster.local",
      "node_status": "running",
      "otp_release": "27.2-3/15.2",
      "role": "core",
      "version": "5.10.0"
    },
    {
      "node": "emqx@emqx-core-1.emqx-headless.default.svc.cluster.local",
      "node_status": "running",
      "otp_release": "27.2-3/15.2",
      "role": "core",
      "version": "5.10.0"
    },
     {
      "node": "emqx@emqx-core-2.emqx-headless.default.svc.cluster.local",
      "node_status": "running",
      "otp_release": "27.2-3/15.2",
      "role": "core",
      "version": "5.10.0"
    }
  ]
  ```


  ```bash
  $ kubectl get emqx emqx -o json | jq .status.replicantNodes
  [
    {
      "node": "emqx@10.244.4.56",
      "node_status": "running",
      "otp_release": "27.2-3/15.2",
      "role": "replicant",
      "version": "5.10.0"
    },
    {
      "node": "emqx@10.244.4.57",
      "node_status": "running",
      "otp_release": "27.2-3/15.2",
      "role": "replicant",
      "version": "5.10.0"
    },
    {
      "node": "emqx@10.244.4.58",
      "node_status": "running",
      "otp_release": "27.2-3/15.2",
      "role": "replicant",
      "version": "5.10.0"
    }
  ]
  ```
