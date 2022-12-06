# Configure EMQX Service

## Task target
- Learn how to configure EMQX cluster Service through the serviceTemplate field

## EMQX Service configuration

EMQX CRD supports configuring the EMQX cluster Service by configuring the `.spec.emqxTemplate.serviceTemplate` field. For the specific description of the serviceTemplate field, please refer to: [serviceTemplate](https://github.com/emqx/emqx-operator/blob/2.0.2/docs/en_US/reference/v1beta3-reference.md#servicetemplate) ,For the use of Kubernetes Service, please refer to the document: [Service](https://kubernetes.io/docs/concepts/services-networking/service/), The serviceTemplate field is consistent with the Kubernetes Service semantics. The type field allows specifying the required Service type. For the value type of the type field, please refer to the document: [Service Type](https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types), The ports field defines the service ports

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  emqxTemplate:
    image: emqx/emqx-ee:4.4.8
    serviceTemplate:
      metadata:
        name: emqx-ee
        namespace: default
        labels:
          "apps.emqx.io/instance": "emqx-ee"
      spec:
        type: NodePort
        selector:
          "apps.emqx.io/instance": "emqx-ee"
        ports:
          - name: "http-management-8081"
            port: 8081
            protocol: "TCP"
            targetPort: 8081
          - name: "http-dashboard-18083"
            port: 18083
            protocol: "TCP"
            targetPort: 18083
            nodePort: 30653
          - name: "mqtt-tcp-1883"
            protocol: "TCP"
            port: 1883
            targetPort: 1883
            nodePort: 30654
          - name: "mqtt-tcp-11883"
            protocol: "TCP"
            port: 11883
            targetPort: 11883
            nodePort: 30655
          - name: "mqtt-ws-8083"
            protocol: "TCP"
            port: 8083
            targetPort: 8083
            nodePort: 30656
          - name: "mqtt-wss-8084"
            protocol: "TCP"
            port: 8084
            targetPort: 8084
            nodePort: 30657
```

Save the above content as: emqx-service.yaml and deploy the EMQX cluster

```
kubectl apply -f emqx-service.yaml
```

- Get EMQX service information

```
kubectl get service -l apps.emqx.io/instance=emqx-ee
```

The output is similar to:

```
NAME               TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)                                                                                       AGE
emqx-ee            NodePort    10.103.174.230   <none>        8081:31126/TCP,18083:30653/TCP,1883:30654/TCP,11883:30655/TCP,8083:30656/TCP,8084:30657/TCP   39s                                                                                    39s
```

**Remarks**: EMQX Operator supports automatic discovery of EMQX listener, EMQX Operator will associate the EMQX listener with the configuration `.spec.serviceTemplate.spec.ports[].port` through the name field, when the port name configured by the user and EMQX When the name of the listener is the same, the port configured by the user will be used first.

- EMQX built-in listener

    |        name            |     port        | 
    | :--------------------- |:---------------:|
    | http-management-8081   |    8081         |   
    | http-dashboard-18083   |    18083        |
    | mqtt-tcp-1883          |    1883         |
    | mqtt-tcp-11883         |    11883        |
    | mqtt-ws-8083           |    8083         |
    | mqtt-wss-8084          |    8084         |
