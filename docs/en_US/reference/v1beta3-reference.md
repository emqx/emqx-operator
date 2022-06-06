# API Reference

## Packages
- [apps.emqx.io/v1beta3](#appsemqxiov1beta3)


## apps.emqx.io/v1beta3

Package v1beta3 contains API Schema definitions for the apps v1beta3 API group

### Resource Types
- [EmqxBroker](#emqxbroker)
- [EmqxEnterprise](#emqxenterprise)



#### ACL





_Appears in:_
- [EmqxBrokerTemplate](#emqxbrokertemplate)
- [EmqxEnterpriseTemplate](#emqxenterprisetemplate)

| Field | Description |
| --- | --- |
| `permission` _string_ |  |
| `username` _string_ |  |
| `clientid` _string_ |  |
| `ipaddress` _string_ |  |
| `action` _string_ |  |
| `topics` _[Topics](#topics)_ |  |




#### CertConf





_Appears in:_
- [ListenerPort](#listenerport)

| Field | Description |
| --- | --- |
| `data` _[CertData](#certdata)_ |  |
| `stringData` _[CertStringData](#certstringdata)_ |  |


#### CertData





_Appears in:_
- [Cert](#cert)
- [CertConf](#certconf)

| Field | Description |
| --- | --- |
| `ca.crt` _integer array_ |  |
| `tls.crt` _integer array_ |  |
| `tls.key` _integer array_ |  |


#### CertStringData





_Appears in:_
- [Cert](#cert)
- [CertConf](#certconf)

| Field | Description |
| --- | --- |
| `ca.crt` _string_ |  |
| `tls.crt` _string_ |  |
| `tls.key` _string_ |  |


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
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#envvar-v1-core) array_ |  |
| `args` _string array_ |  |
| `securityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#podsecuritycontext-v1-core)_ |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#resourcerequirements-v1-core)_ |  |
| `readinessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#probe-v1-core)_ |  |
| `livenessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#probe-v1-core)_ |  |
| `startupProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#probe-v1-core)_ |  |
| `listener` _[Listener](#listener)_ |  |
| `acl` _[ACL](#acl) array_ |  |
| `plugins` _[Plugin](#plugin) array_ |  |
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
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#envvar-v1-core) array_ |  |
| `args` _string array_ |  |
| `securityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#podsecuritycontext-v1-core)_ |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#resourcerequirements-v1-core)_ |  |
| `readinessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#probe-v1-core)_ |  |
| `livenessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#probe-v1-core)_ |  |
| `startupProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#probe-v1-core)_ |  |
| `listener` _[Listener](#listener)_ |  |
| `acl` _[ACL](#acl) array_ |  |
| `plugins` _[Plugin](#plugin) array_ |  |
| `modules` _[EmqxEnterpriseModule](#emqxenterprisemodule) array_ |  |
| `license` _[License](#license)_ |  |








#### License





_Appears in:_
- [EmqxEnterpriseTemplate](#emqxenterprisetemplate)

| Field | Description |
| --- | --- |
| `data` _integer array_ |  |
| `stringData` _string_ |  |


#### Listener





_Appears in:_
- [EmqxBrokerTemplate](#emqxbrokertemplate)
- [EmqxEnterpriseTemplate](#emqxenterprisetemplate)

| Field | Description |
| --- | --- |
| `labels` _object (keys:string, values:string)_ |  |
| `annotations` _object (keys:string, values:string)_ |  |
| `type` _[ServiceType](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#servicetype-v1-core)_ |  |
| `loadBalancerIP` _string_ |  |
| `loadBalancerSourceRanges` _string array_ |  |
| `externalIPs` _string array_ |  |
| `api` _[ListenerPort](#listenerport)_ |  |
| `dashboard` _[ListenerPort](#listenerport)_ |  |
| `mqtt` _[ListenerPort](#listenerport)_ |  |
| `mqtts` _[ListenerPort](#listenerport)_ |  |
| `ws` _[ListenerPort](#listenerport)_ |  |
| `wss` _[ListenerPort](#listenerport)_ |  |


#### ListenerPort





_Appears in:_
- [Listener](#listener)

| Field | Description |
| --- | --- |
| `port` _integer_ |  |
| `nodePort` _integer_ |  |
| `cert` _[CertConf](#certconf)_ |  |




#### Plugin





_Appears in:_
- [EmqxBrokerTemplate](#emqxbrokertemplate)
- [EmqxEnterpriseTemplate](#emqxenterprisetemplate)
- [PluginList](#pluginlist)

| Field | Description |
| --- | --- |
| `name` _string_ |  |
| `enable` _boolean_ |  |




#### Status



Emqx Status defines the observed state of EMQX

_Appears in:_
- [EmqxBroker](#emqxbroker)
- [EmqxEnterprise](#emqxenterprise)

| Field | Description |
| --- | --- |
| `conditions` _[Condition](#condition) array_ | INSERT ADDITIONAL STATUS FIELD - define observed state of cluster Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html |


#### Topics





_Appears in:_
- [ACL](#acl)

| Field | Description |
| --- | --- |
| `filter` _string array_ |  |
| `equal` _string array_ |  |


