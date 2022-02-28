# EmqxBroker

`apiVersion: v1beta2`
`import: "github.com/emqx/emqx-operator/apis/apps/v1beta2"`

- **apiVersion**: v1beta2

- **kind**: EmqxBroker

- **metadata** ([ObjectMeta](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/object-meta/#ObjectMeta))

  Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata

- **spec** ([EmqxBrokerSpec](#emqxbrokerspec))

  Specification of the desired behavior of the pod. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status

- **status** (EmqxBrokerStatus)

  Most recently observed status of the EmqxBroker. This data may not be up to date. Populated by the system. Read-only. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status

## EmqxBrokerSpec

EmqxBrokerSpec is a description of a EmqxBroker.

------
+ **emqxTemplate** ([EmqxBrokerTemplate](#emqxbrokertemplate))

+ **telegrafTemplate** ([TelegrafTemplate](#telegraftemplate))

+ **replicas**(int32)

  replicas is the desired number of replicas of the given Template. These are replicas in the sense that they are instantiations of the same Template, but individual replicas also have a consistent identity. If unspecified, defaults to 3.

+ **image** (string), required

  Docker image name. More info: https://kubernetes.io/docs/concepts/containers/images This field is optional to allow higher level config management to default or override container images in workload controllers like Deployments and StatefulSets.

+ **imagePullPolicy** (string)

  Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. More info: https://kubernetes.io/docs/concepts/containers/images#updating-images

  Possible enum values:

  - `"Always"` means that kubelet always attempts to pull the latest image. Container will fail If the pull fails.
  - `"IfNotPresent"` means that kubelet pulls if the image isn't present on disk. Container will fail if the image isn't present and the pull fails.
  - `"Never"` means that kubelet never pulls an image, but only uses a local image. Container will fail if the image isn't present


+ **imagePullSecrets** ([][LocalObjectReference](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/local-object-reference/#LocalObjectReference))

  *Patch strategy: merge on key `name`*

  ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec. If specified, these secrets will be passed to individual puller implementations for them to use. For example, in the case of docker, only DockerConfig type secrets are honored. More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod

+ **serviceAccountName** (string)

  ServiceAccountName is the name of the ServiceAccount to use to run this pod. More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/

+ **resources** (ResourceRequirements)

  Compute Resources required by this container. Cannot be updated. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/

  *ResourceRequirements describes the compute resource requirements.*

  - **resources.limits** (map[string][Quantity](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/quantity/#Quantity))

    Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/

  - **resources.requests** (map[string][Quantity](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/quantity/#Quantity))

    Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/

+ **labels** (map[string]string)

  Map of string keys and values that can be used to organize and categorize (scope and select) objects. May match selectors of replication controllers and services. More info: http://kubernetes.io/docs/user-guide/labels

+ **annotations** (map[string]string)

  Annotations is an unstructured key value map stored with a resource that may be set by external tools to store and retrieve arbitrary metadata. They are not queryable and should be preserved when modifying objects. More info: http://kubernetes.io/docs/user-guide/annotations

+ **affinity** (Affinity)

  If specified, the pod's scheduling constraints

  *Affinity is a group of affinity scheduling rules.*

  - **affinity.nodeAffinity** ([NodeAffinity](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#NodeAffinity))

    Describes node affinity scheduling rules for the pod.

  - **affinity.podAffinity** ([PodAffinity](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#PodAffinity))

    Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).

  - **affinity.podAntiAffinity** ([PodAntiAffinity](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#PodAntiAffinity))

    Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).

+ **tolerations** ([]Toleration)

  If specified, the pod's tolerations.

  *The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator .*

  - **tolerations.key** (string)

    Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.

  - **tolerations.operator** (string)

    Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.

    Possible enum values:

    - `"Equal"`
    - `"Exists"`

  - **tolerations.value** (string)

    Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.

  - **tolerations.effect** (string)

    Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.

    Possible enum values:

    - `"NoExecute"` Evict any already-running pods that do not tolerate the taint. Currently enforced by NodeController.
    - `"NoSchedule"` Do not allow new pods to schedule onto the node unless they tolerate the taint, but allow all pods submitted to Kubelet without going through the scheduler to start, and allow all already-running pods to continue running. Enforced by the scheduler.
    - `"PreferNoSchedule"` Like TaintEffectNoSchedule, but the scheduler tries not to schedule new pods onto the node, rather than prohibiting new pods from scheduling onto the node entirely. Enforced by the scheduler.

  - **tolerations.tolerationSeconds** (int64)

    TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.

  - **nodeSelector** (map[string]string)

    NodeSelector is a selector which must be true for the pod to fit on a node. Selector which must match a node's labels for the pod to be scheduled on that node. More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/

  - **nodeName** (string)

    NodeName is a request to schedule this pod onto a specific node. If it is non-empty, the scheduler simply schedules this pod onto that node, assuming that it fits resource requirements.

+ **env** ([]EnvVar)

  *Patch strategy: merge on key `name`*

  List of environment variables to set in the container. Cannot be updated unless restart the container.

  *EnvVar represents an environment variable present in a Container.*

  - **env.name** (string), required

    Name of the environment variable. More info: https://www.emqx.io/docs/en/v4.4/configuration/environment-variable.html

  - **env.value** (string)

    Variable references \$(VAR_NAME) are expanded using the previously defined environment variables in the container and any service environment variables. If a variable cannot be resolved, the reference in the input string will be unchanged. Double \$\$ are reduced to a single \$, which allows for escaping the \$(VAR_NAME) syntax: i.e. "\$\$(VAR_NAME)" will produce the string literal "\$(VAR_NAME)". Escaped references will never be expanded, regardless of whether the variable exists or not. Defaults to "".

  - **env.valueFrom** (EnvVarSource)

    Source for the environment variable's value. Cannot be used if value is not empty.

    *EnvVarSource represents a source for the value of an EnvVar.*

    - **env.valueFrom.configMapKeyRef** (ConfigMapKeySelector)

      Selects a key of a ConfigMap.

      *Selects a key from a ConfigMap.*

      - **env.valueFrom.configMapKeyRef.key** (string), required

        The key to select.

      - **env.valueFrom.configMapKeyRef.name** (string)

        Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names

      - **env.valueFrom.configMapKeyRef.optional** (boolean)

        Specify whether the ConfigMap or its key must be defined

    - **env.valueFrom.fieldRef** ([ObjectFieldSelector](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/object-field-selector/#ObjectFieldSelector))

      Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['\<KEY>']`, `metadata.annotations['\<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.podIP, status.podIPs.

    - **env.valueFrom.resourceFieldRef** ([ResourceFieldSelector](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/resource-field-selector/#ResourceFieldSelector))

      Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.ephemeral-storage) are currently supported.

    - **env.valueFrom.secretKeyRef** (SecretKeySelector)

      Selects a key of a secret in the pod's namespace

      *SecretKeySelector selects a key of a Secret.*

      - **env.valueFrom.secretKeyRef.key** (string), required

        The key of the secret to select from. Must be a valid secret key.

      - **env.valueFrom.secretKeyRef.name** (string)

        Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names

      - **env.valueFrom.secretKeyRef.optional** (boolean)

        Specify whether the Secret or its key must be defined

+ **storage** ([PersistentVolumeClaimSpec](https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/persistent-volume-claim-v1/#PersistentVolumeClaimSpec))

  + **storage.accessModes** ([]string)

    AccessModes contains the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1

  + **storage.selector** ([LabelSelector](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/label-selector/#LabelSelector))

    A label query over volumes to consider for binding.

  + **storage.resources** (ResourceRequirements)

    Resources represents the minimum resources the volume should have. If RecoverVolumeExpansionFailure feature is enabled users are allowed to specify resource requirements that are lower than previous value but must still be higher than capacity recorded in the status field of the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources

    *ResourceRequirements describes the compute resource requirements.*

    - **storage.resources.limits** (map[string][Quantity](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/quantity/#Quantity))

      Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/

    - **storage.resources.requests** (map[string][Quantity](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/quantity/#Quantity))

      Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/

  + **storage.volumeName** (string)

    VolumeName is the binding reference to the PersistentVolume backing this claim.

  + **storage.storageClassName** (string)

    Name of the StorageClass required by the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1

  + **storage.volumeMode** (string)

    volumeMode defines what type of volume is required by the claim. Value of Filesystem is implied when not included in claim spec.

+ **extraVolumes**([]Volume)

  *Patch strategies: retainKeys, merge on key `name`*

  List of volumes that can be mounted by containers belonging to the pod. More info: https://kubernetes.io/docs/concepts/storage/volumes

+ **extraVolumeMounts** ([]VolumeMount)

  *Patch strategy: merge on key `mountPath`*

  Pod volumes to mount into the container's filesystem. Cannot be updated.

  *VolumeMount describes a mounting of a Volume within a container.*

  - **extraVolumeMounts.mountPath** (string), required

    Path within the container at which the volume should be mounted. Must not contain ':'.

  - **extraVolumeMounts.name** (string), required

    This must match the Name of a Volume.

  - **extraVolumeMounts.mountPropagation** (string)

    mountPropagation determines how mounts are propagated from the host to container and the other way around. When not set, MountPropagationNone is used. This field is beta in 1.10.

  - **extraVolumeMounts.readOnly** (boolean)

    Mounted read-only if true, read-write otherwise (false or unspecified). Defaults to false.

  - **extraVolumeMounts.subPath** (string)

    Path within the volume from which the container's volume should be mounted. Defaults to "" (volume's root).

  - **extraVolumeMounts.subPathExpr** (string)

    Expanded path within the volume from which the container's volume should be mounted. Behaves similarly to SubPath but environment variable references $(VAR_NAME) are expanded using the container's environment. Defaults to "" (volume's root). SubPathExpr and SubPath are mutually exclusive.

## EmqxBrokerTemplate

- **acl**
  - **acl.permission** (string), Required

    Possible enum values:

    - `"allow"`
    - `"deny"`

  - **acl.username** (string)

  - **acl.clientid** (string)

  - **acl.ipaddress** (string)

  - **acl.action** (string)

    Possible enum values:

    - `"publish"`
    - `"subscribe"`

  - **acl.topics**

    - **acl.topics.filter** ([]string)

    - **acl.topics.equal** ([]string)


- **plugins**

  - **plugins.name** (string)

    EMQX Broker plugins. More info: https://www.emqx.io/docs/en/v4.4/advanced/plugins.html

  - **plugins.enable** (bool)

- **modules**

  - **modules.name** (string)

    EMQX Broker modules. More info: https://www.emqx.io/docs/zh/v4.4/advanced/internal-modules.html

  - **modules.enable** (bool)

- **listener**

  + **listener.labels** (map[string]string)

    Map of string keys and values that can be used to organize and categorize (scope and select) objects. May match selectors of replication controllers and services. More info: http://kubernetes.io/docs/user-guide/labels

  + **listener.annotations** (map[string]string)

    Annotations is an unstructured key value map stored with a resource that may be set by external tools to store and retrieve arbitrary metadata. They are not queryable and should be preserved when modifying objects. More info: http://kubernetes.io/docs/user-guide/annotations

  + **listener.type** (string)

    type determines how the Service is exposed. Defaults to ClusterIP. Valid options are ExternalName, ClusterIP, NodePort, and LoadBalancer. "ClusterIP" allocates a cluster-internal IP address for load-balancing to endpoints. Endpoints are determined by the selector or if that is not specified, by manual construction of an Endpoints object or EndpointSlice objects. If clusterIP is "None", no virtual IP is allocated and the endpoints are published as a set of endpoints rather than a virtual IP. "NodePort" builds on ClusterIP and allocates a port on every node which routes to the same endpoints as the clusterIP. "LoadBalancer" builds on NodePort and creates an external load-balancer (if supported in the current cloud) which routes to the same endpoints as the clusterIP. "ExternalName" aliases this service to the specified externalName. Several other fields do not apply to ExternalName services. More info: https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types

    Possible enum values:

    - `"ClusterIP"` means a service will only be accessible inside the cluster, via the cluster IP.
    - `"ExternalName"` means a service consists of only a reference to an external name that kubedns or equivalent will return as a CNAME record, with no exposing or proxying of any pods involved.
    - `"LoadBalancer"` means a service will be exposed via an external load balancer (if the cloud provider supports it), in addition to 'NodePort' type.
    - `"NodePort"` means a service will be exposed on one port of every node, in addition to 'ClusterIP' type.

  + **listener.externalIPs** ([]string)

    externalIPs is a list of IP addresses for which nodes in the cluster will also accept traffic for this service. These IPs are not managed by Kubernetes. The user is responsible for ensuring that traffic arrives at a node with this IP. A common example is external load-balancers that are not part of the Kubernetes system.

  + **listener.loadBalancerIP** (string)

    Only applies to Service Type: LoadBalancer LoadBalancer will get created with the IP specified in this field. This feature depends on whether the underlying cloud-provider supports specifying the loadBalancerIP when a load balancer is created. This field will be ignored if the cloud-provider does not support the feature.

  + **listener.loadBalancerSourceRanges** ([]string)

    If specified and supported by the platform, this will restrict traffic through the cloud-provider load-balancer will be restricted to the specified client IPs. This field will be ignored if the cloud-provider does not support the feature." More info: https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/

  + **listener.loadBalancerSourceRanges** ([]string)

  + **listener.ports**

    + **listener.ports.mqtt** (int32)

    + **listener.ports.mqtts** (int32)

    + **listener.ports.ws** (int32)

    + **listener.ports.wss** (int32)

    + **listener.ports.api** (int32)

    + **listener.ports.dashboard** (int32)

  + **listener.nodePorts**

    + **listener.nodePorts.mqtt** (int32)

    + **listener.nodePorts.mqtts** (int32)

    + **listener.nodePorts.ws** (int32)

    + **listener.nodePorts.wss** (int32)

    + **listener.nodePorts.api** (int32)

    + **listener.nodePorts.dashboard** (int32)

  + **listener.certificate**

    + **listener.certificate.mqtts**

      + **listener.certificate.mqtts.data** (map[string][]byte)

        Data contains the secret data. Each key must consist of alphanumeric characters, '-', '_' or '.'. The serialized form of the secret data is a base64 encoded string, representing the arbitrary (possibly non-string) data value here. Described in https://tools.ietf.org/html/rfc4648#section-4

        + **listener.certificate.mqtts.data.'ca.cert'** ([]byte)

        + **listener.certificate.mqtts.data.'tls.cert'** ([]byte)

        + **listener.certificate.mqtts.data.'tls.key'** ([]byte)

      + **listener.certificate.mqtts.stringData** (map[string]string)

        stringData allows specifying non-binary secret data in string form. It is provided as a write-only input field for convenience. All keys and values are merged into the data field on write, overwriting any existing values. The stringData field is never output when reading from the API.

        + **listener.certificate.mqtts.stringData.'ca.cert'** ([]string)

        + **listener.certificate.mqtts.stringData.'tls.cert'** ([]string)

        + **listener.certificate.mqtts.stringData.'tls.key'** ([]string)

    + **listener.certificate.wss**

      + **listener.certificate.wss.data** (map[string][]byte)

        Data contains the secret data. Each key must consist of alphanumeric characters, '-', '_' or '.'. The serialized form of the secret data is a base64 encoded string, representing the arbitrary (possibly non-string) data value here. Described in https://tools.ietf.org/html/rfc4648#section-4

        + **listener.certificate.wss.data.'ca.cert'** ([]byte)

        + **listener.certificate.wss.data.'tls.cert'** ([]byte)

        + **listener.certificate.wss.data.'tls.key'** ([]byte)

      + **listener.certificate.wss.stringData** (map[string]string)

        stringData allows specifying non-binary secret data in string form. It is provided as a write-only input field for convenience. All keys and values are merged into the data field on write, overwriting any existing values. The stringData field is never output when reading from the API.

        + **listener.certificate.wss.stringData.'ca.cert'** ([]string)

        + **listener.certificate.wss.stringData.'tls.cert'** ([]string)

        + **listener.certificate.wss.stringData.'tls.key'** ([]string)

## TelegrafTemplate

+ **image** (string)

  Docker image name. More info: https://kubernetes.io/docs/concepts/containers/images This field is optional to allow higher level config management to default or override container images in workload controllers like Deployments and StatefulSets.

-  **config** (string)

