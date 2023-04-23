/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta4

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var emqxenterpriselog = logf.Log.WithName("emqxenterprise-resource")

func (r *EmqxEnterprise) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-apps-emqx-io-v1beta4-emqxenterprise,mutating=true,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxenterprises,verbs=create;update,versions=v1beta4,name=mutating.enterprise.emqx.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &EmqxEnterprise{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *EmqxEnterprise) Default() {
	emqxenterpriselog.Info("default", "name", r.Name)

	defaultLabelsAndAnnotations(r)
	defaultEmqxImage(r)
	defaultEmqxACL(r)
	defaultEmqxConfig(r)
	defaultServiceTemplate(r)
	defaultPersistent(r)
	defaultTerminationMessage(r)
}

//+kubebuilder:webhook:path=/validate-apps-emqx-io-v1beta4-emqxenterprise,mutating=false,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxenterprises,verbs=create;update,versions=v1beta4,name=validator.enterprise.emqx.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &EmqxEnterprise{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *EmqxEnterprise) ValidateCreate() error {
	emqxenterpriselog.Info("validate create", "name", r.Name)

	if err := validateImageVersion(r, nil); err != nil {
		emqxbrokerlog.Error(err, "validate create failed")
		return err
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *EmqxEnterprise) ValidateUpdate(old runtime.Object) error {
	emqxenterpriselog.Info("validate update", "name", r.Name)

	callbacks := []func(new, old Emqx) error{
		validateBootstrapAPIKey,
		validateImageVersion,
		validatePersistent,
		validateEmqxConfig,
	}
	for _, cb := range callbacks {
		if err := cb(r, old.(*EmqxEnterprise)); err != nil {
			emqxbrokerlog.Error(err, "validate create failed")
			return err
		}
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *EmqxEnterprise) ValidateDelete() error {
	emqxenterpriselog.Info("validate delete", "name", r.Name)

	return nil
}
