# API Reference

## Packages
- [apps.emqx.io/v2alpha1](#appsemqxiov2alpha1)


## apps.emqx.io/v2alpha1

Package v2alpha1 contains API Schema definitions for the apps v2alpha1 API group

### Resource Types
- [EMQX](#emqx)
- [EMQXList](#emqxlist)



#### BootstrapAPIKey







_Appears in:_
- [EMQXSpec](#emqxspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `key` _string_ |  |  | Pattern: `^[a-zA-Z\d_]+$` <br /> |
| `secret` _string_ |  |  | MaxLength: 32 <br />MinLength: 3 <br /> |


#### Condition







_Appears in:_
- [EMQXStatus](#emqxstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[ConditionType](#conditiontype)_ | Status of cluster condition. |  |  |
| `status` _[ConditionStatus](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#conditionstatus-v1-core)_ | Status of the condition, one of True, False, Unknown. |  |  |
| `reason` _string_ | The reason for the condition's last transition. |  |  |
| `message` _string_ | A human readable message indicating details about the transition. |  |  |
| `lastTransitionTime` _string_ | Last time the condition transitioned from one status to another. |  |  |
| `lastUpdateTime` _string_ | The last time this condition was updated. |  |  |


#### ConditionType

_Underlying type:_ _string_





_Appears in:_
- [Condition](#condition)



#### EMQX



EMQX is the Schema for the emqxes API



_Appears in:_
- [EMQXList](#emqxlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `apps.emqx.io/v2alpha1` | | |
| `kind` _string_ | `EMQX` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[EMQXSpec](#emqxspec)_ | Spec defines the desired identities of EMQX nodes in this set. |  |  |
| `status` _[EMQXStatus](#emqxstatus)_ | Status is the current status of EMQX nodes. This data<br />may be out of date by some window of time. |  |  |


#### EMQXCoreTemplate







_Appears in:_
- [EMQXSpec](#emqxspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[EMQXCoreTemplateSpec](#emqxcoretemplatespec)_ | Specification of the desired behavior of the EMQX core node.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status |  |  |


#### EMQXCoreTemplateSpec







_Appears in:_
- [EMQXCoreTemplate](#emqxcoretemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `nodeSelector` _object (keys:string, values:string)_ | NodeSelector is a selector which must be true for the pod to fit on a node. Selector which must match a node's labels for the pod to be scheduled on that node.<br />More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/ |  |  |
| `nodeName` _string_ | NodeName is a request to schedule this pod onto a specific node. If it is non-empty, the scheduler simply schedules this pod onto that node, assuming that it fits resource requirements. |  |  |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#affinity-v1-core)_ | Affinity for pod assignment<br />ref: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity |  |  |
| `toleRations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#toleration-v1-core) array_ | If specified, the pod's tolerations.<br />The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator . |  |  |
| `replicas` _integer_ | Replicas is the desired number of replicas of the given Template.<br />These are replicas in the sense that they are instantiations of the<br />same Template, but individual replicas also have a consistent identity.<br />If unspecified, defaults to 3. | 3 |  |
| `command` _string array_ | Entrypoint array. Not executed within a shell.<br />The container image's ENTRYPOINT is used if this is not provided.<br />Variable references $(VAR_NAME) are expanded using the container's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced<br />to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will<br />produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless<br />of whether the variable exists or not. Cannot be updated.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell |  |  |
| `args` _string array_ | Arguments to the entrypoint.<br />The container image's CMD is used if this is not provided.<br />Variable references $(VAR_NAME) are expanded using the container's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced<br />to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will<br />produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless<br />of whether the variable exists or not. Cannot be updated.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell |  |  |
| `ports` _[ContainerPort](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#containerport-v1-core) array_ | List of ports to expose from the container. Exposing a port here gives<br />the system additional information about the network connections a<br />container uses, but is primarily informational. Not specifying a port here<br />DOES NOT prevent that port from being exposed. Any port which is<br />listening on the default "0.0.0.0" address inside a container will be<br />accessible from the network.<br />Cannot be updated. |  |  |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#envvar-v1-core) array_ | List of environment variables to set in the container.<br />Cannot be updated. |  |  |
| `envFrom` _[EnvFromSource](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#envfromsource-v1-core) array_ | List of sources to populate environment variables in the container.<br />The keys defined within a source must be a C_IDENTIFIER. All invalid keys<br />will be reported as an event when the container is starting. When a key exists in multiple<br />sources, the value associated with the last source will take precedence.<br />Values defined by an Env with a duplicate key will take precedence.<br />Cannot be updated. |  |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#resourcerequirements-v1-core)_ | Compute Resources required by this container.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ |  |  |
| `podSecurityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#podsecuritycontext-v1-core)_ | SecurityContext holds pod-level security attributes and common container settings.<br />Optional: Defaults to empty.  See type description for default values of each field. |  |  |
| `containerSecurityContext` _[SecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#securitycontext-v1-core)_ | SecurityContext defines the security options the container should be run with.<br />If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.<br />More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/ |  |  |
| `initContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#container-v1-core) array_ | List of initialization containers belonging to the pod.<br />Init containers are executed in order prior to containers being started. If any<br />init container fails, the pod is considered to have failed and is handled according<br />to its restartPolicy. The name for an init container or normal container must be<br />unique among all containers.<br />Init containers may not have Lifecycle actions, Readiness probes, Liveness probes, or Startup probes.<br />The resourceRequirements of an init container are taken into account during scheduling<br />by finding the highest request/limit for each resource type, and then using the max of<br />of that value or the sum of the normal containers. Limits are applied to init containers<br />in a similar fashion.<br />Init containers cannot currently be added or removed.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/init-containers/ |  |  |
| `extraContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#container-v1-core) array_ | ExtraContainers represents extra containers to be added to the pod.<br />See https://github.com/emqx/emqx-operator/issues/252 |  |  |
| `extraVolumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#volume-v1-core) array_ | See https://github.com/emqx/emqx-operator/pull/72 |  |  |
| `extraVolumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#volumemount-v1-core) array_ | See https://github.com/emqx/emqx-operator/pull/72 |  |  |
| `livenessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#probe-v1-core)_ | Periodic probe of container liveness.<br />Container will be restarted if the probe fails.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes |  |  |
| `readinessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#probe-v1-core)_ | Periodic probe of container service readiness.<br />Container will be removed from service endpoints if the probe fails.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes |  |  |
| `startupProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#probe-v1-core)_ | StartupProbe indicates that the Pod has successfully initialized.<br />If specified, no other probes are executed until this completes successfully.<br />If this probe fails, the Pod will be restarted, just as if the livenessProbe failed.<br />This can be used to provide different probe parameters at the beginning of a Pod's lifecycle,<br />when it might take a long time to load data or warm a cache, than during steady-state operation.<br />This cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes |  |  |
| `lifecycle` _[Lifecycle](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#lifecycle-v1-core)_ | Actions that the management system should take in response to container lifecycle events.<br />Cannot be updated. |  |  |
| `volumeClaimTemplates` _[PersistentVolumeClaimSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#persistentvolumeclaimspec-v1-core)_ | VolumeClaimTemplates is a list of claims that pods are allowed to reference.<br />The StatefulSet controller is responsible for mapping network identities to<br />claims in a way that maintains the identity of a pod. Every claim in<br />this list must have at least one matching (by name) volumeMount in one<br />container in the template. A claim in this list takes precedence over<br />any volumes in the template, with the same name.<br />More than EMQXReplicantTemplateSpec |  |  |


#### EMQXList



EMQXList contains a list of EMQX





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `apps.emqx.io/v2alpha1` | | |
| `kind` _string_ | `EMQXList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[EMQX](#emqx) array_ |  |  |  |


#### EMQXNode







_Appears in:_
- [EMQXStatus](#emqxstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `node` _string_ | EMQX node name, example: emqx@127.0.0.1 |  |  |
| `node_status` _string_ | EMQX node status, example: Running |  |  |
| `otp_release` _string_ | Erlang/OTP version used by EMQX, example: 24.2/12.2 |  |  |
| `version` _string_ | EMQX version |  |  |
| `role` _string_ | EMQX cluster node role, enum: "core" "replicant" |  |  |
| `edition` _string_ | EMQX cluster node edition, enum: "Opensource" "Enterprise" |  |  |
| `uptime` _integer_ | EMQX node uptime, milliseconds |  |  |
| `connections` _integer_ | MQTT connection count |  |  |


#### EMQXReplicantTemplate







_Appears in:_
- [EMQXSpec](#emqxspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[EMQXReplicantTemplateSpec](#emqxreplicanttemplatespec)_ | Specification of the desired behavior of the EMQX replicant node.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status |  |  |


#### EMQXReplicantTemplateSpec







_Appears in:_
- [EMQXCoreTemplateSpec](#emqxcoretemplatespec)
- [EMQXReplicantTemplate](#emqxreplicanttemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `nodeSelector` _object (keys:string, values:string)_ | NodeSelector is a selector which must be true for the pod to fit on a node. Selector which must match a node's labels for the pod to be scheduled on that node.<br />More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/ |  |  |
| `nodeName` _string_ | NodeName is a request to schedule this pod onto a specific node. If it is non-empty, the scheduler simply schedules this pod onto that node, assuming that it fits resource requirements. |  |  |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#affinity-v1-core)_ | Affinity for pod assignment<br />ref: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity |  |  |
| `toleRations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#toleration-v1-core) array_ | If specified, the pod's tolerations.<br />The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator . |  |  |
| `replicas` _integer_ | Replicas is the desired number of replicas of the given Template.<br />These are replicas in the sense that they are instantiations of the<br />same Template, but individual replicas also have a consistent identity.<br />If unspecified, defaults to 3. | 3 |  |
| `command` _string array_ | Entrypoint array. Not executed within a shell.<br />The container image's ENTRYPOINT is used if this is not provided.<br />Variable references $(VAR_NAME) are expanded using the container's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced<br />to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will<br />produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless<br />of whether the variable exists or not. Cannot be updated.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell |  |  |
| `args` _string array_ | Arguments to the entrypoint.<br />The container image's CMD is used if this is not provided.<br />Variable references $(VAR_NAME) are expanded using the container's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced<br />to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will<br />produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless<br />of whether the variable exists or not. Cannot be updated.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell |  |  |
| `ports` _[ContainerPort](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#containerport-v1-core) array_ | List of ports to expose from the container. Exposing a port here gives<br />the system additional information about the network connections a<br />container uses, but is primarily informational. Not specifying a port here<br />DOES NOT prevent that port from being exposed. Any port which is<br />listening on the default "0.0.0.0" address inside a container will be<br />accessible from the network.<br />Cannot be updated. |  |  |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#envvar-v1-core) array_ | List of environment variables to set in the container.<br />Cannot be updated. |  |  |
| `envFrom` _[EnvFromSource](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#envfromsource-v1-core) array_ | List of sources to populate environment variables in the container.<br />The keys defined within a source must be a C_IDENTIFIER. All invalid keys<br />will be reported as an event when the container is starting. When a key exists in multiple<br />sources, the value associated with the last source will take precedence.<br />Values defined by an Env with a duplicate key will take precedence.<br />Cannot be updated. |  |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#resourcerequirements-v1-core)_ | Compute Resources required by this container.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ |  |  |
| `podSecurityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#podsecuritycontext-v1-core)_ | SecurityContext holds pod-level security attributes and common container settings.<br />Optional: Defaults to empty.  See type description for default values of each field. |  |  |
| `containerSecurityContext` _[SecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#securitycontext-v1-core)_ | SecurityContext defines the security options the container should be run with.<br />If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.<br />More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/ |  |  |
| `initContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#container-v1-core) array_ | List of initialization containers belonging to the pod.<br />Init containers are executed in order prior to containers being started. If any<br />init container fails, the pod is considered to have failed and is handled according<br />to its restartPolicy. The name for an init container or normal container must be<br />unique among all containers.<br />Init containers may not have Lifecycle actions, Readiness probes, Liveness probes, or Startup probes.<br />The resourceRequirements of an init container are taken into account during scheduling<br />by finding the highest request/limit for each resource type, and then using the max of<br />of that value or the sum of the normal containers. Limits are applied to init containers<br />in a similar fashion.<br />Init containers cannot currently be added or removed.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/init-containers/ |  |  |
| `extraContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#container-v1-core) array_ | ExtraContainers represents extra containers to be added to the pod.<br />See https://github.com/emqx/emqx-operator/issues/252 |  |  |
| `extraVolumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#volume-v1-core) array_ | See https://github.com/emqx/emqx-operator/pull/72 |  |  |
| `extraVolumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#volumemount-v1-core) array_ | See https://github.com/emqx/emqx-operator/pull/72 |  |  |
| `livenessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#probe-v1-core)_ | Periodic probe of container liveness.<br />Container will be restarted if the probe fails.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes |  |  |
| `readinessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#probe-v1-core)_ | Periodic probe of container service readiness.<br />Container will be removed from service endpoints if the probe fails.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes |  |  |
| `startupProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#probe-v1-core)_ | StartupProbe indicates that the Pod has successfully initialized.<br />If specified, no other probes are executed until this completes successfully.<br />If this probe fails, the Pod will be restarted, just as if the livenessProbe failed.<br />This can be used to provide different probe parameters at the beginning of a Pod's lifecycle,<br />when it might take a long time to load data or warm a cache, than during steady-state operation.<br />This cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes |  |  |
| `lifecycle` _[Lifecycle](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#lifecycle-v1-core)_ | Actions that the management system should take in response to container lifecycle events.<br />Cannot be updated. |  |  |


#### EMQXSpec



EMQXSpec defines the desired state of EMQX



_Appears in:_
- [EMQX](#emqx)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `image` _string_ | EMQX image name.<br />More info: https://kubernetes.io/docs/concepts/containers/images |  |  |
| `imagePullPolicy` _[PullPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#pullpolicy-v1-core)_ | Image pull policy.<br />More info: https://kubernetes.io/docs/concepts/containers/images#updating-images | IfNotPresent |  |
| `imagePullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#localobjectreference-v1-core) array_ | ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.<br />If specified, these secrets will be passed to individual puller implementations for them to use.<br />More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod |  |  |
| `bootstrapAPIKeys` _[BootstrapAPIKey](#bootstrapapikey) array_ | EMQX bootstrap user<br />Cannot be updated. |  |  |
| `bootstrapConfig` _string_ | EMQX bootstrap config, hocon style, like emqx.conf<br />Cannot be updated. |  |  |
| `coreTemplate` _[EMQXCoreTemplate](#emqxcoretemplate)_ | CoreTemplate is the object that describes the EMQX core node that will be created |  |  |
| `replicantTemplate` _[EMQXReplicantTemplate](#emqxreplicanttemplate)_ | ReplicantTemplate is the object that describes the EMQX replicant node that will be created |  |  |
| `dashboardServiceTemplate` _[Service](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#service-v1-core)_ | DashboardServiceTemplate is the object that describes the EMQX dashboard service that will be created<br />This service always selector the EMQX core node |  |  |
| `listenersServiceTemplate` _[Service](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#service-v1-core)_ | ListenersServiceTemplate is the object that describes the EMQX listener service that will be created<br />If the EMQX replicant node exist, this service will selector the EMQX replicant node<br />Else this service will selector EMQX core node |  |  |


#### EMQXStatus



EMQXStatus defines the observed state of EMQX



_Appears in:_
- [EMQX](#emqx)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `currentImage` _string_ | CurrentImage, indicates the image of the EMQX used to generate Pods in the |  |  |
| `coreNodeReplicas` _integer_ | CoreNodeReplicas is the number of EMQX core node Pods created by the EMQX controller. |  |  |
| `coreNodeReadyReplicas` _integer_ | CoreNodeReadyReplicas is the number of EMQX core node Pods created for this EMQX Custom Resource with a Ready Condition. |  |  |
| `replicantNodeReplicas` _integer_ | ReplicantNodeReplicas is the number of EMQX replicant node Pods created by the EMQX controller. |  |  |
| `replicantNodeReadyReplicas` _integer_ | ReplicantNodeReadyReplicas is the number of EMQX replicant node Pods created for this EMQX Custom Resource with a Ready Condition. |  |  |
| `emqxNodes` _[EMQXNode](#emqxnode) array_ | EMQX nodes info |  |  |
| `conditions` _[Condition](#condition) array_ | Represents the latest available observations of a EMQX Custom Resource current state. |  |  |




