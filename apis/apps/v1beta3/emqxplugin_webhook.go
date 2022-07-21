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

package v1beta3

import (
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var emqxpluginlog = logf.Log.WithName("emqxplugin-resource")

var notAllowedPlugin = []string{"emqx_management", "emqx_dashboard"}

func (r *EmqxPlugin) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-apps-emqx-io-v1beta3-emqxplugin,mutating=true,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxplugins,verbs=create;update,versions=v1beta3,name=mutating.emqxplugin.emqx.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &EmqxPlugin{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *EmqxPlugin) Default() {
	emqxpluginlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-apps-emqx-io-v1beta3-emqxplugin,mutating=false,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxplugins,verbs=create;update,versions=v1beta3,name=validator.emqxplugin.emqx.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &EmqxPlugin{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *EmqxPlugin) ValidateCreate() error {
	emqxpluginlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *EmqxPlugin) ValidateUpdate(old runtime.Object) error {
	emqxpluginlog.Info("validate update", "name", r.Name)
	if filerEmqxPlugin(r.Spec.PluginName) {
		err := errors.New("refuse to update this EmqxPlugin")
		emqxpluginlog.Error(err, "validate update failed")
		return err
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *EmqxPlugin) ValidateDelete() error {
	emqxpluginlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func filerEmqxPlugin(s string) bool {
	for _, plugin := range notAllowedPlugin {
		if plugin == s {
			return true
		}
	}
	return false
}
