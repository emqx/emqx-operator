# 配置 emqx_auth_mysql 认证/访问控制插件

## 任务目标

- 学习如何通过配置 emqx_auth_mysql 认证/访问控制插件

## 部署 EMQX 集群

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  emqxTemplate:
    image: emqx/emqx-ee:4.4.8
```

将上述文件保存为：emqx-mysql.yaml 并部署 EMQX 集群

```
kubectl apply -f emqx-mysql.yaml
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

## 部署 emqx_auth_mysql 插件

EMQX Operator 提供 EmqxPlugin CRD 来支持用户配置 EMQX 插件，EmqxPlugin 相关的文档可以参考： [EmqxPlugin](https://github.com/emqx/emqx-operator/blob/2.0.2/docs/en_US/reference/v1beta3-reference.md#emqxplugin)，EQMX 插件相关的文档可以参考：[插件管理](https://docs.emqx.com/zh/enterprise/v4.4/advanced/plugins.html)，关于 emqx_auth_mysql 的文档可以参考： [emqx_auth_mysql](https://github.com/emqx/emqx/tree/main-v4.3/apps/emqx_auth_mysql)

```yaml 
apiVersion: apps.emqx.io/v1beta3
kind: EmqxPlugin
metadata:
  name: emqx-auth-mysql
spec:
  pluginName: emqx_auth_mysql
  config:
     "auth.mysql.server": "10.244.2.149:3306"
     "auth.mysql.username": "root"
     "auth.mysql.password_hash": "plain"
     "auth.mysql.password": "root"
     "auth.mysql.database": "mqtt"
     "auth.mysql.auth_query": "select password from mqtt_user where username = '%u' limit 1"
     "auth.mysql.super_query": "select is_superuser from mqtt_user where username = '%u' limit 1"
     "auth.mysql.acl_query": "select allow, ipaddr, username, clientid, access, topic from mqtt_acl where ipaddr = '%a' or username = '%u' or username = '$all' or clientid = '%c'"
```

**备注**：emqx_auth_mysql 插件依赖 mysql 服务提供认证/访问控制权限，在部署此插件之前请确保 mysql 服务运行正常，`auth.mysql.server` 对应 mysql 服务的 ip:port  

将上述内容保存为：emqx-auth-mysql.yaml 并部署 EmqxPlugin 

```
kubectl apply -f emqx-auth-mysql.yaml
```

- 查看插件 emqx-auth-mysql 是否被加载

```
kubectl get emqxplugin  emqx-auth-mysql   -o json | jq  '.status.phase'
```

输出类似于：

```
loaded
```

**备注**： 若输出结果不是`loaded` ，可以稍等会再执行上述命令，因为 EMQX 加载插件会有延迟