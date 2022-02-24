# FAQ

## `make docker build` 错误:

```bash
Unexpected error: msg: "failed to start the controlplane. retried 5 times: fork/exec /usr/local/kubebuilder/bin/etcd: no such file or directory"
```

[References](https://github.com/kubernetes-sigs/kubebuilder/issues/1599)

```bash
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/')
curl -fsL "https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-1.16.4-${OS}-${ARCH}.tar.gz" -o kubebuilder-tools
tar -zvxf kubebuilder-tools
sudo mv kubebuilder/ /usr/local/kubebuilder
```