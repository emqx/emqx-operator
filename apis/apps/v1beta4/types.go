package v1beta4

import (
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:object:generate=false
type Emqx interface {
	client.Object

	GetSpec() EmqxSpec
	GetStatus() EmqxStatus

	Default()
	ValidateCreate() (admission.Warnings, error)
	ValidateUpdate(runtime.Object) (admission.Warnings, error)
	ValidateDelete() (admission.Warnings, error)
}
