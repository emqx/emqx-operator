module github.com/emqx/emqx-operator

go 1.16

require (
	github.com/banzaicloud/k8s-objectmatcher v1.7.0
	github.com/go-logr/logr v1.2.0
	github.com/google/uuid v1.2.0 // indirect
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.17.0
	github.com/stretchr/testify v1.7.0
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.19.1
	k8s.io/api v0.23.4
	k8s.io/apimachinery v0.23.4
	k8s.io/client-go v0.23.4
	sigs.k8s.io/controller-runtime v0.11.1
)
