# 配置 EMQX Service

## 任务目标

- 学习如何通过 serviceTemplate 字段配置 EMQX 集群 Service

## EMQX Service 配置

EMQX CRD 支持通过来配置 `.spec.emqxTemplate.serviceTemplate` 字段来配置 EMQX 集群 Service，serviceTemplate 字段的具体描述可以参考：[serviceTemplate](https://github.com/emqx/emqx-operator/blob/main/docs/en_US/reference/v1beta3-reference.md#servicetemplate)，关于 Kubernetes Service 的使用可以参考文档：[Service](https://kubernetes.io/zh-cn/docs/concepts/services-networking/service/)， serviceTemplate 字段与 Kubernetes Service 语义一致，type 字段允许指定需要的 Service 类型， type字段的取值类型可以参考文档：[服务类型（Service)](https://kubernetes.io/zh-cn/docs/concepts/services-networking/service/#publishing-services-service-types)，ports 字段定义了服务端口

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

将上述内容保存为：emqx-service.yaml 并部署 EMQX 集群

```
kubectl apply -f emqx-service.yaml
```

- 获取 EMQX service 信息

```
kubectl get service -l apps.emqx.io/instance=emqx-ee
```

输出类似于：

```
NAME               TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)                                                                                       AGE
emqx-ee            NodePort    10.103.174.230   <none>        8081:31126/TCP,18083:30653/TCP,1883:30654/TCP,11883:30655/TCP,8083:30656/TCP,8084:30657/TCP   39s                                                                                    39s
```