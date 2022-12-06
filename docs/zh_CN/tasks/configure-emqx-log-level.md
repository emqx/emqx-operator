# 配置 EMQX 日志等级

## 任务目标

- 学习如何通过 env 字段配置 EMQX 日志等级
- 学习如何通过 config 字段配置 EMQX 日志等级

## EMQX 日志等级配置

EMQX CRD 支持通过 `.spec.env` 字段来配置 EMQX 集群日志等级， env 字段的具体描述可以参考：[env 字段描述](https://github.com/emqx/emqx-operator/blob/2.0.2/docs/en_US/reference/v1beta3-reference.md),  也支持通过 `.spec.emqxTemplate.config` 字段配置日志等级，config 字段的具体描述可以参考：[config 字段描述](https://github.com/emqx/emqx-operator/blob/2.0.2/docs/en_US/reference/v1beta3-reference.md#emqxenterprisespec)， 这两种方式本质上没有区别，最终都会为环境变量 EMQX 配置日志等级，EMQX 环境变量的配置可以参考：[使用环境变量修改配置](https://www.emqx.io/docs/zh/v4/configuration/configuration.html) 

### 通过 env 字段配置 EMQX 集群日志等级

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  env:
    - name: EMQX_LOG__LEVEL
      value: debug
  emqxTemplate:
    image: emqx/emqx-ee:4.4.8
```

将上述内容保存为：emqx-log.yaml 并部署 EMQX 集群

```
kubectl apply -f emqx-log.yaml
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

```
kubectl logs emqx-ee-0
```

### 通过 config 字段配置 EMQX 日志等级

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  emqxTemplate:
    image: emqx/emqx-ee:4.4.8
  config:
    log.level: debug
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

- 查看 EMQX 集群日志信息

```
kubectl logs emqx-ee-0
```