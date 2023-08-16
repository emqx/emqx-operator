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

package v2beta1

import (
	"reflect"

	emperror "emperror.dev/errors"

	// "github.com/gurkankaymak/hocon"

	hocon "github.com/rory-z/go-hocon"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var emqxlog = logf.Log.WithName("emqx-resource")

func (r *EMQX) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-apps-emqx-io-v2beta1-emqx,mutating=true,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxes,verbs=create;update,versions=v2beta1,name=mutating.apps.emqx.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &EMQX{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *EMQX) Default() {
	emqxlog.Info("default", "name", r.Name)
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-apps-emqx-io-v2beta1-emqx,mutating=false,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxes,verbs=create;update,versions=v2beta1,name=validator.apps.emqx.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &EMQX{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *EMQX) ValidateCreate() error {
	emqxlog.Info("validate create", "name", r.Name)

	if *r.Spec.CoreTemplate.Spec.Replicas <= 1 {
		err := emperror.New("the number of EMQX core nodes must be greater than 1")
		emqxlog.Error(err, "validate create failed")
		return err
	}

	if *r.Spec.CoreTemplate.Spec.Replicas > 4 {
		err := emperror.New("the number of EMQX core nodes must be less than or equal to 4")
		emqxlog.Error(err, "validate create failed")
		return err
	}

	if _, err := hocon.ParseString(r.Spec.Config.Data); err != nil {
		err = emperror.Wrap(err, "failed to parse config")
		emqxlog.Error(err, "validate create failed")
		return err
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *EMQX) ValidateUpdate(old runtime.Object) error {
	emqxlog.Info("validate update", "name", r.Name)

	if *r.Spec.CoreTemplate.Spec.Replicas <= 1 {
		err := emperror.New("the number of EMQX core nodes must be greater than 1")
		emqxlog.Error(err, "validate update failed")
		return err
	}

	if *r.Spec.CoreTemplate.Spec.Replicas > 4 {
		err := emperror.New("the number of EMQX core nodes must be less than or equal to 4")
		emqxlog.Error(err, "validate create failed")
		return err
	}

	oldEMQX := old.(*EMQX)
	if !reflect.DeepEqual(oldEMQX.Spec.BootstrapAPIKeys, r.Spec.BootstrapAPIKeys) {
		err := emperror.Errorf("bootstrap APIKey cannot be updated")
		emqxlog.Error(err, "validate update failed")
		return err
	}

	_, err := hocon.ParseString(r.Spec.Config.Data)
	if err != nil {
		err = emperror.Wrap(err, "failed to parse config")
		emqxlog.Error(err, "validate update failed")
		return err
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *EMQX) ValidateDelete() error {
	emqxlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
