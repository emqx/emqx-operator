# API Reference

## Packages
- [apps.emqx.io/v1beta3](#appsemqxiov1beta3)


## apps.emqx.io/v1beta3

Package v1beta3 contains API Schema definitions for the apps v1beta3 API group

### Resource Types
- [EmqxBroker](#emqxbroker)
- [EmqxEnterprise](#emqxenterprise)
- [EmqxPlugin](#emqxplugin)



#### Condition



Condition saves the state information of the EMQX cluster



_Appears in:_
- [Status](#status)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[ConditionType](#conditiontype)_ | Status of cluster condition. |  |  |
| `status` _[ConditionStatus](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#conditionstatus-v1-core)_ | Status of the condition, one of True, False, Unknown. |  |  |
| `lastUpdateTime` _string_ | The last time this condition was updated. |  |  |
| `lastTransitionTime` _string_ | Last time the condition transitioned from one status to another. |  |  |
| `reason` _string_ | The reason for the condition's last transition. |  |  |
| `message` _string_ | A human readable message indicating details about the transition. |  |  |


#### ConditionType

_Underlying type:_ _string_

ConditionType defines the condition that the RF can have



_Appears in:_
- [Condition](#condition)





#### EmqxBroker



EmqxBroker is the Schema for the emqxbrokers API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `apps.emqx.io/v1beta3` | | |
| `kind` _string_ | `EmqxBroker` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[EmqxBrokerSpec](#emqxbrokerspec)_ |  |  |  |
| `status` _[Status](#status)_ |  |  |  |


#### EmqxBrokerModule







_Appears in:_
- [EmqxBrokerModuleList](#emqxbrokermodulelist)
- [EmqxBrokerTemplate](#emqxbrokertemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `enable` _boolean_ |  |  |  |




#### EmqxBrokerSpec



EmqxBrokerSpec defines the desired state of EmqxBroker



_Appears in:_
- [EmqxBroker](#emqxbroker)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `replicas` _integer_ |  | 3 |  |
| `imagePullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#localobjectreference-v1-core) array_ | ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.<br />If specified, these secrets will be passed to individual puller implementations for them to use.<br />More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod |  |  |
| `persistent` _[PersistentVolumeClaimSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#persistentvolumeclaimspec-v1-core)_ | Persistent describes the common attributes of storage devices |  |  |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#envvar-v1-core) array_ | List of environment variables to set in the container. |  |  |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#affinity-v1-core)_ | If specified, the pod's scheduling constraints |  |  |
| `toleRations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#toleration-v1-core) array_ | If specified, the pod's tolerations. |  |  |
| `nodeName` _string_ |  |  |  |
| `nodeSelector` _object (keys:string, values:string)_ | NodeSelector is a selector which must be true for the pod to fit on a node.<br />Selector which must match a node's labels for the pod to be scheduled on that node.<br />More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/ |  |  |
| `initContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#container-v1-core) array_ | List of initialization containers belonging to the pod.<br />Init containers are executed in order prior to containers being started. If any<br />init container fails, the pod is considered to have failed and is handled according<br />to its restartPolicy. The name for an init container or normal container must be<br />unique among all containers.<br />Init containers may not have Lifecycle actions, Readiness probes, Liveness probes, or Startup probes.<br />The resourceRequirements of an init container are taken into account during scheduling<br />by finding the highest request/limit for each resource type, and then using the max of<br />of that value or the sum of the normal containers. Limits are applied to init containers<br />in a similar fashion.<br />Init containers cannot currently be added or removed.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/init-containers/ |  |  |
| `extraContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#container-v1-core) array_ | ExtraContainers represents extra containers to be added to the pod.<br />See https://github.com/emqx/emqx-operator/issues/252 |  |  |
| `emqxTemplate` _[EmqxBrokerTemplate](#emqxbrokertemplate)_ |  |  |  |


#### EmqxBrokerTemplate







_Appears in:_
- [EmqxBrokerSpec](#emqxbrokerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `registry` _string_ | Registry will used for EMQX owner image,<br />like ${registry}/emqx/emqx and ${registry}/emqx/emqx-operator-reloader,<br />but it will not be used by other images, like sidecar container or else. |  |  |
| `image` _string_ |  |  | Required: {} <br /> |
| `imagePullPolicy` _[PullPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#pullpolicy-v1-core)_ | Image pull policy.<br />One of Always, Never, IfNotPresent.<br />Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/containers/images#updating-images |  |  |
| `username` _string_ | Username for EMQX Dashboard and API | admin |  |
| `password` _string_ | Password for EMQX Dashboard and API | public |  |
| `extraVolumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#volume-v1-core) array_ | See https://github.com/emqx/emqx-operator/pull/72 |  |  |
| `extraVolumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#volumemount-v1-core) array_ | See https://github.com/emqx/emqx-operator/pull/72 |  |  |
| `config` _[EmqxConfig](#emqxconfig)_ | Config represents the configurations of EMQX<br />More info: https://www.emqx.io/docs/en/v4.4/configuration/configuration.html |  |  |
| `args` _string array_ | Arguments to the entrypoint. The container image's CMD is used if this is not provided.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell |  |  |
| `securityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#podsecuritycontext-v1-core)_ | SecurityContext defines the security options the container should be run with.<br />If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.<br />More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/ |  |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#resourcerequirements-v1-core)_ | Compute Resources required by EMQX container.<br />More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ |  |  |
| `readinessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#probe-v1-core)_ | Periodic probe of container service readiness.<br />Container will be removed from service endpoints if the probe fails.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes |  |  |
| `livenessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#probe-v1-core)_ | Periodic probe of container liveness.<br />Container will be restarted if the probe fails.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes |  |  |
| `startupProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#probe-v1-core)_ | StartupProbe indicates that the Pod has successfully initialized.<br />If specified, no other probes are executed until this completes successfully.<br />If this probe fails, the Pod will be restarted, just as if the livenessProbe failed.<br />This can be used to provide different probe parameters at the beginning of a Pod's lifecycle,<br />when it might take a long time to load data or warm a cache, than during steady-state operation.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes |  |  |
| `serviceTemplate` _[ServiceTemplate](#servicetemplate)_ | ServiceTemplate defines a logical set of ports and a policy by which to access them |  |  |
| `acl` _string array_ | ACL defines ACL rules<br />More info: https://www.emqx.io/docs/en/v4.4/advanced/acl.html |  |  |
| `modules` _[EmqxBrokerModule](#emqxbrokermodule) array_ | Modules define functional modules for EMQX broker |  |  |


#### EmqxConfig

_Underlying type:_ _object_





_Appears in:_
- [EmqxBrokerTemplate](#emqxbrokertemplate)
- [EmqxEnterpriseTemplate](#emqxenterprisetemplate)



#### EmqxEnterprise



EmqxEnterprise is the Schema for the emqxEnterprises API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `apps.emqx.io/v1beta3` | | |
| `kind` _string_ | `EmqxEnterprise` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[EmqxEnterpriseSpec](#emqxenterprisespec)_ |  |  |  |
| `status` _[Status](#status)_ |  |  |  |


#### EmqxEnterpriseModule







_Appears in:_
- [EmqxEnterpriseModuleList](#emqxenterprisemodulelist)
- [EmqxEnterpriseTemplate](#emqxenterprisetemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `enable` _boolean_ |  |  |  |
| `configs` _[RawExtension](#rawextension)_ |  |  |  |




#### EmqxEnterpriseSpec



EmqxEnterpriseSpec defines the desired state of EmqxEnterprise



_Appears in:_
- [EmqxEnterprise](#emqxenterprise)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `replicas` _integer_ |  | 3 |  |
| `imagePullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#localobjectreference-v1-core) array_ | ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.<br />If specified, these secrets will be passed to individual puller implementations for them to use.<br />More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod |  |  |
| `persistent` _[PersistentVolumeClaimSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#persistentvolumeclaimspec-v1-core)_ | Persistent describes the common attributes of storage devices |  |  |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#envvar-v1-core) array_ | List of environment variables to set in the container. |  |  |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#affinity-v1-core)_ | If specified, the pod's scheduling constraints |  |  |
| `toleRations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#toleration-v1-core) array_ | If specified, the pod's tolerations. |  |  |
| `nodeName` _string_ |  |  |  |
| `nodeSelector` _object (keys:string, values:string)_ | NodeSelector is a selector which must be true for the pod to fit on a node.<br />Selector which must match a node's labels for the pod to be scheduled on that node.<br />More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/ |  |  |
| `initContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#container-v1-core) array_ | List of initialization containers belonging to the pod.<br />Init containers are executed in order prior to containers being started. If any<br />init container fails, the pod is considered to have failed and is handled according<br />to its restartPolicy. The name for an init container or normal container must be<br />unique among all containers.<br />Init containers may not have Lifecycle actions, Readiness probes, Liveness probes, or Startup probes.<br />The resourceRequirements of an init container are taken into account during scheduling<br />by finding the highest request/limit for each resource type, and then using the max of<br />of that value or the sum of the normal containers. Limits are applied to init containers<br />in a similar fashion.<br />Init containers cannot currently be added or removed.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/init-containers/ |  |  |
| `extraContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#container-v1-core) array_ | ExtraContainers represents extra containers to be added to the pod.<br />See https://github.com/emqx/emqx-operator/issues/252 |  |  |
| `emqxTemplate` _[EmqxEnterpriseTemplate](#emqxenterprisetemplate)_ |  |  |  |


#### EmqxEnterpriseTemplate







_Appears in:_
- [EmqxEnterpriseSpec](#emqxenterprisespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `registry` _string_ | Registry will used for EMQX owner image,<br />like ${registry}/emqx/emqx-ee and ${registry}/emqx/emqx-operator-reloader,<br />but it will not be used by other images, like sidecar container or else. |  |  |
| `image` _string_ |  |  | Required: {} <br /> |
| `imagePullPolicy` _[PullPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#pullpolicy-v1-core)_ | Image pull policy.<br />One of Always, Never, IfNotPresent.<br />Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/containers/images#updating-images |  |  |
| `username` _string_ | Username for EMQX Dashboard and API | admin |  |
| `password` _string_ | Password for EMQX Dashboard and API | public |  |
| `extraVolumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#volume-v1-core) array_ | See https://github.com/emqx/emqx-operator/pull/72 |  |  |
| `extraVolumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#volumemount-v1-core) array_ | See https://github.com/emqx/emqx-operator/pull/72 |  |  |
| `config` _[EmqxConfig](#emqxconfig)_ | Config represents the configurations of EMQX<br />More info: https://docs.emqx.com/en/enterprise/v4.4/configuration/configuration.html |  |  |
| `args` _string array_ | Arguments to the entrypoint. The container image's CMD is used if this is not provided.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell |  |  |
| `securityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#podsecuritycontext-v1-core)_ | SecurityContext defines the security options the container should be run with.<br />If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.<br />More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/ |  |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#resourcerequirements-v1-core)_ | Compute Resources required by EMQX container.<br />More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ |  |  |
| `readinessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#probe-v1-core)_ | Periodic probe of container service readiness.<br />Container will be removed from service endpoints if the probe fails.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes |  |  |
| `livenessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#probe-v1-core)_ | Periodic probe of container liveness.<br />Container will be restarted if the probe fails.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes |  |  |
| `startupProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#probe-v1-core)_ | StartupProbe indicates that the Pod has successfully initialized.<br />If specified, no other probes are executed until this completes successfully.<br />If this probe fails, the Pod will be restarted, just as if the livenessProbe failed.<br />This can be used to provide different probe parameters at the beginning of a Pod's lifecycle,<br />when it might take a long time to load data or warm a cache, than during steady-state operation.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes |  |  |
| `serviceTemplate` _[ServiceTemplate](#servicetemplate)_ | ServiceTemplate defines a logical set of ports and a policy by which to access them |  |  |
| `acl` _string array_ | ACL defines ACL rules<br />More info: https://docs.emqx.com/en/enterprise/v4.4/modules/internal_acl.html#builtin-acl-file-2 |  |  |
| `modules` _[EmqxEnterpriseModule](#emqxenterprisemodule) array_ | Modules define functional modules for EMQX Enterprise broker<br />More info: https://docs.emqx.com/en/enterprise/v4.4/modules/modules.html |  |  |
| `license` _[License](#license)_ | License for EMQX Enterprise broker |  |  |


#### EmqxNode







_Appears in:_
- [Status](#status)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `node` _string_ | EMQX node name |  |  |
| `node_status` _string_ | EMQX node status |  |  |
| `otp_release` _string_ | Erlang/OTP version used by EMQX |  |  |
| `version` _string_ | EMQX version |  |  |


#### EmqxPlugin



EmqxPlugin is the Schema for the emqxplugins API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `apps.emqx.io/v1beta3` | | |
| `kind` _string_ | `EmqxPlugin` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[EmqxPluginSpec](#emqxpluginspec)_ |  |  |  |
| `status` _[EmqxPluginStatus](#emqxpluginstatus)_ |  |  |  |


#### EmqxPluginSpec



EmqxPluginSpec defines the desired state of EmqxPlugin



_Appears in:_
- [EmqxPlugin](#emqxplugin)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `pluginName` _string_ | More info: https://www.emqx.io/docs/en/v4.4/advanced/plugins.html#list-of-plugins |  | Required: {} <br /> |
| `selector` _object (keys:string, values:string)_ | Selector matches the labels of the EMQX |  | Required: {} <br /> |
| `config` _object (keys:string, values:string)_ | Config defines the configurations of the EMQX plugins |  |  |


#### EmqxPluginStatus



EmqxPluginStatus defines the observed state of EmqxPlugin



_Appears in:_
- [EmqxPlugin](#emqxplugin)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `phase` _[phase](#phase)_ |  |  |  |






#### License







_Appears in:_
- [EmqxEnterpriseTemplate](#emqxenterprisetemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `data` _integer array_ | Data contains the secret data. Each key must consist of alphanumeric<br />characters, '-', '_' or '.'. The serialized form of the secret data is a<br />base64 encoded string, representing the arbitrary (possibly non-string)<br />data value here. Described in https://tools.ietf.org/html/rfc4648#section-4 |  |  |
| `stringData` _string_ | StringData allows specifying non-binary secret data in string form.<br />It is provided as a write-only input field for convenience.<br />All keys and values are merged into the data field on write, overwriting any existing values. |  |  |
| `secretName` _string_ | SecretName is the name of the secret in the pod's namespace to use.<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#secret |  |  |






#### ServiceTemplate







_Appears in:_
- [EmqxBrokerTemplate](#emqxbrokertemplate)
- [EmqxEnterpriseTemplate](#emqxenterprisetemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ServiceSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#servicespec-v1-core)_ |  |  |  |


#### Status



Emqx Status defines the observed state of EMQX



_Appears in:_
- [EmqxBroker](#emqxbroker)
- [EmqxEnterprise](#emqxenterprise)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](#condition) array_ | Represents the latest available observations of a EMQX current state. |  |  |
| `emqxNodes` _[EmqxNode](#emqxnode) array_ | Nodes of the EMQX cluster |  |  |
| `replicas` _integer_ | replicas is the number of Pods created by the EMQX Custom Resource controller. |  |  |
| `readyReplicas` _integer_ | readyReplicas is the number of pods created for this EMQX Custom Resource with a EMQX Ready. |  |  |


