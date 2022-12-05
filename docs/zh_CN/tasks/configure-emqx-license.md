# 配置 EMQX License

## 任务目标

- 学习如何通过 data 字段配置 EMQX License 
- 学习如何通过 secretName 字段配置 EMQX License 
- 学习如何更新 EMQX License 

## EMQX License 配置

EMQX CRD 支持通过 `.spec.emqxTemplate.license` 字段来配置 EMQX 集群 License，license 字段的具体描述可以参考：[License 参考文档](https://github.com/emqx/emqx-operator/blob/1.2.8/docs/en_US/reference/v1beta3-reference.md#license)，EMQX License 可以在 EMQ 官网申请：[免费申请 EMQX 企业版 License](https://www.emqx.com/zh/apply-licenses/emqx)

### 通过 data 字段配置 License

- 将 License 内容的 Base64 编码结果填充到 data 字段

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  emqxTemplate:
    image: emqx/emqx-ee:4.4.8
    license:
      data:
```

将上述内容保存为：emqx-license.yaml 并部署 EMQX 集群

```
kubectl apply -f emqx-license.yaml
```

- 待 EMQX 集群就绪之后，查看 EMQX 集群 License 信息

```
kubectl exec -it emqx-ee-0 -- emqx_ctl license info
```

输出类似于：

```
Defaulted container "emqx" out of: emqx, reloader
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

- 将新的 License 内容更新到 data 字段并更新 EMQX 集群，命令查看 EMQX 集群 License 是否被更新

```
kubectl exec -it emqx-ee-0 -- emqx_ctl license info
```

输出类似于：

``` 
Defaulted container "emqx" out of: emqx, reloader
customer                 : raoxiaoli
email                    : xiaoli.rao@emqx.io
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

**备注**：若证书信息没有更新，可以等待一会，License 的更新依赖 reloader 容器，会有些延迟。

### 通过 secretName 字段配置 License

- 基于 License 文件 创建 secret

```
kubectl create secret generic test --from-file=emqx.lic=license.lic
```

- 将 secretName 字段设置为上一步中创建的 secret 名称：test

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

将上述内容保存为：emqx-license.yaml 并部署 EMQX 集群

```
kubectl apply -f emqx-license.yaml
```

- 查看 EMQX 集群 License 信息

```
kubectl exec -it emqx-ee-0 -- emqx_ctl license info 
```

输出类似于：

```
Defaulted container "emqx" out of: emqx, reloader
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

- 用新的 License 更新 secret，new.license.lic 是新的 License 文件名

```
kubectl create secret generic test --from-file=emqx.lic=new.license.lic --dry-run -o yaml | kubectl apply  -f - 
```

- 查看 EMQX 集群 License 是否被更新

```
kubectl exec -it emqx-ee-0 -- emqx_ctl license info 
```

输出类似于：

```
efaulted container "emqx" out of: emqx, reloader
customer                 : raoxiaoli
email                    : xiaoli.rao@emqx.io
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

**备注**：若证书信息没有更新，可以等待一会，License 的更新依赖 reloader 容器，会有些延迟。