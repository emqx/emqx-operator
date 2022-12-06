# 配置 EMQX Dashboard 账号

## 任务目标
- 学习如何通过 username 和 password 字段配置 EMQX Dashboard 账号

## EMQX Dashboard 配置
EMQX CRD 支持通过 `.spec.emqxTemplate.username` 和 `.spec.emqxTemplate.passowrd` 字段来配置 EMQX 集群 Dashboard 账号

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  emqxTemplate:
    username: test
    password: test
    image: emqx/emqx-ee:4.4.8
```

将上述文件内容保存为: emqx-dashboard.yaml 并部署 EMQX 集群

```
kubectl apply -f emqx-dashboard.yaml
```

- 查看 EMQX 集群是否正常运行

```
kubectl get pods  -l  apps.emqx.io/instance=emqx-ee
```

输出类似于：

```
NAME        READY   STATUS    RESTARTS   AGE
emqx-ee-0   2/2     Running   0          48m
emqx-ee-1   2/2     Running   0          48m
emqx-ee-2   2/2     Running   0          48m
```

- 使用端口转发来访问 EMQX 集群 Dashboard

```
kubectl port-forward  service/emqx-ee 32010:18083
```

备注：待集群就绪之后就可以使用配置的 username 和 password 登录 EMQX Dashboard
