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
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var emqxbrokerlog = logf.Log.WithName("emqxbroker-resource")

const (
	DefaultUsername       = "admin"
	DefaultPassword       = "public"
	DefaultManagementPort = "8081"
)

func (r *EmqxBroker) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-apps-emqx-io-v1beta3-emqxbroker,mutating=true,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxbrokers,verbs=create;update,versions=v1beta3,name=mutating.broker.emqx.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &EmqxBroker{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *EmqxBroker) Default() {
	emqxbrokerlog.Info("default", "name", r.Name)

	if r.Labels == nil {
		r.Labels = make(map[string]string)
	}
	r.Labels["apps.emqx.io/managed-by"] = "emqx-operator"
	r.Labels["apps.emqx.io/instance"] = r.GetName()

	modules := &EmqxBrokerModuleList{
		Items: r.Spec.EmqxTemplate.Modules,
	}
	modules.Default()
	r.Spec.EmqxTemplate.Modules = modules.Items

	if r.Spec.EmqxTemplate.EmqxConfig == nil {
		r.Spec.EmqxTemplate.EmqxConfig = make(EmqxConfig)
	}
	r.Spec.EmqxTemplate.EmqxConfig.Default(r)
	r.Spec.EmqxTemplate.ServiceTemplate.Default(r)

	if r.Spec.EmqxTemplate.SecurityContext == nil {
		emqxUserGroup := int64(1000)
		fsGroupChangeAlways := corev1.FSGroupChangeAlways

		r.Spec.EmqxTemplate.SecurityContext = &corev1.PodSecurityContext{
			RunAsUser:           &emqxUserGroup,
			RunAsGroup:          &emqxUserGroup,
			FSGroup:             &emqxUserGroup,
			FSGroupChangePolicy: &fsGroupChangeAlways,
			SupplementalGroups:  []int64{emqxUserGroup},
		}
	}

	if len(r.Spec.EmqxTemplate.Username) == 0 {
		r.Spec.EmqxTemplate.Username = DefaultUsername
	}
	if len(r.Spec.EmqxTemplate.Password) == 0 {
		r.Spec.EmqxTemplate.Password = DefaultPassword
	}
}

//+kubebuilder:webhook:path=/validate-apps-emqx-io-v1beta3-emqxbroker,mutating=false,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxbrokers,verbs=create;update,versions=v1beta3,name=validator.broker.emqx.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &EmqxBroker{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *EmqxBroker) ValidateCreate() error {
	emqxbrokerlog.Info("validate create", "name", r.Name)

	if err := validateImageTag(r); err != nil {
		emqxbrokerlog.Error(err, "validate create failed")
		return err
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *EmqxBroker) ValidateUpdate(old runtime.Object) error {
	emqxbrokerlog.Info("validate update", "name", r.Name)

	if err := validateImageTag(r); err != nil {
		emqxbrokerlog.Error(err, "validate update failed")
		return err
	}
	oldEmqx := old.(*EmqxBroker)
	if err := validateUsernameAndPassword(r, oldEmqx); err != nil {
		emqxbrokerlog.Error(err, "validate update failed")
		return err
	}

	if err := validatePersistent(r, oldEmqx); err != nil {
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

func validateImageTag(emqx Emqx) error {
	image := emqx.GetImage()
	str := strings.Split(image, ":")
	l := len(str)
	if l > 1 {
		match, _ := regexp.MatchString("^[0-4]+.[0-3]+.*$", str[l-1])
		if match {
			return errors.New("EMQX Operator only support EMQX 4.4 and higher version")
		}
	}
	return nil
}
