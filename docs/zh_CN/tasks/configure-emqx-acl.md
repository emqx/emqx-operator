# 配置 EMQX 发布订阅 ACL


## 任务目标
- 学习如何通过 acl 字段配置 EMQX 集群发布订阅 ACL

## EMQX 发布订阅 ACL 配置

EMQX CRD 支持通过 `.spec.emqxTemplate.acl` 字段来配置 EMQX 发布订阅 ACL ，acl 字段的具体描述可以参考: [acl 字段](https://github.com/emqx/emqx-operator/blob/2.0.2/docs/en_US/reference/v1beta3-reference.md#emqxenterprisetemplate) ， EMQX 发布订阅 ACL的文档可以参考：[发布订阅 ACL](https://docs.emqx.com/zh/enterprise/v4.4/advanced/acl.html)

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  emqxTemplate:
    image: emqx/emqx-ee:4.4.8
    acl: 
    # 拒绝 "所有用户" 订阅 "$SYS/#" "#" 主题
    - "{deny, all, subscribe, ["$SYS/#", {eq, "#"}]}."
```

将上述内容保存为 emqx-acl.yaml 并部署 EMQX 集群

```
kubectl apply -f emqx-acl.yaml
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