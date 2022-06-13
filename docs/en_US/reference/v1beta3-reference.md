# API Reference

## Packages
- [apps.emqx.io/v1beta3](#appsemqxiov1beta3)


## apps.emqx.io/v1beta3

Package v1beta3 contains API Schema definitions for the apps v1beta3 API group

### Resource Types
- [EmqxBroker](#emqxbroker)
- [EmqxEnterprise](#emqxenterprise)
- [EmqxPlugin](#emqxplugin)
- [EmqxPluginList](#emqxpluginlist)



#### Condition



Condition saves the state information of the EMQX cluster

_Appears in:_
- [Status](#status)

| Field | Description |
| --- | --- |
| `type` _[ConditionType](#conditiontype)_ | Status of cluster condition. |
| `lastUpdateTime` _string_ | The last time this condition was updated. |
| `lastTransitionTime` _string_ | Last time the condition transitioned from one status to another. |
| `reason` _string_ | The reason for the condition's last transition. |
| `message` _string_ | A human readable message indicating details about the transition. |


#### ConditionType

_Underlying type:_ `string`

ConditionType defines the condition that the RF can have

_Appears in:_
- [Condition](#condition)





#### EmqxBroker



EmqxBroker is the Schema for the emqxbrokers API



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apps.emqx.io/v1beta3`
| `kind` _string_ | `EmqxBroker`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[EmqxBrokerSpec](#emqxbrokerspec)_ |  |
| `status` _[Status](#status)_ |  |


#### EmqxBrokerModule





_Appears in:_
- [EmqxBrokerModuleList](#emqxbrokermodulelist)
- [EmqxBrokerTemplate](#emqxbrokertemplate)

| Field | Description |
| --- | --- |
| `name` _string_ |  |
| `enable` _boolean_ |  |




#### EmqxBrokerSpec



EmqxBrokerSpec defines the desired state of EmqxBroker

_Appears in:_
- [EmqxBroker](#emqxbroker)

| Field | Description |
| --- | --- |
| `replicas` _integer_ |  |
| `imagePullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#localobjectreference-v1-core) array_ |  |
| `persistent` _[PersistentVolumeClaimSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#persistentvolumeclaimspec-v1-core)_ |  |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#envvar-v1-core) array_ |  |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#affinity-v1-core)_ |  |
| `toleRations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#toleration-v1-core) array_ |  |
| `nodeName` _string_ |  |
| `nodeSelector` _object (keys:string, values:string)_ |  |
| `initContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#container-v1-core) array_ |  |
| `emqxTemplate` _[EmqxBrokerTemplate](#emqxbrokertemplate)_ |  |


#### EmqxBrokerTemplate





_Appears in:_
- [EmqxBrokerSpec](#emqxbrokerspec)

| Field | Description |
| --- | --- |
| `image` _string_ |  |
| `imagePullPolicy` _[PullPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#pullpolicy-v1-core)_ |  |
| `extraVolumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#volume-v1-core) array_ |  |
| `extraVolumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#volumemount-v1-core) array_ |  |
| `config` _object (keys:string, values:string)_ |  |
| `args` _string array_ |  |
| `securityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#podsecuritycontext-v1-core)_ |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#resourcerequirements-v1-core)_ |  |
| `readinessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#probe-v1-core)_ |  |
| `livenessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#probe-v1-core)_ |  |
| `startupProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#probe-v1-core)_ |  |
| `serviceTemplate` _[ServiceTemplate](#servicetemplate)_ |  |
| `acl` _string array_ |  |
| `modules` _[EmqxBrokerModule](#emqxbrokermodule) array_ |  |




#### EmqxEnterprise



EmqxEnterprise is the Schema for the emqxEnterprises API



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apps.emqx.io/v1beta3`
| `kind` _string_ | `EmqxEnterprise`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[EmqxEnterpriseSpec](#emqxenterprisespec)_ |  |
| `status` _[Status](#status)_ |  |


#### EmqxEnterpriseModule





_Appears in:_
- [EmqxEnterpriseModuleList](#emqxenterprisemodulelist)
- [EmqxEnterpriseTemplate](#emqxenterprisetemplate)

| Field | Description |
| --- | --- |
| `name` _string_ |  |
| `enable` _boolean_ |  |
| `configs` _[RawExtension](#rawextension)_ |  |




#### EmqxEnterpriseSpec



EmqxEnterpriseSpec defines the desired state of EmqxEnterprise

_Appears in:_
- [EmqxEnterprise](#emqxenterprise)

| Field | Description |
| --- | --- |
| `replicas` _integer_ |  |
| `imagePullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#localobjectreference-v1-core) array_ |  |
| `persistent` _[PersistentVolumeClaimSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#persistentvolumeclaimspec-v1-core)_ |  |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#envvar-v1-core) array_ |  |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#affinity-v1-core)_ |  |
| `toleRations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#toleration-v1-core) array_ |  |
| `nodeName` _string_ |  |
| `nodeSelector` _object (keys:string, values:string)_ |  |
| `initContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#container-v1-core) array_ |  |
| `emqxTemplate` _[EmqxEnterpriseTemplate](#emqxenterprisetemplate)_ |  |


#### EmqxEnterpriseTemplate





_Appears in:_
- [EmqxEnterpriseSpec](#emqxenterprisespec)

| Field | Description |
| --- | --- |
| `image` _string_ |  |
| `imagePullPolicy` _[PullPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#pullpolicy-v1-core)_ |  |
| `extraVolumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#volume-v1-core) array_ |  |
| `extraVolumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#volumemount-v1-core) array_ |  |
| `config` _object (keys:string, values:string)_ |  |
| `args` _string array_ |  |
| `securityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#podsecuritycontext-v1-core)_ |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#resourcerequirements-v1-core)_ |  |
| `readinessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#probe-v1-core)_ |  |
| `livenessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#probe-v1-core)_ |  |
| `startupProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#probe-v1-core)_ |  |
| `serviceTemplate` _[ServiceTemplate](#servicetemplate)_ |  |
| `acl` _string array_ |  |
| `modules` _[EmqxEnterpriseModule](#emqxenterprisemodule) array_ |  |
| `license` _[License](#license)_ |  |


#### EmqxPlugin



EmqxPlugin is the Schema for the emqxplugins API

_Appears in:_
- [EmqxPluginList](#emqxpluginlist)

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apps.emqx.io/v1beta3`
| `kind` _string_ | `EmqxPlugin`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[EmqxPluginSpec](#emqxpluginspec)_ |  |


#### EmqxPluginList



EmqxPluginList contains a list of EmqxPlugin



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apps.emqx.io/v1beta3`
| `kind` _string_ | `EmqxPluginList`
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[EmqxPlugin](#emqxplugin) array_ |  |


#### EmqxPluginSpec



EmqxPluginSpec defines the desired state of EmqxPlugin

_Appears in:_
- [EmqxPlugin](#emqxplugin)

| Field | Description |
| --- | --- |
| `pluginName` _string_ |  |
| `selector` _object (keys:string, values:string)_ |  |
| `config` _object (keys:string, values:string)_ |  |








#### License





_Appears in:_
- [EmqxEnterpriseTemplate](#emqxenterprisetemplate)

| Field | Description |
| --- | --- |
| `data` _integer array_ |  |
| `stringData` _string_ |  |




#### ServiceTemplate





_Appears in:_
- [EmqxBrokerTemplate](#emqxbrokertemplate)
- [EmqxEnterpriseTemplate](#emqxenterprisetemplate)

| Field | Description |
| --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[ServiceSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#servicespec-v1-core)_ |  |


#### Status



Emqx Status defines the observed state of EMQX

_Appears in:_
- [EmqxBroker](#emqxbroker)
- [EmqxEnterprise](#emqxenterprise)

| Field | Description |
| --- | --- |
| `conditions` _[Condition](#condition) array_ | INSERT ADDITIONAL STATUS FIELD - define observed state of cluster Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html |


