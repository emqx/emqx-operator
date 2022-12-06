# 配置 EMQX 模块

## 任务目标
- 学习如何通过 modules 字段配置 EMQX 集群各种功能模块

## EMQX 模块配置
EMQX CRD 支持通过 `.spec.emqxTemplate.modules` 字段来配置 EMQX 集群各种功能模块 ，modules 字段的描述可以参考：[modules 字段](https://github.com/emqx/emqx-operator/blob/2.0.2/docs/en_US/reference/v1beta3-reference.md#emqxenterprisetemplate)， EMQX 模块文档可以参考: [模块管理](https://docs.emqx.com/zh/enterprise/v4.4/modules/modules.html)

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:kubectl exec -it emqx-ee-0 -- emqx ctl modules list
  name: emqx-ee
spec:
  emqxTemplate:
    image: emqx/emqx-ee:4.4.8
    modules:
      - name: "internal_acl"
        enable: true
        configs:
          acl_rule_file: "/mounted/acl/acl.conf"
      - name: "retainer"
        enable: true
        configs:
          expiry_interval: 0
          max_payload_size: "1MB"
          max_retained_messages: 0
          storage_type: "ram"
```

将上述文件保存为 emqx-modules.yaml 并部署 EMQX 集群

```
kubectl apply -f emqx-modules.yaml
```

待 EMQX 集群就绪之后使用如下命令查看配置的模块是否开启

```
kubectl exec -it emqx-ee-0 -- emqx ctl modules list
```

输出类似于：

```
Module(internal_acl, description = "Internal ACL File", enabled = true)
Module(retainer, description = "Set parameters such as enable status, storage location, and expiration date for MQTT retain messages.", enabled = true)
```