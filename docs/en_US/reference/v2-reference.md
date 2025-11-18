# API Reference

## Packages
- [apps.emqx.io/v2](#appsemqxiov2)


## apps.emqx.io/v2

package v2 contains API Schema definitions for the apps v2 API group.

### Resource Types
- [EMQX](#emqx)



#### BootstrapAPIKey







_Appears in:_
- [EMQXSpec](#emqxspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `key` _string_ |  |  | Pattern: `^[a-zA-Z\d-_]+$` <br /> |
| `secret` _string_ |  |  | MaxLength: 128 <br />MinLength: 3 <br /> |
| `secretRef` _[SecretRef](#secretref)_ | Reference to a Secret entry containing the EMQX API Key. |  |  |


#### Config







_Appears in:_
- [EMQXSpec](#emqxspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `mode` _string_ | Determines how configuration updates are applied.<br />* `Merge`: Merge the new configuration into the existing configuration.<br />* `Replace`: Replace the whole configuration. | Merge | Enum: [Merge Replace] <br /> |
| `data` _string_ | EMQX configuration, in HOCON format.<br />This configuration will be supplied as `base.hocon` to the container. See respective<br />[documentation](https://docs.emqx.com/en/emqx/latest/configuration/configuration.html#base-configuration-file). |  |  |


#### DSDBReplicationStatus







_Appears in:_
- [DSReplicationStatus](#dsreplicationstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the database |  |  |
| `numShards` _integer_ | Number of shards of the database |  |  |
| `numShardReplicas` _integer_ | Total number of shard replicas |  |  |
| `lostShardReplicas` _integer_ | Total number of shard replicas belonging to lost sites |  |  |
| `numTransitions` _integer_ | Current number of shard ownership transitions |  |  |
| `minReplicas` _integer_ | Minimum replication factor among database shards |  |  |
| `maxReplicas` _integer_ | Maximum replication factor among database shards |  |  |


#### DSReplicationStatus



Summary of DS replication status per database.



_Appears in:_
- [EMQXStatus](#emqxstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `dbs` _[DSDBReplicationStatus](#dsdbreplicationstatus) array_ |  |  |  |


#### EMQX



Custom Resource representing an EMQX cluster.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `apps.emqx.io/v2` | | |
| `kind` _string_ | `EMQX` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[EMQXSpec](#emqxspec)_ | Specification of the desired state of the EMQX cluster. |  |  |
| `status` _[EMQXStatus](#emqxstatus)_ | Current status of the EMQX cluster. |  |  |


#### EMQXCoreTemplate







_Appears in:_
- [EMQXSpec](#emqxspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[EMQXCoreTemplateSpec](#emqxcoretemplatespec)_ | Specification of the desired state of a core node.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status |  |  |


#### EMQXCoreTemplateSpec







_Appears in:_
- [EMQXCoreTemplate](#emqxcoretemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `nodeSelector` _object (keys:string, values:string)_ | Selector which must be true for the pod to fit on a node.<br />Must match a node's labels for the pod to be scheduled on that node.<br />More info: https://kubernetes.io/docs/concepts/config/assign-pod-node/ |  |  |
| `nodeName` _string_ | Request to schedule this pod onto a specific node.<br />If it is non-empty, the scheduler simply schedules this pod onto that node, assuming that it fits resource requirements. |  |  |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#affinity-v1-core)_ | Affinity for pod assignment<br />ref: https://kubernetes.io/docs/concepts/config/assign-pod-node/#affinity-and-anti-affinity |  |  |
| `tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#toleration-v1-core) array_ | Pod tolerations.<br />If specified, Pod tolerates any taint that matches the triple <key,value,effect> using the matching operator. |  |  |
| `topologySpreadConstraints` _[TopologySpreadConstraint](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#topologyspreadconstraint-v1-core) array_ | Specifies how to spread matching pods among the given topology. |  |  |
| `replicas` _integer_ | Desired number of instances.<br />In case of core nodes, each instance has a consistent identity. | 2 | Minimum: 0 <br /> |
| `minAvailable` _[IntOrString](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#intorstring-intstr-util)_ | An eviction is allowed if at least "minAvailable" pods selected by<br />"selector" will still be available after the eviction, i.e. even in the<br />absence of the evicted pod.  So for example you can prevent all voluntary<br />evictions by specifying "100%". |  | XIntOrString: \{\} <br /> |
| `maxUnavailable` _[IntOrString](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#intorstring-intstr-util)_ | An eviction is allowed if at most "maxUnavailable" pods selected by<br />"selector" are unavailable after the eviction, i.e. even in absence of<br />the evicted pod. For example, one can prevent all voluntary evictions<br />by specifying 0. This is a mutually exclusive setting with "minAvailable". |  | XIntOrString: \{\} <br /> |
| `command` _string array_ | Entrypoint array. Not executed within a shell.<br />The container image's ENTRYPOINT is used if this is not provided.<br />Variable references `$(VAR_NAME)` are expanded using the container's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double `$$` are reduced<br />to a single `$`, which allows for escaping the `$(VAR_NAME)` syntax: i.e. `$$(VAR_NAME)` will<br />produce the string literal `$(VAR_NAME)`. Escaped references will never be expanded, regardless<br />of whether the variable exists or not. Cannot be updated.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell |  |  |
| `args` _string array_ | Arguments to the entrypoint.<br />The container image's CMD is used if this is not provided.<br />Variable references `$(VAR_NAME)` are expanded using the container's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double `$$` are reduced<br />to a single `$`, which allows for escaping the `$(VAR_NAME)` syntax: i.e. `$$(VAR_NAME)` will<br />produce the string literal `$(VAR_NAME)`. Escaped references will never be expanded, regardless<br />of whether the variable exists or not.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell |  |  |
| `ports` _[ContainerPort](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#containerport-v1-core) array_ | List of ports to expose from the container.<br />Exposing a port here gives the system additional information about the network connections a<br />container uses, but is primarily informational. Not specifying a port here DOES NOT prevent that<br />port from being exposed. Any port which is listening on the default `0.0.0.0` address inside a<br />container will be accessible from the network. |  |  |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#envvar-v1-core) array_ | List of environment variables to set in the container. |  |  |
| `envFrom` _[EnvFromSource](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#envfromsource-v1-core) array_ | List of sources to populate environment variables from in the container.<br />The keys defined within a source must be a C_IDENTIFIER. All invalid keys<br />will be reported as an event when the container is starting. When a key exists in multiple<br />sources, the value associated with the last source will take precedence.<br />Values defined by an Env with a duplicate key will take precedence. |  |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#resourcerequirements-v1-core)_ | Compute Resources required by this container.<br />More info: https://kubernetes.io/docs/concepts/config/manage-resources-containers/ |  |  |
| `podSecurityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#podsecuritycontext-v1-core)_ | Pod-level security attributes and common container settings. | \{ fsGroup:1000 fsGroupChangePolicy:Always runAsGroup:1000 runAsUser:1000 supplementalGroups:[1000] \} |  |
| `containerSecurityContext` _[SecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#securitycontext-v1-core)_ | Security options the container should be run with.<br />If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.<br />More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/ | \{ runAsGroup:1000 runAsNonRoot:true runAsUser:1000 \} |  |
| `initContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#container-v1-core) array_ | List of initialization containers belonging to the pod.<br />Init containers are executed in order prior to containers being started. If any<br />init container fails, the pod is considered to have failed and is handled according<br />to its restartPolicy. The name for an init container or normal container must be<br />unique among all containers.<br />Init containers may not have Lifecycle actions, Readiness probes, Liveness probes, or Startup probes.<br />The resourceRequirements of an init container are taken into account during scheduling<br />by finding the highest request/limit for each resource type, and then using the max of<br />of that value or the sum of the normal containers. Limits are applied to init containers<br />in a similar fashion.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/init-containers/ |  |  |
| `extraContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#container-v1-core) array_ | Additional containers to run alongside the main container. |  |  |
| `extraVolumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#volume-v1-core) array_ | Additional volumes to provide to a Pod. |  |  |
| `extraVolumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#volumemount-v1-core) array_ | Specifies how additional volumes are mounted into the main container. |  |  |
| `livenessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#probe-v1-core)_ | Periodic probe of container liveness.<br />Container will be restarted if the probe fails.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes | \{ failureThreshold:3 httpGet:map[path:/status port:dashboard] initialDelaySeconds:60 periodSeconds:30 \} |  |
| `readinessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#probe-v1-core)_ | Periodic probe of container service readiness.<br />Container will be removed from service endpoints if the probe fails.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes | \{ failureThreshold:12 httpGet:map[path:/status port:dashboard] initialDelaySeconds:10 periodSeconds:5 \} |  |
| `startupProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#probe-v1-core)_ | StartupProbe indicates that the Pod has successfully initialized.<br />If specified, no other probes are executed until this completes successfully.<br />If this probe fails, the Pod will be restarted, just as if the `livenessProbe` failed.<br />This can be used to provide different probe parameters at the beginning of a Pod's lifecycle,<br />when it might take a long time to load data or warm a cache, than during steady-state operation.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes |  |  |
| `lifecycle` _[Lifecycle](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#lifecycle-v1-core)_ | Actions that the management system should take in response to container lifecycle events. |  |  |
| `volumeClaimTemplates` _[PersistentVolumeClaimSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#persistentvolumeclaimspec-v1-core)_ | PVC specification for a core node data storage.<br />Note: this field named inconsistently, it is actually just a `PersistentVolumeClaimSpec`. |  |  |


#### EMQXNode







_Appears in:_
- [EMQXStatus](#emqxstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Node name |  |  |
| `podName` _string_ | Corresponding pod name |  |  |
| `status` _string_ | Node status |  |  |
| `otpRelease` _string_ | Erlang/OTP version node is running on |  |  |
| `version` _string_ | EMQX version |  |  |
| `role` _string_ | Node role, either "core" or "replicant" |  |  |
| `sessions` _integer_ | Number of MQTT sessions |  |  |
| `connections` _integer_ | Number of connected MQTT clients |  |  |


#### EMQXNodesStatus







_Appears in:_
- [EMQXStatus](#emqxstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `replicas` _integer_ | Total number of replicas. |  |  |
| `readyReplicas` _integer_ | Number of ready replicas. |  |  |
| `currentRevision` _string_ | Current revision of the respective core or replicant set. |  |  |
| `currentReplicas` _integer_ | Number of replicas running current revision. |  |  |
| `updateRevision` _string_ | Update revision of the respective core or replicant set.<br />When different from the current revision, the set is being updated. |  |  |
| `updateReplicas` _integer_ | Number of replicas running update revision. |  |  |
| `collisionCount` _integer_ |  |  |  |


#### EMQXReplicantTemplate







_Appears in:_
- [EMQXSpec](#emqxspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[EMQXReplicantTemplateSpec](#emqxreplicanttemplatespec)_ | Specification of the desired state of a replicant node.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status |  |  |


#### EMQXReplicantTemplateSpec







_Appears in:_
- [EMQXCoreTemplateSpec](#emqxcoretemplatespec)
- [EMQXReplicantTemplate](#emqxreplicanttemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `nodeSelector` _object (keys:string, values:string)_ | Selector which must be true for the pod to fit on a node.<br />Must match a node's labels for the pod to be scheduled on that node.<br />More info: https://kubernetes.io/docs/concepts/config/assign-pod-node/ |  |  |
| `nodeName` _string_ | Request to schedule this pod onto a specific node.<br />If it is non-empty, the scheduler simply schedules this pod onto that node, assuming that it fits resource requirements. |  |  |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#affinity-v1-core)_ | Affinity for pod assignment<br />ref: https://kubernetes.io/docs/concepts/config/assign-pod-node/#affinity-and-anti-affinity |  |  |
| `tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#toleration-v1-core) array_ | Pod tolerations.<br />If specified, Pod tolerates any taint that matches the triple <key,value,effect> using the matching operator. |  |  |
| `topologySpreadConstraints` _[TopologySpreadConstraint](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#topologyspreadconstraint-v1-core) array_ | Specifies how to spread matching pods among the given topology. |  |  |
| `replicas` _integer_ | Desired number of instances.<br />In case of core nodes, each instance has a consistent identity. | 2 | Minimum: 0 <br /> |
| `minAvailable` _[IntOrString](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#intorstring-intstr-util)_ | An eviction is allowed if at least "minAvailable" pods selected by<br />"selector" will still be available after the eviction, i.e. even in the<br />absence of the evicted pod.  So for example you can prevent all voluntary<br />evictions by specifying "100%". |  | XIntOrString: \{\} <br /> |
| `maxUnavailable` _[IntOrString](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#intorstring-intstr-util)_ | An eviction is allowed if at most "maxUnavailable" pods selected by<br />"selector" are unavailable after the eviction, i.e. even in absence of<br />the evicted pod. For example, one can prevent all voluntary evictions<br />by specifying 0. This is a mutually exclusive setting with "minAvailable". |  | XIntOrString: \{\} <br /> |
| `command` _string array_ | Entrypoint array. Not executed within a shell.<br />The container image's ENTRYPOINT is used if this is not provided.<br />Variable references `$(VAR_NAME)` are expanded using the container's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double `$$` are reduced<br />to a single `$`, which allows for escaping the `$(VAR_NAME)` syntax: i.e. `$$(VAR_NAME)` will<br />produce the string literal `$(VAR_NAME)`. Escaped references will never be expanded, regardless<br />of whether the variable exists or not. Cannot be updated.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell |  |  |
| `args` _string array_ | Arguments to the entrypoint.<br />The container image's CMD is used if this is not provided.<br />Variable references `$(VAR_NAME)` are expanded using the container's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double `$$` are reduced<br />to a single `$`, which allows for escaping the `$(VAR_NAME)` syntax: i.e. `$$(VAR_NAME)` will<br />produce the string literal `$(VAR_NAME)`. Escaped references will never be expanded, regardless<br />of whether the variable exists or not.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell |  |  |
| `ports` _[ContainerPort](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#containerport-v1-core) array_ | List of ports to expose from the container.<br />Exposing a port here gives the system additional information about the network connections a<br />container uses, but is primarily informational. Not specifying a port here DOES NOT prevent that<br />port from being exposed. Any port which is listening on the default `0.0.0.0` address inside a<br />container will be accessible from the network. |  |  |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#envvar-v1-core) array_ | List of environment variables to set in the container. |  |  |
| `envFrom` _[EnvFromSource](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#envfromsource-v1-core) array_ | List of sources to populate environment variables from in the container.<br />The keys defined within a source must be a C_IDENTIFIER. All invalid keys<br />will be reported as an event when the container is starting. When a key exists in multiple<br />sources, the value associated with the last source will take precedence.<br />Values defined by an Env with a duplicate key will take precedence. |  |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#resourcerequirements-v1-core)_ | Compute Resources required by this container.<br />More info: https://kubernetes.io/docs/concepts/config/manage-resources-containers/ |  |  |
| `podSecurityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#podsecuritycontext-v1-core)_ | Pod-level security attributes and common container settings. | \{ fsGroup:1000 fsGroupChangePolicy:Always runAsGroup:1000 runAsUser:1000 supplementalGroups:[1000] \} |  |
| `containerSecurityContext` _[SecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#securitycontext-v1-core)_ | Security options the container should be run with.<br />If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.<br />More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/ | \{ runAsGroup:1000 runAsNonRoot:true runAsUser:1000 \} |  |
| `initContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#container-v1-core) array_ | List of initialization containers belonging to the pod.<br />Init containers are executed in order prior to containers being started. If any<br />init container fails, the pod is considered to have failed and is handled according<br />to its restartPolicy. The name for an init container or normal container must be<br />unique among all containers.<br />Init containers may not have Lifecycle actions, Readiness probes, Liveness probes, or Startup probes.<br />The resourceRequirements of an init container are taken into account during scheduling<br />by finding the highest request/limit for each resource type, and then using the max of<br />of that value or the sum of the normal containers. Limits are applied to init containers<br />in a similar fashion.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/init-containers/ |  |  |
| `extraContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#container-v1-core) array_ | Additional containers to run alongside the main container. |  |  |
| `extraVolumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#volume-v1-core) array_ | Additional volumes to provide to a Pod. |  |  |
| `extraVolumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#volumemount-v1-core) array_ | Specifies how additional volumes are mounted into the main container. |  |  |
| `livenessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#probe-v1-core)_ | Periodic probe of container liveness.<br />Container will be restarted if the probe fails.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes | \{ failureThreshold:3 httpGet:map[path:/status port:dashboard] initialDelaySeconds:60 periodSeconds:30 \} |  |
| `readinessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#probe-v1-core)_ | Periodic probe of container service readiness.<br />Container will be removed from service endpoints if the probe fails.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes | \{ failureThreshold:12 httpGet:map[path:/status port:dashboard] initialDelaySeconds:10 periodSeconds:5 \} |  |
| `startupProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#probe-v1-core)_ | StartupProbe indicates that the Pod has successfully initialized.<br />If specified, no other probes are executed until this completes successfully.<br />If this probe fails, the Pod will be restarted, just as if the `livenessProbe` failed.<br />This can be used to provide different probe parameters at the beginning of a Pod's lifecycle,<br />when it might take a long time to load data or warm a cache, than during steady-state operation.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes |  |  |
| `lifecycle` _[Lifecycle](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#lifecycle-v1-core)_ | Actions that the management system should take in response to container lifecycle events. |  |  |


#### EMQXSpec



EMQXSpec defines the desired state of EMQX.



_Appears in:_
- [EMQX](#emqx)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `image` _string_ | EMQX container image.<br />More info: https://kubernetes.io/docs/concepts/containers/images |  |  |
| `imagePullPolicy` _[PullPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#pullpolicy-v1-core)_ | Container image pull policy.<br />One of `Always`, `Never`, `IfNotPresent`.<br />Defaults to `Always` if `:latest` tag is specified, or `IfNotPresent` otherwise.<br />More info: https://kubernetes.io/docs/concepts/containers/images#updating-images |  |  |
| `imagePullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#localobjectreference-v1-core) array_ | ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.<br />If specified, these secrets will be passed to individual puller implementations for them to use.<br />More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod |  |  |
| `serviceAccountName` _string_ | ServiceAccount name.<br />Managed ReplicaSets and StatefulSets are associated with the specified ServiceAccount for authentication purposes.<br />More info: https://kubernetes.io/docs/concepts/security/service-accounts |  |  |
| `bootstrapAPIKeys` _[BootstrapAPIKey](#bootstrapapikey) array_ | Bootstrap API keys to access EMQX API.<br />Cannot be updated. |  |  |
| `config` _[Config](#config)_ | EMQX Configuration. |  |  |
| `clusterDomain` _string_ | Kubernetes cluster domain. | cluster.local |  |
| `revisionHistoryLimit` _integer_ | Number of old ReplicaSets, old StatefulSets and old PersistentVolumeClaims to retain to allow rollback. | 3 |  |
| `updateStrategy` _[UpdateStrategy](#updatestrategy)_ | Cluster upgrade strategy settings. | \{ type:Recreate \} |  |
| `coreTemplate` _[EMQXCoreTemplate](#emqxcoretemplate)_ | Template for Pods running EMQX core nodes. | \{ spec:map[replicas:2] \} |  |
| `replicantTemplate` _[EMQXReplicantTemplate](#emqxreplicanttemplate)_ | Template for Pods running EMQX replicant nodes. |  |  |
| `dashboardServiceTemplate` _[ServiceTemplate](#servicetemplate)_ | Template for Service exposing the EMQX Dashboard.<br />Dashboard Service always points to the set of EMQX core nodes. |  |  |
| `listenersServiceTemplate` _[ServiceTemplate](#servicetemplate)_ | Template for Service exposing enabled EMQX listeners.<br />Listeners Service points to the set of EMQX replicant nodes if they are enabled and exist.<br />Otherwise, it points to the set of EMQX core nodes. |  |  |


#### EMQXStatus



EMQXStatus defines the observed state of EMQX



_Appears in:_
- [EMQX](#emqx)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ | Conditions representing the current status of the EMQX Custom Resource. |  |  |
| `coreNodes` _[EMQXNode](#emqxnode) array_ | Status of each core node in the cluster. |  |  |
| `coreNodesStatus` _[EMQXNodesStatus](#emqxnodesstatus)_ | Summary status of the set of core nodes. |  |  |
| `replicantNodes` _[EMQXNode](#emqxnode) array_ | Status of each replicant node in the cluster. |  |  |
| `replicantNodesStatus` _[EMQXNodesStatus](#emqxnodesstatus)_ | Summary status of the set of replicant nodes. |  |  |
| `nodeEvacuationsStatus` _[NodeEvacuationStatus](#nodeevacuationstatus) array_ | Status of active node evacuations in the cluster. |  |  |
| `dsReplication` _[DSReplicationStatus](#dsreplicationstatus)_ | Status of EMQX Durable Storage replication. |  |  |


#### EvacuationStrategy







_Appears in:_
- [UpdateStrategy](#updatestrategy)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connEvictRate` _integer_ | Client disconnect rate (number per second).<br />Same as `conn-evict-rate` in [EMQX Node Evacuation](https://docs.emqx.com/en/emqx/v5.10/deploy/cluster/rebalancing.html#node-evacuation). | 1000 | Minimum: 1 <br /> |
| `sessEvictRate` _integer_ | Session evacuation rate (number per second).<br />Same as `sess-evict-rate` in [EMQX Node Evacuation](https://docs.emqx.com/en/emqx/v5.10/deploy/cluster/rebalancing.html#node-evacuation). | 1000 | Minimum: 1 <br /> |
| `waitTakeover` _integer_ | Amount of time (in seconds) to wait before starting session evacuation.<br />Same as `wait-takeover` in [EMQX Node Evacuation](https://docs.emqx.com/en/emqx/v5.10/deploy/cluster/rebalancing.html#node-evacuation). | 10 | Minimum: 0 <br /> |
| `waitHealthCheck` _integer_ | Duration (in seconds) during which the node waits for the Load Balancer to remove it from the active backend node list.<br />Same as `wait-health-check` in [EMQX Node Evacuation](https://docs.emqx.com/en/emqx/v5.10/deploy/cluster/rebalancing.html#node-evacuation). | 60 | Minimum: 0 <br /> |


#### KeyRef







_Appears in:_
- [SecretRef](#secretref)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretName` _string_ | Name of the Secret object. |  |  |
| `secretKey` _string_ | Entry within the Secret data. |  | Pattern: `^[a-zA-Z\d-_]+$` <br /> |


#### NodeEvacuationStatus







_Appears in:_
- [EMQXStatus](#emqxstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `nodeName` _string_ | Evacuated node name |  |  |
| `state` _string_ | Evacuation state |  |  |
| `sessionRecipients` _string array_ | Session recipients |  |  |
| `sessionEvictionRate` _integer_ | Session eviction rate, in sessions per second. |  |  |
| `connectionEvictionRate` _integer_ | Connection eviction rate, in connections per second. |  |  |
| `initialSessions` _integer_ | Initial number of sessions on this node |  |  |
| `initialConnections` _integer_ | Initial number of connections to this node |  |  |


#### SecretRef







_Appears in:_
- [BootstrapAPIKey](#bootstrapapikey)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `key` _[KeyRef](#keyref)_ | Reference to a Secret entry containing the EMQX API Key. |  |  |
| `secret` _[KeyRef](#keyref)_ | Reference to a Secret entry containing the EMQX API Key's secret. |  |  |


#### ServiceTemplate







_Appears in:_
- [EMQXSpec](#emqxspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Specifies whether the Service should be created. | true |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ServiceSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#servicespec-v1-core)_ | Specification of the desired state of a Service.<br />https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status |  |  |


#### UpdateStrategy







_Appears in:_
- [EMQXSpec](#emqxspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Determines how cluster upgrade is performed.<br />* `Recreate`: Perform blue-green upgrade. | Recreate | Enum: [Recreate] <br /> |
| `initialDelaySeconds` _integer_ | Number of seconds before connection evacuation starts. | 10 | Minimum: 0 <br /> |
| `evacuationStrategy` _[EvacuationStrategy](#evacuationstrategy)_ | Evacuation strategy settings. |  |  |


