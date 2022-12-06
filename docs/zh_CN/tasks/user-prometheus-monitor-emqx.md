# 使用 Prometheus 监控 EMQX 集群

## 任务目标
- 学习如何通过 Prometheus 监控 EMQX 集群

## 部署 Prometheus 
Prometheus 部署文档可以参考：[Prometheus 部署文档](https://github.com/prometheus-operator/prometheus-operator)

## 部署 EMQX 集群
EMQX 支持通过 http 接口对外暴露指标，集群下所有统计指标数据可以参考文档：[HTTP API](https://www.emqx.io/docs/zh/v4.4/advanced/http-api.html#%E7%BB%9F%E8%AE%A1%E6%8C%87%E6%A0%87) 

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
          - name: "dashboard"
            port: 18083
            protocol: "TCP"
            targetPort: 18083
            nodePort: 32053
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
```
将上述内容保存为：emqx.yaml 并部署 EMQX 集群

```
kubectl apply -f emqx.yaml
```

## 配置 ServiceMonitor 

ServiceMonitor 自定义资源定义 (CRD) 允许以声明方式定义应如何监视一组动态服务。 使用标签选择来定义选择哪些服务以使用所需配置进行监视，其文档可以参考：[ServiceMonitor 文档](https://github.com/prometheus-operator/prometheus-operator/blob/main/Documentation/design.md#servicemonitor)

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: emqx
  namespace: default
  labels:
    app.kubernetes.io/name: emqx-ee
spec:
  endpoints:
  - interval: 10s
    port: 18083
    scheme: http
    path: /api/v4/emqx_prometheus
    params:
      type:
        - prometheus
    basicAuth:
      password:
        name: emqx-basic-auth
        key: password
      username:
        name: emqx-basic-auth
        key: username
  jobLabel: emqx-scraping
  namespaceSelector:
    matchNames:
      -  default
  selector:
    matchLabels:
      apps.emqx.io/instance: emqx-ee
```
将上述内容保存为：serviceMonitor.yaml 并创建 ServiceMonitor 

```
kubectl apply -f serviceMonitor.yaml
```
使用 basicAuth 为 ServiceMonitors 进行身份验证

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: emqx-basic-auth
  namespace: default
type: kubernetes.io/basic-auth
stringData:
  username: admin
  password: public
```
将上述内容保存为：secret.yaml 并创建 Secret

```
kubectl apply -f secret.yaml
```

## 访问 Promethus 查看是否有收集 EMQX 集群的指标
打开 Prometheus 的界面，切换到 Graph 页面，输入 emqx 显示如下图

![](./assets/Prometheus-Graph.png)

切换到 Status → Targets 页面，显示如下图，可以看到集群中所有被监控的 EMQX Pod 信息  

![](./assets/Prometheus-Targets.png)

 

