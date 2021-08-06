# `Operator Demo`文档 

## 开发

### 预置准备
* [git](https://git-scm.com/downloads)
* [go](https://golang.org/dl/) version v1.16+
* [docker](https://docs.docker.com/get-docker/)
* `kubernetes` 集群版本 v1.20+

*[`kubebuilder book`](https://book.kubebuilder.io/)*

## 快速指导

* *在运行 `operator` 之前，首先得将 `CRD` 注册到 `kubernetes apiserver`中*
```
$ kubectl create -f config/samples/crd_custom.emqx.io_brokers.yaml
```
* 有两种方式运行 `operator`:
  * 在 `kubernetes` 集群中以`Deployment`部署`operator`
  * 在集群外作为`Go`工程项目运行
1. 部署 `operator deployment` 运行
编译及制作镜像，并推送到镜像仓库
```
$ IMG=xxx make docker-build
$ IMG=xxx make docker-push
```
* 创建 `operator manager controller` 相关组件服务
```
$ kubectl create -f config/samples/operator/operator_namespace.yaml
$ kubectl create -f config/samples/operator/operator_service_account.yaml
$ kubectl create -f config/samples/operator/operator_role.yaml
$ kubectl create -f config/samples/operator/operator_role_binding.yaml
$ kubectl create -f config/samples/operator_deployment.yaml
```
* 创建 `cr` 关联的 `RBAC` 及 `cr`
```
$ kubectl create -f config/samples/broker/broker_serviceaccount.yaml
$ kubectl create -f config/samples/broker/broker_role.yaml
$ kubectl create -f config/samples/broker/broker_role_binding.yaml
$ kubectl create -f config/samples/broker/custom_v1alpha1_broker.yaml
```
* 确认 `broker pod` 正常运行
```
$ kubectl get pods               
NAME              READY   STATUS    RESTARTS   AGE
broker-sample-0   1/1     Running   0          22m
$ kubectl logs -f broker-sample-0
listener.ssl.external.acceptors = "32"
listener.ssl.external.max_connections = "102400"
listener.tcp.external.acceptors = "64"
listener.tcp.external.max_connections = "1024000"
listener.ws.external.acceptors = "16"
listener.ws.external.max_connections = "102400"
listener.wss.external.acceptors = "16"
listener.wss.external.max_connections = "102400"
log.to = "console"
node.max_ets_tables = "2097152"
node.max_ports = "1048576"
node.name = "broker-sample-0@172.19.98.33"
node.process_limit = "2097152"
rpc.port_discovery = "manual"
Exec: /opt/emqx/erts-11.1.8/bin/erlexec -noshell -noinput +Bd -boot /opt/emqx/releases/4.3.1/start -mode embedded -boot_var ERTS_LIB_DIR /opt/emqx/erts-11.1.8/../lib -mnesia dir "/opt/emqx/data/mnesia/broker-sample-0@172.19.98.33" -config /opt/emqx/data/configs/app.2021.08.06.02.43.32.config -args_file /opt/emqx/data/configs/vm.2021.08.06.02.43.32.args -vm_args /opt/emqx/data/configs/vm.2021.08.06.02.43.32.args -start_epmd false -epmd_module ekka_epmd -proto_dist ekka -- foreground
Root: /opt/emqx
Starting emqx on node broker-sample-0@172.19.98.33
Start http:management listener on 8081 successfully.
Start http:dashboard listener on 18083 successfully.
Start mqtt:tcp:internal listener on 127.0.0.1:11883 successfully.
Start mqtt:tcp:external listener on 0.0.0.0:1883 successfully.
Start mqtt:ws:external listener on 0.0.0.0:8083 successfully.
Start mqtt:ssl:external listener on 0.0.0.0:8883 successfully.
Start mqtt:wss:external listener on 0.0.0.0:8084 successfully.
EMQ X Enterprise 4.3.1 is running now!
```
2. 集群外运行本地工程项目
* 将 `kubernetes` 集群的配置文件 `.kube/config` 复制到本地 `$HOME/.kube/config`
* 运行 `main.go` 文件
* 创建 `cr` 关联的 `RBAC` 及 `cr`
```
$ kubectl create -f config/samples/broker/broker_serviceaccount.yaml
$ kubectl create -f config/samples/broker/broker_role.yaml
$ kubectl create -f config/samples/broker/broker_role_binding.yaml
$ kubectl create -f config/samples/broker/custom_v1alpha1_broker.yaml
```
* 确认 `broker pod` 成功运行


# Q&A
1. 在本地执行 `make docker build` 时出现如下报错：
```
Unexpected error: msg: "failed to start the controlplane. retried 5 times: fork/exec /usr/local/kubebuilder/bin/etcd: no such file or directory"
```
[解决方法参考](https://github.com/kubernetes-sigs/kubebuilder/issues/1599)
执行如下脚本
```
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/')
curl -fsL "https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-1.16.4-${OS}-${ARCH}.tar.gz" -o kubebuilder-tools
tar -zvxf kubebuilder-tools
sudo mv kubebuilder/ /usr/local/kubebuilder
```
  

