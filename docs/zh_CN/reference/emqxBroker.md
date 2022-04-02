# EmqxBroker

`apiVersion: v1beta2`
`import: "github.com/emqx/emqx-operator/apis/apps/v1beta2"`

- **apiVersion**: v1beta2

- **kind**: EmqxBroker

- **metadata** ([ObjectMeta](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/object-meta/#ObjectMeta))

  标准对象的元数据。更多信息: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata

- **spec** ([EmqxBrokerSpec](#emqxbrokerspec))

  容器的所需行为的规范。更多信息: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status

- **status** (EmqxBrokerStatus)

  最近观察到的EmqxBroker状态。此数据可能不是最新的。由系统填充。只读。更多信息: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status

## EmqxBrokerSpec

EmqxBrokerSpec is a description of a EmqxBroker.

------
+ **emqxTemplate** ([EmqxBrokerTemplate](#emqxbrokertemplate))

+ **telegrafTemplate** ([TelegrafTemplate](#telegraftemplate))

+ **replicas**(int32)

+ **image** (string), required

+ **imagePullPolicy** (string)

  更多信息: https://kubernetes.io/docs/concepts/containers/images#updating-images

  枚举值:

  - `"Always"`
  - `"IfNotPresent"`
  - `"Never"`


+ **imagePullSecrets** ([][LocalObjectReference](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/local-object-reference/#LocalObjectReference))

  更多信息:  https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod

+ **serviceAccountName** (string)

  更多信息:  https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/

+ **resources** (ResourceRequirements)

  更多信息:  https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/

  - **resources.limits** (map[string][Quantity](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/quantity/#Quantity))

    更多信息:  https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/

  - **resources.requests** (map[string][Quantity](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/quantity/#Quantity))

    更多信息:   https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/

+ **labels** (map[string]string)

  更多信息:   http://kubernetes.io/docs/user-guide/labels

+ **annotations** (map[string]string)

  更多信息:   http://kubernetes.io/docs/user-guide/annotations

+ **affinity** (Affinity)

  - **affinity.nodeAffinity** ([NodeAffinity](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#NodeAffinity))

  - **affinity.podAffinity** ([PodAffinity](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#PodAffinity))

  - **affinity.podAntiAffinity** ([PodAntiAffinity](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#PodAntiAffinity))

+ **tolerations** ([]Toleration)

  - **tolerations.key** (string)

    **tolerations.operator** (string)

    枚举值:

    - `"Equal"`
    - `"Exists"`

- **tolerations.value** (string)

- **tolerations.effect** (string)

  枚举值:

    - `"NoExecute"`
    - `"NoSchedule"`
    - `"PreferNoSchedule"`

  - **tolerations.tolerationSeconds** (int64)

  - **nodeSelector** (map[string]string)

     更多信息:  https://kubernetes.io/docs/concepts/configuration/assign-pod-node/

  - **nodeName** (string)

+ **env** ([]EnvVar)

  - **env.name** (string), required

    更多信息: https://www.emqx.io/docs/zh/v4.4/configuration/environment-variable.html

  - **env.value** (string)

  - **env.valueFrom** (EnvVarSource)

    - **env.valueFrom.configMapKeyRef** (ConfigMapKeySelector)

      - **env.valueFrom.configMapKeyRef.key** (string), required

      - **env.valueFrom.configMapKeyRef.name** (string)

        更多信息:  https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names

      - **env.valueFrom.configMapKeyRef.optional** (boolean)

    - **env.valueFrom.fieldRef** ([ObjectFieldSelector](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/object-field-selector/#ObjectFieldSelector))

    - **env.valueFrom.resourceFieldRef** ([ResourceFieldSelector](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/resource-field-selector/#ResourceFieldSelector))

    - **env.valueFrom.secretKeyRef** (SecretKeySelector)

      - **env.valueFrom.secretKeyRef.key** (string), required

      - **env.valueFrom.secretKeyRef.name** (string)

        更多信息:  https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names

      - **env.valueFrom.secretKeyRef.optional** (boolean)

+ **storage** ([PersistentVolumeClaimSpec](https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/persistent-volume-claim-v1/#PersistentVolumeClaimSpec))

  + **storage.accessModes** ([]string)

    更多信息:  https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1

  + **storage.selector** ([LabelSelector](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/label-selector/#LabelSelector))

  + **storage.resources** (ResourceRequirements)

    更多信息:  https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources

    - **storage.resources.limits** (map[string][Quantity](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/quantity/#Quantity))

      更多信息:  https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/

    - **storage.resources.requests** (map[string][Quantity](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/quantity/#Quantity))

      更多信息:  https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/

  + **storage.volumeName** (string)

  + **storage.storageClassName** (string)

    更多信息:  https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1

  + **storage.volumeMode** (string)

+ **extraVolumes**([]Volume)

  更多信息:  https://kubernetes.io/docs/concepts/storage/volumes

+ **extraVolumeMounts** ([]VolumeMount)

  - **extraVolumeMounts.mountPath** (string), required

  - **extraVolumeMounts.name** (string), required

  - **extraVolumeMounts.mountPropagation** (string)

  - **extraVolumeMounts.readOnly** (boolean)

  - **extraVolumeMounts.subPath** (string)

  - **extraVolumeMounts.subPathExpr** (string)

- **securityContext** (PodSecurityContext)
  
  更多信息: https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#security-context

## EmqxBrokerTemplate

- **acl**
  - **acl.permission** (string), Required

    枚举值:

    - `"allow"`
    - `"deny"`

  - **acl.username** (string)

  - **acl.clientid** (string)

  - **acl.ipaddress** (string)

  - **acl.action** (string)

    枚举值:

    - `"publish"`
    - `"subscribe"`

  - **acl.topics**

    - **acl.topics.filter** ([]string)

    - **acl.topics.equal** ([]string)


- **plugins**

  - **plugins.name** (string)

    EMQX Broker 插件。更多信息:  https://www.emqx.io/docs/zh/v4.4/advanced/plugins.html

  - **plugins.enable** (bool)

- **modules**

  - **modules.name** (string)

    EMQX Broker 内置模块。更多信息: https://www.emqx.io/docs/zh/v4.4/advanced/internal-modules.html

  - **modules.enable** (bool)

- **listener**

  + **listener.labels** (map[string]string)

    更多信息:  http://kubernetes.io/docs/user-guide/labels

  + **listener.annotations** (map[string]string)

    更多信息:  http://kubernetes.io/docs/user-guide/annotations

  + **listener.type** (string)

    更多信息:  https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types

    枚举值:

    - `"ClusterIP"`
    - `"ExternalName"`
    - `"LoadBalancer"`
    - `"NodePort"`

  + **listener.externalIPs** ([]string)

  + **listener.loadBalancerIP** (string)

  + **listener.loadBalancerSourceRanges** ([]string)

    更多信息:  https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/

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

        + **listener.certificate.mqtts.data.'ca.cert'** ([]byte)

          base64 编码的字符串。

        + **listener.certificate.mqtts.data.'tls.cert'** ([]byte)

          base64 编码的字符串。

        + **listener.certificate.mqtts.data.'tls.key'** ([]byte)

          base64 编码的字符串。

      + **listener.certificate.mqtts.stringData** (map[string]string)

        + **listener.certificate.mqtts.stringData.'ca.cert'** ([]string)

          原始字符串。

        + **listener.certificate.mqtts.stringData.'tls.cert'** ([]string)

          原始字符串。

        + **listener.certificate.mqtts.stringData.'tls.key'** ([]string)

          原始字符串。

    + **listener.certificate.wss**

      + **listener.certificate.wss.data** (map[string][]byte)

        + **listener.certificate.wss.data.'ca.cert'** ([]byte)

          base64 编码的字符串。

        + **listener.certificate.wss.data.'tls.cert'** ([]byte)

          base64 编码的字符串。

        + **listener.certificate.wss.data.'tls.key'** ([]byte)

          base64 编码的字符串。

      + **listener.certificate.wss.stringData** (map[string]string)

        + **listener.certificate.wss.stringData.'ca.cert'** ([]string)

          原始字符串。

        + **listener.certificate.wss.stringData.'tls.cert'** ([]string)

          原始字符串。

        + **listener.certificate.wss.stringData.'tls.key'** ([]string)

          原始字符串。

## TelegrafTemplate

+ **image** (string), required

  更多信息:  https://kubernetes.io/docs/concepts/containers/images

- **config** (string), required

  Telegraf 的配置
