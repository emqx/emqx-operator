# API Reference

## Packages
- [apps.emqx.io/v1beta2](#appsemqxiov1beta2)


## apps.emqx.io/v1beta2

Package v1beta2 contains API Schema definitions for the apps v1beta2 API group

### Resource Types
- [EmqxBroker](#emqxbroker)
- [EmqxEnterprise](#emqxenterprise)



#### Certificate





_Appears in:_
- [Listener](#listener)

| Field | Description |
| --- | --- |
| `wss` _[CertificateConf](#certificateconf)_ |  |
| `mqtts` _[CertificateConf](#certificateconf)_ |  |


#### CertificateConf





_Appears in:_
- [Certificate](#certificate)

| Field | Description |
| --- | --- |
| `data` _[CertificateData](#certificatedata)_ |  |
| `stringData` _[CertificateStringData](#certificatestringdata)_ |  |


#### CertificateData





_Appears in:_
- [CertificateConf](#certificateconf)

| Field | Description |
| --- | --- |
| `ca.crt` _integer array_ |  |
| `tls.crt` _integer array_ |  |
| `tls.key` _integer array_ |  |


#### CertificateStringData





_Appears in:_
- [CertificateConf](#certificateconf)

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
| `apiVersion` _string_ | `apps.emqx.io/v1beta2`
| `kind` _string_ | `EmqxBroker`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[EmqxBrokerSpec](#emqxbrokerspec)_ |  |
| `status` _[Status](#status)_ |  |


#### EmqxBrokerSpec



EmqxBrokerSpec defines the desired state of EmqxBroker

_Appears in:_
- [EmqxBroker](#emqxbroker)

| Field | Description |
| --- | --- |
| `replicas` _integer_ | The fields of Broker. The replicas of emqx broker |
| `image` _string_ |  |
| `imagePullPolicy` _[PullPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#pullpolicy-v1-core)_ |  |
| `imagePullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#localobjectreference-v1-core) array_ |  |
| `serviceAccountName` _string_ |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#resourcerequirements-v1-core)_ | The service account name which is being bind with the service account of the crd instance. |
| `storage` _[PersistentVolumeClaimSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#persistentvolumeclaimspec-v1-core)_ |  |
| `labels` _object (keys:string, values:string)_ | The labels configure must be specified. |
| `annotations` _object (keys:string, values:string)_ |  |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#affinity-v1-core)_ |  |
| `toleRations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#toleration-v1-core) array_ |  |
| `nodeName` _string_ |  |
| `nodeSelector` _object (keys:string, values:string)_ |  |
| `extraVolumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#volume-v1-core) array_ |  |
| `extraVolumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#volumemount-v1-core) array_ |  |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#envvar-v1-core) array_ |  |
| `securityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#podsecuritycontext-v1-core)_ |  |
| `emqxTemplate` _[EmqxBrokerTemplate](#emqxbrokertemplate)_ |  |
| `telegrafTemplate` _[TelegrafTemplate](#telegraftemplate)_ |  |


#### EmqxBrokerTemplate





_Appears in:_
- [EmqxBrokerSpec](#emqxbrokerspec)

| Field | Description |
| --- | --- |
| `listener` _[Listener](#listener)_ |  |
| `acl` _[ACL](#acl) array_ |  |
| `plugins` _[Plugin](#plugin) array_ |  |
| `modules` _[EmqxBrokerModule](#emqxbrokermodule) array_ |  |


#### EmqxEnterprise



EmqxEnterprise is the Schema for the emqxenterprises API



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apps.emqx.io/v1beta2`
| `kind` _string_ | `EmqxEnterprise`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[EmqxEnterpriseSpec](#emqxenterprisespec)_ |  |
| `status` _[Status](#status)_ |  |


#### EmqxEnterpriseSpec



EmqxEnterpriseSpec defines the desired state of EmqxEnterprise

_Appears in:_
- [EmqxEnterprise](#emqxenterprise)

| Field | Description |
| --- | --- |
| `replicas` _integer_ | The fields of Broker. The replicas of emqx broker |
| `image` _string_ |  |
| `imagePullPolicy` _[PullPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#pullpolicy-v1-core)_ |  |
| `imagePullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#localobjectreference-v1-core) array_ |  |
| `serviceAccountName` _string_ |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#resourcerequirements-v1-core)_ | The service account name which is being bind with the service account of the crd instance. |
| `storage` _[PersistentVolumeClaimSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#persistentvolumeclaimspec-v1-core)_ |  |
| `labels` _object (keys:string, values:string)_ | The labels configure must be specified. |
| `annotations` _object (keys:string, values:string)_ |  |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#affinity-v1-core)_ |  |
| `toleRations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#toleration-v1-core) array_ |  |
| `nodeName` _string_ |  |
| `nodeSelector` _object (keys:string, values:string)_ |  |
| `extraVolumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#volume-v1-core) array_ |  |
| `extraVolumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#volumemount-v1-core) array_ |  |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#envvar-v1-core) array_ |  |
| `securityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#podsecuritycontext-v1-core)_ |  |
| `emqxTemplate` _[EmqxEnterpriseTemplate](#emqxenterprisetemplate)_ |  |
| `telegrafTemplate` _[TelegrafTemplate](#telegraftemplate)_ |  |


#### EmqxEnterpriseTemplate





_Appears in:_
- [EmqxEnterpriseSpec](#emqxenterprisespec)

| Field | Description |
| --- | --- |
| `license` _string_ |  |
| `listener` _[Listener](#listener)_ |  |
| `acl` _[ACL](#acl) array_ |  |
| `plugins` _[Plugin](#plugin) array_ |  |
| `modules` _[EmqxEnterpriseModule](#emqxenterprisemodule) array_ |  |






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
| `ports` _[Ports](#ports)_ |  |
| `nodePorts` _[Ports](#ports)_ |  |
| `certificate` _[Certificate](#certificate)_ |  |




#### Ports





_Appears in:_
- [Listener](#listener)

| Field | Description |
| --- | --- |
| `mqtt` _integer_ |  |
| `mqtts` _integer_ |  |
| `ws` _integer_ |  |
| `wss` _integer_ |  |
| `dashboard` _integer_ |  |
| `api` _integer_ |  |


#### Status



Emqx Status defines the observed state of EMQX

_Appears in:_
- [EmqxBroker](#emqxbroker)
- [EmqxEnterprise](#emqxenterprise)

| Field | Description |
| --- | --- |
| `conditions` _[Condition](#condition) array_ | INSERT ADDITIONAL STATUS FIELD - define observed state of cluster Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html |


#### TelegrafTemplate





_Appears in:_
- [EmqxBrokerSpec](#emqxbrokerspec)
- [EmqxEnterpriseSpec](#emqxenterprisespec)

| Field | Description |
| --- | --- |
| `image` _string_ |  |
| `conf` _string_ |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#resourcerequirements-v1-core)_ |  |
| `imagePullPolicy` _[PullPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#pullpolicy-v1-core)_ |  |


