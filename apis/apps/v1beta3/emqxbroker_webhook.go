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
	"reflect"
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

func (r *EmqxBroker) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-apps-emqx-io-v1beta3-emqxbroker,mutating=true,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxbrokers,verbs=create;update,versions=v1beta3,name=mutating.emqxbroker.v1beta3.emqx.io,admissionReviewVersions={v1,v1beta3}

var _ webhook.Defaulter = &EmqxBroker{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *EmqxBroker) Default() {
	emqxbrokerlog.Info("default", "name", r.Name)

	if r.Labels == nil {
		r.Labels = make(map[string]string)
	}
	r.Labels["apps.emqx.io/managed-by"] = "emqx-operator"
	r.Labels["apps.emqx.io/instance"] = r.GetName()

	if reflect.ValueOf(r.Spec.Replicas).IsZero() {
		defaultReplicas := int32(3)
		r.Spec.Replicas = &defaultReplicas
	}

	if r.Spec.EmqxTemplate.ACL == nil {
		acls := &ACLList{}
		acls.Default()
		r.Spec.EmqxTemplate.ACL = acls.Items
	}

	plugins := &PluginList{
		Items: r.Spec.EmqxTemplate.Plugins,
	}
	plugins.Default()
	if r.Spec.TelegrafTemplate != nil {
		_, index := plugins.Lookup("emqx_prometheus")
		if index == -1 {
			plugins.Items = append(plugins.Items, Plugin{Name: "emqx_prometheus", Enable: true})
		}
	}
	r.Spec.EmqxTemplate.Plugins = plugins.Items

	modules := &EmqxBrokerModuleList{
		Items: r.Spec.EmqxTemplate.Modules,
	}
	modules.Default()
	r.Spec.EmqxTemplate.Modules = modules.Items

	r.Spec.EmqxTemplate.Listener.Default()

	env := &EnvList{
		Items: r.Spec.Env,
	}
	str := strings.Split(r.GetImage(), ":")
	if len(str) > 1 {
		match, _ := regexp.MatchString("^[4-9].[4-9]+.[0-9]+$", str[1])
		if match {
			// 4.4.x uses dns clustering by default
			env.ClusterForDNS(r)
		} else {
			env.ClusterForK8S(r)
		}
	} else {
		env.ClusterForK8S(r)
	}
	if r.Spec.TelegrafTemplate != nil {
		env.Append([]corev1.EnvVar{
			{Name: "EMQX_PROMETHEUS__PUSH__GATEWAY__SERVER", Value: ""},
		})
	}

	r.Spec.Env = env.Items
}

//+kubebuilder:webhook:path=/validate-apps-emqx-io-v1beta3-emqxbroker,mutating=false,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxbrokers,verbs=create;update,versions=v1beta3,name=validator.emqxbroker.v1beta3.emqx.io,admissionReviewVersions={v1,v1beta3}

var _ webhook.Validator = &EmqxBroker{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *EmqxBroker) ValidateCreate() error {
	emqxbrokerlog.Info("validate create", "name", r.Name)

	if err := validateTag(r.Spec.Image); err != nil {
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
	if len(str) > 1 {
		match, _ := regexp.MatchString("^[0-9]+.[0-9]+.[0-9]+$", str[1])
		if !match {
			match, _ := regexp.MatchString("^latest$", str[1])
			if match {
				return nil
			}
			return errors.New("the tag of the image must match '^[0-9]+.[0-9]+.[0-9]+$'")
		}
	}
	return nil
}
