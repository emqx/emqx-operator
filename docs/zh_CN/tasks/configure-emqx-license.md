# 配置 EMQX 企业版 License

## 任务目标
 
- 如何使用 secretName 字段配置 EMQX 企业版 License。
- 如何更新 EMQX 企业版 License。

## 使用 secretName 字段配置 License

- 基于 License 文件创建 Secret

Secret 是一种包含少量敏感信息例如密码、令牌或密钥的对象。关于 Secret 更加详尽的文档可以参考：[Secret](https://kubernetes.io/zh-cn/docs/concepts/configuration/secret/)。EMQX Operator 支持使用 Secret 挂载 License 信息，因此在创建 EMQX 集群之前我们需要基于 License 创建好 Secret。

EMQX 企业版 License 可以在 EMQ 官网免费申请：[申请 EMQX 企业版 License](https://www.emqx.com/zh/apply-licenses/emqx)。

```
kubectl create secret generic test --from-file=emqx.lic=/path/to/license/file
```

**说明**：`/path/to/license/file` 表示 EMQX 企业版 License 文件路径，可以是绝对路径，也可以是相对路径。更多使用 kubectl 创建 Secret 的细节可以参考文档：[使用 kubectl 创建 secret](https://kubernetes.io/zh-cn/docs/tasks/configmap-secret/managing-secret-using-kubectl/)。

输出类似于：

```
secret/test created
```

- 配置 EMQX 集群

EMQX 企业版在 EMQX Operator 里面对应的 CRD 为 EmqxEnterprise，EmqxEnterprise 支持通过 `.spec.emqxTemplate.license.secretName` 字段来配置 EMQX 企业版 License，secretName 字段的具体描述可以参考：[secretName](https://github.com/emqx/emqx-operator/blob/2.0.2/docs/en_US/reference/v1beta3-reference.md#license)。

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  emqxTemplate:
    image: emqx/emqx-ee:4.4.8
    license:
      secretName: test
```

**说明**：`secretName` 表示上一步中创建的 Secret 名称。

将上述内容保存为：emqx-license.yaml，执行如下命令部署 EMQX 企业版集群。

```
kubectl apply -f emqx-license.yaml
```

输出类似于：

```
emqxenterprise.apps.emqx.io/emqx-ee created
```

- 检查 EMQX 企业版集群是否就绪

```
kubectl get emqxenterprise emqx-ee -o json | jq ".status.emqxNodes"
```

输出类似于：

```
[
  {
    "node": "emqx-ee@emqx-ee-1.emqx-ee-headless.default.svc.cluster.local",
    "node_status": "Running",
    "otp_release": "24.1.5/12.1.5",
    "version": "4.4.8"
  },
  {
    "node": "emqx-ee@emqx-ee-0.emqx-ee-headless.default.svc.cluster.local",
    "node_status": "Running",
    "otp_release": "24.1.5/12.1.5",
    "version": "4.4.8"
  },
  {
    "node": "emqx-ee@emqx-ee-2.emqx-ee-headless.default.svc.cluster.local",
    "node_status": "Running",
    "otp_release": "24.1.5/12.1.5",
    "version": "4.4.8"
  }
]
```

**说明**：`node` 表示 EMQX 节点在集群的唯一标识。`node_status` 表示 EMQX 节点的状态。`otp_release` 表示 EMQX 使用的 Erlang 的版本。`version` 表示 EMQX 版本。EMQX Operator 默认会拉起三个节点的 EMQX 集群，所以当集群运行正常时，可以看到三个运行的节点信息。如果你配置了 `.spec.replicas` 字段，当集群运行正常时，输出结果中显示的运行节点数量应和 replicas 的值相等。

- 检查 EMQX 企业版 License 信息 

```
kubectl exec -it emqx-ee-0 -c emqx -- emqx_ctl license info 
```

输出类似于：

```
customer                 : EMQ X Evaluation
email                    : contact@emqx.io
max_connections          : 10
original_max_connections : 10
issued_at                : 2020-06-20 03:02:52
expiry_at                : 2049-01-01 03:02:52
vendor                   : EMQ Technologies Co., Ltd.
version                  : 4.4.8
type                     : official
customer_type            : 10
expiry                   : false
```

**说明**：从输出结果可以看到我们申请的 License 的基本信息，包括申请人的信息和 License 支持最大连接数以及 License 过期时间等。

## 更新 EMQX 企业版 License  

- 更新 EMQX 企业版 License Secret

```
kubectl create secret generic test --from-file=emqx.lic=/path/to/license/file --dry-run -o yaml | kubectl apply -f -
```

输出类似于：

```
secret/test configured
```

- 查看 EMQX 集群 License 是否被更新

```
kubectl exec -it emqx-ee-0 -c emqx -- emqx_ctl license info 
```

输出类似于：

```
customer                 : cloudnative
email                    : cloudnative@emqx.io
max_connections          : 100000
original_max_connections : 100000
issued_at                : 2022-11-21 02:49:35
expiry_at                : 2022-12-01 02:49:35
vendor                   : EMQ Technologies Co., Ltd.
version                  : 4.4.8
type                     : official
customer_type            : 2
expiry                   : false
```

**说明**：若证书信息没有更新，可以等待一会，License 的更新会有些时延。从上面输出的结果可以看出，License 的内容已经更新，则说明 EMQX 企业版 License 更新成功。 