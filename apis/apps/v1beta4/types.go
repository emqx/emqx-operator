package v1beta4

import (
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:object:generate=false
type Emqx interface {
	client.Object

	GetSpec() EmqxSpec
	GetStatus() EmqxStatus

	Default()
	ValidateCreate() error
	ValidateUpdate(runtime.Object) error
	ValidateDelete() error
}
