module github.com/emqx/emqx-operator

go 1.16

require (
	emperror.dev/errors v0.8.0
	github.com/banzaicloud/k8s-objectmatcher v1.7.0
	github.com/google/uuid v1.2.0 // indirect
	github.com/json-iterator/go v1.1.12
	github.com/onsi/ginkgo/v2 v2.1.4
	github.com/onsi/gomega v1.19.0
	github.com/stretchr/testify v1.7.0
	github.com/tidwall/gjson v1.14.1
	github.com/tidwall/sjson v1.2.4
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.19.1
	k8s.io/api v0.24.2
	k8s.io/apimachinery v0.24.2
	k8s.io/client-go v0.24.2
	sigs.k8s.io/controller-runtime v0.11.1
)
