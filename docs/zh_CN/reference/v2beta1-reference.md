# API Reference

## Packages
- [apps.emqx.io/v2beta1](#appsemqxiov2beta1)


## apps.emqx.io/v2beta1

Package v2beta1 contains API Schema definitions for the apps v2beta1 API group

### Resource Types
- [EMQX](#emqx)
- [EMQXList](#emqxlist)
- [Rebalance](#rebalance)
- [RebalanceList](#rebalancelist)



#### BootstrapAPIKey







_Appears in:_
- [EMQXSpec](#emqxspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `key` _string_ |  |  | Pattern: `^[a-zA-Z\d-_]+$` <br /> |
| `secret` _string_ |  |  | MaxLength: 128 <br />MinLength: 3 <br /> |
| `secretRef` _[SecretRef](#secretref)_ |  |  |  |


#### Config







_Appears in:_
- [EMQXSpec](#emqxspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `mode` _string_ |  | Merge | Enum: [Merge Replace] <br /> |
| `data` _string_ | EMQX config, HOCON format, like etc/emqx.conf file |  |  |


#### EMQX



EMQX is the Schema for the emqxes API



_Appears in:_
- [EMQXList](#emqxlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `apps.emqx.io/v2beta1` | | |
| `kind` _string_ | `EMQX` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[EMQXSpec](#emqxspec)_ | Spec defines the desired identities of EMQX nodes in this set. |  |  |
| `status` _[EMQXStatus](#emqxstatus)_ | Status is the current status of EMQX nodes. This data<br />may be out of date by some window of time. |  |  |


#### EMQXCoreTemplate







_Appears in:_
- [EMQXSpec](#emqxspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[EMQXCoreTemplateSpec](#emqxcoretemplatespec)_ | Specification of the desired behavior of the EMQX core node.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status |  |  |


#### EMQXCoreTemplateSpec







_Appears in:_
- [EMQXCoreTemplate](#emqxcoretemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `nodeSelector` _object (keys:string, values:string)_ | NodeSelector is a selector which must be true for the pod to fit on a node. Selector which must match a node's labels for the pod to be scheduled on that node.<br />More info: https://kubernetes.io/docs/concepts/config/assign-pod-node/ |  |  |
| `nodeName` _string_ | NodeName is a request to schedule this pod onto a specific node. If it is non-empty, the scheduler simply schedules this pod onto that node, assuming that it fits resource requirements. |  |  |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#affinity-v1-core)_ | Affinity for pod assignment<br />ref: https://kubernetes.io/docs/concepts/config/assign-pod-node/#affinity-and-anti-affinity |  |  |
| `toleRations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#toleration-v1-core) array_ | If specified, the pod's tolerations.<br />The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator .<br />TODO: should use `tolerations` instead, this field just for compatible with old version, will delete in future. |  |  |
| `tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#toleration-v1-core) array_ | If specified, the pod's tolerations.<br />The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator . |  |  |
| `topologySpreadConstraints` _[TopologySpreadConstraint](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#topologyspreadconstraint-v1-core) array_ | // TopologySpreadConstraint specifies how to spread matching pods among the given topology. |  |  |
| `replicas` _integer_ | Replicas is the desired number of replicas of the given Template.<br />These are replicas in the sense that they are instantiations of the<br />same Template, but individual replicas also have a consistent identity.<br />Defaults to 2. | 2 |  |
| `command` _string array_ | Entrypoint array. Not executed within a shell.<br />The container image's ENTRYPOINT is used if this is not provided.<br />Variable references $(VAR_NAME) are expanded using the container's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced<br />to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will<br />produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless<br />of whether the variable exists or not. Cannot be updated.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell |  |  |
| `args` _string array_ | Arguments to the entrypoint.<br />The container image's CMD is used if this is not provided.<br />Variable references $(VAR_NAME) are expanded using the container's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced<br />to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will<br />produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless<br />of whether the variable exists or not. Cannot be updated.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell |  |  |
| `ports` _[ContainerPort](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#containerport-v1-core) array_ | List of ports to expose from the container. Exposing a port here gives<br />the system additional information about the network connections a<br />container uses, but is primarily informational. Not specifying a port here<br />DOES NOT prevent that port from being exposed. Any port which is<br />listening on the default "0.0.0.0" address inside a container will be<br />accessible from the network.<br />Cannot be updated. |  |  |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#envvar-v1-core) array_ | List of environment variables to set in the container.<br />Cannot be updated. |  |  |
| `envFrom` _[EnvFromSource](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#envfromsource-v1-core) array_ | List of sources to populate environment variables in the container.<br />The keys defined within a source must be a C_IDENTIFIER. All invalid keys<br />will be reported as an event when the container is starting. When a key exists in multiple<br />sources, the value associated with the last source will take precedence.<br />Values defined by an Env with a duplicate key will take precedence.<br />Cannot be updated. |  |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#resourcerequirements-v1-core)_ | Compute Resources required by this container.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/config/manage-resources-containers/ |  |  |
| `podSecurityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#podsecuritycontext-v1-core)_ | SecurityContext holds pod-level security attributes and common container settings. | \{ fsGroup:1000 fsGroupChangePolicy:Always runAsGroup:1000 runAsUser:1000 supplementalGroups:[1000] \} |  |
| `containerSecurityContext` _[SecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#securitycontext-v1-core)_ | SecurityContext defines the security options the container should be run with.<br />If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.<br />More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/ | \{ runAsGroup:1000 runAsNonRoot:true runAsUser:1000 \} |  |
| `initContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#container-v1-core) array_ | List of initialization containers belonging to the pod.<br />Init containers are executed in order prior to containers being started. If any<br />init container fails, the pod is considered to have failed and is handled according<br />to its restartPolicy. The name for an init container or normal container must be<br />unique among all containers.<br />Init containers may not have Lifecycle actions, Readiness probes, Liveness probes, or Startup probes.<br />The resourceRequirements of an init container are taken into account during scheduling<br />by finding the highest request/limit for each resource type, and then using the max of<br />of that value or the sum of the normal containers. Limits are applied to init containers<br />in a similar fashion.<br />Init containers cannot currently be added or removed.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/init-containers/ |  |  |
| `extraContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#container-v1-core) array_ | ExtraContainers represents extra containers to be added to the pod.<br />See https://github.com/emqx/emqx-operator/issues/252 |  |  |
| `extraVolumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#volume-v1-core) array_ | See https://github.com/emqx/emqx-operator/pull/72 |  |  |
| `extraVolumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#volumemount-v1-core) array_ | See https://github.com/emqx/emqx-operator/pull/72 |  |  |
| `livenessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#probe-v1-core)_ | Periodic probe of container liveness.<br />Container will be restarted if the probe fails.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes | \{ failureThreshold:3 httpGet:map[path:/status port:dashboard] initialDelaySeconds:60 periodSeconds:30 \} |  |
| `readinessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#probe-v1-core)_ | Periodic probe of container service readiness.<br />Container will be removed from service endpoints if the probe fails.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes | \{ failureThreshold:12 httpGet:map[path:/status port:dashboard] initialDelaySeconds:10 periodSeconds:5 \} |  |
| `startupProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#probe-v1-core)_ | StartupProbe indicates that the Pod has successfully initialized.<br />If specified, no other probes are executed until this completes successfully.<br />If this probe fails, the Pod will be restarted, just as if the livenessProbe failed.<br />This can be used to provide different probe parameters at the beginning of a Pod's lifecycle,<br />when it might take a long time to load data or warm a cache, than during steady-state operation.<br />This cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes |  |  |
| `lifecycle` _[Lifecycle](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#lifecycle-v1-core)_ | Actions that the management system should take in response to container lifecycle events.<br />Cannot be updated. |  |  |
| `volumeClaimTemplates` _[PersistentVolumeClaimSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#persistentvolumeclaimspec-v1-core)_ | VolumeClaimTemplates is a list of claims that pods are allowed to reference.<br />The StatefulSet controller is responsible for mapping network identities to<br />claims in a way that maintains the identity of a pod. Every claim in<br />this list must have at least one matching (by name) volumeMount in one<br />container in the template. A claim in this list takes precedence over<br />any volumes in the template, with the same name.<br />More than EMQXReplicantTemplateSpec |  |  |


#### EMQXList



EMQXList contains a list of EMQX





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `apps.emqx.io/v2beta1` | | |
| `kind` _string_ | `EMQXList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[EMQX](#emqx) array_ |  |  |  |


#### EMQXNode







_Appears in:_
- [EMQXStatus](#emqxstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `controllerUID` _[UID](#uid)_ |  |  |  |
| `podUID` _[UID](#uid)_ |  |  |  |
| `node` _string_ | EMQX node name, example: emqx@127.0.0.1 |  |  |
| `node_status` _string_ | EMQX node status, example: Running |  |  |
| `otp_release` _string_ | Erlang/OTP version used by EMQX, example: 24.2/12.2 |  |  |
| `version` _string_ | EMQX version |  |  |
| `role` _string_ | EMQX cluster node role, enum: "core" "replicant" |  |  |
| `edition` _string_ | EMQX cluster node edition, enum: "Opensource" "Enterprise" |  |  |
| `uptime` _integer_ | EMQX node uptime, milliseconds |  |  |
| `connections` _integer_ | In EMQX's API of `/api/v5/nodes`, the `connections` field means the number of MQTT session count, |  |  |
| `live_connections` _integer_ | In EMQX's API of `/api/v5/nodes`, the `live_connections` field means the number of connected MQTT clients.<br />THe `live_connections` just work in EMQX 5.1 or later. |  |  |


#### EMQXNodesStatus







_Appears in:_
- [EMQXStatus](#emqxstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `replicas` _integer_ |  |  |  |
| `readyReplicas` _integer_ |  |  |  |
| `currentRevision` _string_ |  |  |  |
| `currentReplicas` _integer_ |  |  |  |
| `updateRevision` _string_ |  |  |  |
| `updateReplicas` _integer_ |  |  |  |
| `collisionCount` _integer_ |  |  |  |


#### EMQXReplicantTemplate







_Appears in:_
- [EMQXSpec](#emqxspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[EMQXReplicantTemplateSpec](#emqxreplicanttemplatespec)_ | Specification of the desired behavior of the EMQX replicant node.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status |  |  |


#### EMQXReplicantTemplateSpec







_Appears in:_
- [EMQXCoreTemplateSpec](#emqxcoretemplatespec)
- [EMQXReplicantTemplate](#emqxreplicanttemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `nodeSelector` _object (keys:string, values:string)_ | NodeSelector is a selector which must be true for the pod to fit on a node. Selector which must match a node's labels for the pod to be scheduled on that node.<br />More info: https://kubernetes.io/docs/concepts/config/assign-pod-node/ |  |  |
| `nodeName` _string_ | NodeName is a request to schedule this pod onto a specific node. If it is non-empty, the scheduler simply schedules this pod onto that node, assuming that it fits resource requirements. |  |  |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#affinity-v1-core)_ | Affinity for pod assignment<br />ref: https://kubernetes.io/docs/concepts/config/assign-pod-node/#affinity-and-anti-affinity |  |  |
| `toleRations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#toleration-v1-core) array_ | If specified, the pod's tolerations.<br />The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator .<br />TODO: should use `tolerations` instead, this field just for compatible with old version, will delete in future. |  |  |
| `tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#toleration-v1-core) array_ | If specified, the pod's tolerations.<br />The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator . |  |  |
| `topologySpreadConstraints` _[TopologySpreadConstraint](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#topologyspreadconstraint-v1-core) array_ | // TopologySpreadConstraint specifies how to spread matching pods among the given topology. |  |  |
| `replicas` _integer_ | Replicas is the desired number of replicas of the given Template.<br />These are replicas in the sense that they are instantiations of the<br />same Template, but individual replicas also have a consistent identity.<br />Defaults to 2. | 2 |  |
| `command` _string array_ | Entrypoint array. Not executed within a shell.<br />The container image's ENTRYPOINT is used if this is not provided.<br />Variable references $(VAR_NAME) are expanded using the container's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced<br />to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will<br />produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless<br />of whether the variable exists or not. Cannot be updated.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell |  |  |
| `args` _string array_ | Arguments to the entrypoint.<br />The container image's CMD is used if this is not provided.<br />Variable references $(VAR_NAME) are expanded using the container's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced<br />to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will<br />produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless<br />of whether the variable exists or not. Cannot be updated.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell |  |  |
| `ports` _[ContainerPort](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#containerport-v1-core) array_ | List of ports to expose from the container. Exposing a port here gives<br />the system additional information about the network connections a<br />container uses, but is primarily informational. Not specifying a port here<br />DOES NOT prevent that port from being exposed. Any port which is<br />listening on the default "0.0.0.0" address inside a container will be<br />accessible from the network.<br />Cannot be updated. |  |  |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#envvar-v1-core) array_ | List of environment variables to set in the container.<br />Cannot be updated. |  |  |
| `envFrom` _[EnvFromSource](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#envfromsource-v1-core) array_ | List of sources to populate environment variables in the container.<br />The keys defined within a source must be a C_IDENTIFIER. All invalid keys<br />will be reported as an event when the container is starting. When a key exists in multiple<br />sources, the value associated with the last source will take precedence.<br />Values defined by an Env with a duplicate key will take precedence.<br />Cannot be updated. |  |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#resourcerequirements-v1-core)_ | Compute Resources required by this container.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/config/manage-resources-containers/ |  |  |
| `podSecurityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#podsecuritycontext-v1-core)_ | SecurityContext holds pod-level security attributes and common container settings. | \{ fsGroup:1000 fsGroupChangePolicy:Always runAsGroup:1000 runAsUser:1000 supplementalGroups:[1000] \} |  |
| `containerSecurityContext` _[SecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#securitycontext-v1-core)_ | SecurityContext defines the security options the container should be run with.<br />If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.<br />More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/ | \{ runAsGroup:1000 runAsNonRoot:true runAsUser:1000 \} |  |
| `initContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#container-v1-core) array_ | List of initialization containers belonging to the pod.<br />Init containers are executed in order prior to containers being started. If any<br />init container fails, the pod is considered to have failed and is handled according<br />to its restartPolicy. The name for an init container or normal container must be<br />unique among all containers.<br />Init containers may not have Lifecycle actions, Readiness probes, Liveness probes, or Startup probes.<br />The resourceRequirements of an init container are taken into account during scheduling<br />by finding the highest request/limit for each resource type, and then using the max of<br />of that value or the sum of the normal containers. Limits are applied to init containers<br />in a similar fashion.<br />Init containers cannot currently be added or removed.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/init-containers/ |  |  |
| `extraContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#container-v1-core) array_ | ExtraContainers represents extra containers to be added to the pod.<br />See https://github.com/emqx/emqx-operator/issues/252 |  |  |
| `extraVolumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#volume-v1-core) array_ | See https://github.com/emqx/emqx-operator/pull/72 |  |  |
| `extraVolumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#volumemount-v1-core) array_ | See https://github.com/emqx/emqx-operator/pull/72 |  |  |
| `livenessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#probe-v1-core)_ | Periodic probe of container liveness.<br />Container will be restarted if the probe fails.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes | \{ failureThreshold:3 httpGet:map[path:/status port:dashboard] initialDelaySeconds:60 periodSeconds:30 \} |  |
| `readinessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#probe-v1-core)_ | Periodic probe of container service readiness.<br />Container will be removed from service endpoints if the probe fails.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes | \{ failureThreshold:12 httpGet:map[path:/status port:dashboard] initialDelaySeconds:10 periodSeconds:5 \} |  |
| `startupProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#probe-v1-core)_ | StartupProbe indicates that the Pod has successfully initialized.<br />If specified, no other probes are executed until this completes successfully.<br />If this probe fails, the Pod will be restarted, just as if the livenessProbe failed.<br />This can be used to provide different probe parameters at the beginning of a Pod's lifecycle,<br />when it might take a long time to load data or warm a cache, than during steady-state operation.<br />This cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes |  |  |
| `lifecycle` _[Lifecycle](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#lifecycle-v1-core)_ | Actions that the management system should take in response to container lifecycle events.<br />Cannot be updated. |  |  |


#### EMQXSpec



EMQXSpec defines the desired state of EMQX



_Appears in:_
- [EMQX](#emqx)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `image` _string_ | EMQX image name.<br />More info: https://kubernetes.io/docs/concepts/containers/images |  |  |
| `imagePullPolicy` _[PullPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#pullpolicy-v1-core)_ | Image pull policy.<br />One of Always, Never, IfNotPresent.<br />Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/containers/images#updating-images |  |  |
| `imagePullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#localobjectreference-v1-core) array_ | ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.<br />If specified, these secrets will be passed to individual puller implementations for them to use.<br />More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod |  |  |
| `serviceAccountName` _string_ | Service Account Name<br />This associates the ReplicaSet or StatefulSet with the specified Service Account for authentication purposes.<br />More info: https://kubernetes.io/docs/concepts/security/service-accounts |  |  |
| `bootstrapAPIKeys` _[BootstrapAPIKey](#bootstrapapikey) array_ | EMQX bootstrap user<br />Cannot be updated. |  |  |
| `config` _[Config](#config)_ | EMQX config |  |  |
| `clusterDomain` _string_ |  | cluster.local |  |
| `revisionHistoryLimit` _integer_ | The number of old ReplicaSets, old StatefulSet and old PersistentVolumeClaim to retain to allow rollback.<br />This is a pointer to distinguish between explicit zero and not specified.<br />Defaults to 3. | 3 |  |
| `updateStrategy` _[UpdateStrategy](#updatestrategy)_ | UpdateStrategy is the object that describes the EMQX blue-green update strategy | \{ evacuationStrategy:map[connEvictRate:1000 sessEvictRate:1000 waitTakeover:10] initialDelaySeconds:10 type:Recreate \} |  |
| `coreTemplate` _[EMQXCoreTemplate](#emqxcoretemplate)_ | CoreTemplate is the object that describes the EMQX core node that will be created | \{ spec:map[replicas:2] \} |  |
| `replicantTemplate` _[EMQXReplicantTemplate](#emqxreplicanttemplate)_ | ReplicantTemplate is the object that describes the EMQX replicant node that will be created |  |  |
| `dashboardServiceTemplate` _[ServiceTemplate](#servicetemplate)_ | DashboardServiceTemplate is the object that describes the EMQX dashboard service that will be created<br />This service always selector the EMQX core node |  |  |
| `listenersServiceTemplate` _[ServiceTemplate](#servicetemplate)_ | ListenersServiceTemplate is the object that describes the EMQX listener service that will be created<br />If the EMQX replicant node exist, this service will selector the EMQX replicant node<br />Else this service will selector EMQX core node |  |  |


#### EMQXStatus



EMQXStatus defines the observed state of EMQX



_Appears in:_
- [EMQX](#emqx)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#condition-v1-meta) array_ | Represents the latest available observations of a EMQX Custom Resource current state. |  |  |
| `coreNodes` _[EMQXNode](#emqxnode) array_ |  |  |  |
| `coreNodesStatus` _[EMQXNodesStatus](#emqxnodesstatus)_ |  |  |  |
| `replicantNodes` _[EMQXNode](#emqxnode) array_ |  |  |  |
| `replicantNodesStatus` _[EMQXNodesStatus](#emqxnodesstatus)_ |  |  |  |
| `nodEvacuationsStatus` _[NodeEvacuationStatus](#nodeevacuationstatus) array_ |  |  |  |


#### EvacuationStrategy







_Appears in:_
- [UpdateStrategy](#updatestrategy)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `waitTakeover` _integer_ |  |  | Minimum: 0 <br /> |
| `connEvictRate` _integer_ | Just work in EMQX Enterprise. | 1000 | Minimum: 1 <br /> |
| `sessEvictRate` _integer_ | Just work in EMQX Enterprise. | 1000 | Minimum: 1 <br /> |


#### KeyRef







_Appears in:_
- [SecretRef](#secretref)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretName` _string_ |  |  |  |
| `secretKey` _string_ |  |  | Pattern: `^[a-zA-Z\d-_]+$` <br /> |


#### NodeEvacuationStats







_Appears in:_
- [NodeEvacuationStatus](#nodeevacuationstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `initial_sessions` _integer_ |  |  |  |
| `initial_connected` _integer_ |  |  |  |
| `current_sessions` _integer_ |  |  |  |
| `current_connected` _integer_ |  |  |  |


#### NodeEvacuationStatus







_Appears in:_
- [EMQXStatus](#emqxstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `node` _string_ |  |  |  |
| `stats` _[NodeEvacuationStats](#nodeevacuationstats)_ |  |  |  |
| `state` _string_ |  |  |  |
| `session_recipients` _string array_ |  |  |  |
| `session_goal` _integer_ |  |  |  |
| `session_eviction_rate` _integer_ |  |  |  |
| `connection_goal` _integer_ |  |  |  |
| `connection_eviction_rate` _integer_ |  |  |  |


#### Rebalance



Rebalance is the Schema for the rebalances API



_Appears in:_
- [RebalanceList](#rebalancelist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `apps.emqx.io/v2beta1` | | |
| `kind` _string_ | `Rebalance` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[RebalanceSpec](#rebalancespec)_ |  |  |  |
| `status` _[RebalanceStatus](#rebalancestatus)_ |  |  |  |


#### RebalanceCondition



RebalanceCondition describes current state of a EMQX rebalancing job.



_Appears in:_
- [RebalanceStatus](#rebalancestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[RebalanceConditionType](#rebalanceconditiontype)_ | Status of rebalance condition type. one of Processing, Complete, Failed. |  |  |
| `status` _[ConditionStatus](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#conditionstatus-v1-core)_ | Status of the condition, one of True, False, Unknown. |  |  |
| `lastUpdateTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | The last time this condition was updated. |  |  |
| `lastTransitionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | Last time the condition transitioned from one status to another. |  |  |
| `reason` _string_ | The reason for the condition's last transition. |  |  |
| `message` _string_ | A human readable message indicating details about the transition. |  |  |


#### RebalanceConditionType

_Underlying type:_ _string_





_Appears in:_
- [RebalanceCondition](#rebalancecondition)



#### RebalanceList



RebalanceList contains a list of Rebalance





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `apps.emqx.io/v2beta1` | | |
| `kind` _string_ | `RebalanceList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Rebalance](#rebalance) array_ |  |  |  |


#### RebalancePhase

_Underlying type:_ _string_





_Appears in:_
- [RebalanceStatus](#rebalancestatus)



#### RebalanceSpec



RebalanceSpec defines the desired state of Rebalance



_Appears in:_
- [Rebalance](#rebalance)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `instanceKind` _string_ | InstanceKind is used to distinguish between EMQX and EMQXEnterprise.<br />When it is set to "EMQX", it means that the EMQX CR is v2beta1,<br />and when it is set to "EmqxEnterprise", it means that the EmqxEnterprise CR is v1beta4. | EMQX |  |
| `instanceName` _string_ | InstanceName represents the name of EMQX CR, just work for EMQX Enterprise |  | Required: {} <br /> |
| `rebalanceStrategy` _[RebalanceStrategy](#rebalancestrategy)_ | RebalanceStrategy represents the strategy of EMQX rebalancing<br />More info: https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing |  | Required: {} <br /> |


#### RebalanceState



Rebalance defines the observed Rebalancing state of EMQX



_Appears in:_
- [RebalanceStatus](#rebalancestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `state` _string_ | State represents the state of emqx cluster rebalancing. |  |  |
| `session_eviction_rate` _integer_ | SessionEvictionRate represents the node session evacuation rate per second. |  |  |
| `recipients` _string array_ | Recipients represent the target node for rebalancing. |  |  |
| `node` _string_ | Node represents the rebalancing scheduling node. |  |  |
| `donors` _string array_ | Donors represent the source nodes for rebalancing. |  |  |
| `coordinator_node` _string_ | CoordinatorNode represents the node currently undergoing rebalancing. |  |  |
| `connection_eviction_rate` _integer_ | ConnectionEvictionRate represents the node session evacuation rate per second. |  |  |


#### RebalanceStatus



RebalanceStatus represents the current state of Rebalance



_Appears in:_
- [Rebalance](#rebalance)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[RebalanceCondition](#rebalancecondition) array_ | The latest available observations of an object's current state.<br />When Rebalance fails, the condition will have type "Failed" and status false.<br />When Rebalance is in processing, the condition will have a type "Processing" and status true.<br />When Rebalance is completed, the condition will have a type "Complete" and status true. |  |  |
| `phase` _[RebalancePhase](#rebalancephase)_ | Phase represents the phase of Rebalance. |  |  |
| `rebalanceStates` _[RebalanceState](#rebalancestate) array_ |  |  |  |
| `startedTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | StartedTime Represents the time when rebalance job start. |  |  |
| `completedTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | CompletedTime Represents the time when the rebalance job was completed. |  |  |


#### RebalanceStrategy



RebalanceStrategy represents the strategy of EMQX rebalancing



_Appears in:_
- [RebalanceSpec](#rebalancespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connEvictRate` _integer_ | ConnEvictRate represents the source node client disconnect rate per second.<br />same to conn-evict-rate in [EMQX Rebalancing](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing)<br />The value must be greater than 0 |  | Minimum: 1 <br />Required: {} <br /> |
| `sessEvictRate` _integer_ | SessEvictRate represents the source node session evacuation rate per second.<br />same to sess-evict-rate in [EMQX Rebalancing](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing)<br />The value must be greater than 0<br />Defaults to 500. | 500 |  |
| `waitTakeover` _integer_ | WaitTakeover represents the time in seconds to wait for a client to<br />reconnect to take over the session after all connections are disconnected.<br />same to wait-takeover in [EMQX Rebalancing](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing)<br />The value must be greater than 0<br />Defaults to 60 seconds. | 60 |  |
| `waitHealthCheck` _integer_ | WaitHealthCheck represents the time (in seconds) to wait for the LB to<br />remove the source node from the list of active backend nodes. After the<br />specified waiting time is exceeded,the rebalancing task will start.<br />same to wait-health-check in [EMQX Rebalancing](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing)<br />The value must be greater than 0<br />Defaults to 60 seconds. | 60 |  |
| `absConnThreshold` _integer_ | AbsConnThreshold represents the absolute threshold for checking connection balance.<br />same to abs-conn-threshold in [EMQX Rebalancing](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing)<br />The value must be greater than 0<br />Defaults to 1000. | 1000 |  |
| `relConnThreshold` _string_ | RelConnThreshold represents the relative threshold for checkin connection balance.<br />same to rel-conn-threshold in [EMQX Rebalancing](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing)<br />the usage of float highly discouraged, as support for them varies across languages.<br />So we define the RelConnThreshold field as string type and you not float type<br />The value must be greater than "1.0"<br />Defaults to "1.1". | 1.1 |  |
| `absSessThreshold` _integer_ | AbsSessThreshold represents the absolute threshold for checking session connection balance.<br />same to abs-sess-threshold in [EMQX Rebalancing](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing)<br />The value must be greater than 0<br />Default to 1000. | 1000 |  |
| `relSessThreshold` _string_ | RelSessThreshold represents the relative threshold for checking session connection balance.<br />same to rel-sess-threshold in [EMQX Rebalancing](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing)<br />the usage of float highly discouraged, as support for them varies across languages.<br />So we define the RelSessThreshold field as string type and you not float type<br />The value must be greater than "1.0"<br />Defaults to "1.1". | 1.1 |  |


#### SecretRef







_Appears in:_
- [BootstrapAPIKey](#bootstrapapikey)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `key` _[KeyRef](#keyref)_ |  |  |  |
| `secret` _[KeyRef](#keyref)_ |  |  |  |


#### ServiceTemplate







_Appears in:_
- [EMQXSpec](#emqxspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | EMQX Operator will create a service for EMQX nodes.<br />This is a pointer to distinguish between `false` and not specified. | true |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ServiceSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#servicespec-v1-core)_ | Spec defines the behavior of a service.<br />https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status |  |  |


#### UpdateStrategy







_Appears in:_
- [EMQXSpec](#emqxspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ |  | Recreate | Enum: [Recreate] <br /> |
| `initialDelaySeconds` _integer_ | Number of seconds before evacuation connection start. |  |  |
| `evacuationStrategy` _[EvacuationStrategy](#evacuationstrategy)_ | Number of seconds before evacuation connection timeout. |  |  |


