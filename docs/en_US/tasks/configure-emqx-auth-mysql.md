# Configure emqx_auth_mysql authentication/access control plugin

## Task target

- Learn how to configure emqx_auth_mysql authentication/access control plugin

## Deploy EMQX cluster

```yaml
apiVersion: apps.emqx.io/v1beta3
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  emqxTemplate:
    image: emqx/emqx-ee:4.4.8
```

Save the above file as: emqx-mysql.yaml and deploy the EMQX cluster

```
kubectl apply -f emqx-mysql.yaml
```

- Check whether the EMQX cluster is running normally

```
kubectl get pods  -l  apps.emqx.io/instance=emqx-ee
```

The output is similar to:

```
NAME        READY   STATUS    RESTARTS   AGE
emqx-ee-0   2/2     Running   0          48m
emqx-ee-1   2/2     Running   0          48m
emqx-ee-2   2/2     Running   0          48m
```

- Deploy the emqx_auth_mysql plugin

EMQX Operator provides EmqxPlugin CRD to support users to configure EMQX plugins. For related documents of EmqxPlugin, please refer to: [EmqxPlugin](https://github.com/emqx/emqx-operator/blob/2.0.2/docs/en_US/reference/v1beta3-reference.md#emqxplugin), For documentation related to the EQMX plugin, please refer to: [plugin management](https://docs.emqx.com/en/enterprise/v4.4/advanced/plugins.html), Documentation about emqx_auth_mysql can be referred to: [emqx_auth_mysql](https://github.com/emqx/emqx/tree/main-v4.3/apps/emqx_auth_mysql)


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
**Remarks**: The emqx_auth_mysql plugin relies on the mysql service to provide authentication/access control permissions. Before deploying this plugin, please ensure that the mysql service is running normally. `auth.mysql.server` corresponds to the ip:port of the mysql service

Save the above content as: emqx-auth-mysql.yaml and deploy EmqxPlugin

```
kubectl apply -f emqx-auth-mysql.yaml
```

- Check whether the plugin emqx-auth-mysql is loaded

```
kubectl get emqxplugin  emqx-auth-mysql   -o json | jq  '.status.phase'
```

The output is similar to:

```
loaded
```

**Remarks**: If the output result is not `loaded`, you can execute the above command after a while, because there will be a delay in EMQX loading plugins