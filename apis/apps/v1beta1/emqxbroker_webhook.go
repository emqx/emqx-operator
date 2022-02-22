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

package v1beta1

import (
	"errors"
	"reflect"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var emqxbrokerlog = logf.Log.WithName("emqxbroker-resource")

func (r *EmqxBroker) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-apps-emqx-io-v1beta1-emqxbroker,mutating=true,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxbrokers,verbs=create;update,versions=v1beta1,name=mutating.broker.emqx.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &EmqxBroker{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *EmqxBroker) Default() {
	emqxbrokerlog.Info("default", "name", r.Name)

	r.Labels = generateLabels(r)
	r.Spec.Labels = generateLabels(r)

	if reflect.ValueOf(r.Spec.Replicas).IsZero() {
		defaultReplicas := int32(3)
		r.Spec.Replicas = &defaultReplicas
	}

	if r.Spec.ServiceAccountName == "" {
		r.Spec.ServiceAccountName = r.Name
	}

	if r.Spec.ACL == nil {
		r.Spec.ACL = defaultACL()
	}

	r.Spec.Env = generateEnv(r)
	r.Spec.Plugins = generatePlugins(r.Spec.Plugins)
	r.Spec.Modules = generateEmqxBrokerModules(r.Spec.Modules)
	r.Spec.Listener = generateListener(r.Spec.Listener)
	if r.Spec.TelegrafTemplate != nil {
		r.Spec.TelegrafTemplate = generateTelegrafTemplate(r.Spec.TelegrafTemplate)
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-apps-emqx-io-v1beta1-emqxbroker,mutating=false,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxbrokers,verbs=create;update,versions=v1beta1,name=validator.broker.emqx.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &EmqxBroker{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *EmqxBroker) ValidateCreate() error {
	emqxbrokerlog.Info("validate create", "name", r.Name)

	if err := validateTag(r.Spec.Image); err != nil {
		emqxbrokerlog.Error(err, "validate create failed")
		return err
	}

	if err := validateTelegrafTemplate(r.Spec.TelegrafTemplate); err != nil {
		emqxbrokerlog.Error(err, "validate create failed")
		return err
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *EmqxBroker) ValidateUpdate(old runtime.Object) error {
	emqxbrokerlog.Info("validate update", "name", r.Name)

	if err := validateTag(r.Spec.Image); err != nil {
		emqxbrokerlog.Error(err, "validate update failed")
		return err
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *EmqxBroker) ValidateDelete() error {
	emqxbrokerlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func validateTag(image string) error {
	str := strings.Split(image, ":")
	match, _ := regexp.MatchString("^[0-9]+.[0-9]+.[0-9]+$", str[1])
	if !match {
		return errors.New("The tag of the image must match '^[0-9]+.[0-9]+.[0-9]+$'")
	}
	return nil
}

func validateTelegrafTemplate(telegrafTemplate *TelegrafTemplate) error {
	if telegrafTemplate != nil && telegrafTemplate.Image == "" {
		return errors.New("The image of telegraf must be completed")
	}
	if telegrafTemplate != nil && telegrafTemplate.Conf == nil {
		return errors.New("The conf of the telegraf must be completed")
	}
	return nil
}
