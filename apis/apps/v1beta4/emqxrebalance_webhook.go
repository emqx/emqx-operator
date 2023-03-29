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
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var emqxrebalancelog = logf.Log.WithName("emqxrebalance-resource")

func (r *EmqxRebalance) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-apps-emqx-io-v1beta4-emqxrebalance,mutating=false,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxrebalances,verbs=create;update,versions=v1beta4,name=vemqxrebalance.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &EmqxRebalance{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *EmqxRebalance) ValidateCreate() error {
	emqxrebalancelog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *EmqxRebalance) ValidateUpdate(old runtime.Object) error {
	emqxrebalancelog.Info("validate update", "name", r.Name)
	// TODO(user): fill in your validation logic upon object update.
	return errors.New("prohibit to update emqxrebalance")
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *EmqxRebalance) ValidateDelete() error {
	emqxrebalancelog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
