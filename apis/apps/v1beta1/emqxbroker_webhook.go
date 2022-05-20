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
	"regexp"
	"strings"

	"github.com/emqx/emqx-operator/apis/apps/v1beta2"
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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-apps-emqx-io-v1beta1-emqxbroker,mutating=true,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxbrokers,verbs=create;update,versions=v1beta1,name=mutating.broker.emqx.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &EmqxBroker{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *EmqxBroker) Default() {
	emqxbrokerlog.Info("default", "name", r.Name)

	labels := make(map[string]string)
	for k, v := range r.Labels {
		labels[k] = v
	}
	for k, v := range r.Spec.Labels {
		labels[k] = v
	}
	labels["apps.emqx.io/managed-by"] = "emqx-operator"
	labels["apps.emqx.io/instance"] = r.GetName()

	r.Labels = labels
	r.Spec.Labels = labels

	if r.Spec.ServiceAccountName == "" {
		r.Spec.ServiceAccountName = r.Name
	}

	if r.Spec.ACL == nil {
		acls := &v1beta2.ACLs{}
		acls.Default()
		r.Spec.ACL = acls.Items
	}

	plugins := &v1beta2.Plugins{
		Items: r.Spec.Plugins,
	}
	plugins.Default()
	if r.Spec.TelegrafTemplate != nil {
		_, index := plugins.Lookup("emqx_prometheus")
		if index == -1 {
			plugins.Items = append(plugins.Items, v1beta2.Plugin{Name: "emqx_prometheus", Enable: true})
		}
	}
	r.Spec.Plugins = plugins.Items

	modules := &v1beta2.EmqxBrokerModulesList{
		Items: r.Spec.Modules,
	}
	modules.Default()
	r.Spec.Modules = modules.Items

	listener := &r.Spec.Listener
	listener.Default()
	r.Spec.Listener = *listener

	env := &v1beta2.Environments{
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

	e, _ := env.Lookup("EMQX_CLUSTER__DISCOVERY")
	if e != nil && e.Value == "dns" {
		r.Spec.ServiceAccountName = ""
	}
	r.Spec.Env = env.Items

	if r.Spec.SecurityContext == nil {
		emqxUserGroup := int64(1000)
		fsGroupChangeAlways := corev1.FSGroupChangeAlways

		r.Spec.SecurityContext = &corev1.PodSecurityContext{
			RunAsUser:           &emqxUserGroup,
			RunAsGroup:          &emqxUserGroup,
			FSGroup:             &emqxUserGroup,
			FSGroupChangePolicy: &fsGroupChangeAlways,
			SupplementalGroups:  []int64{emqxUserGroup},
		}
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
	l := len(str)
	if l > 1 {
		match, _ := regexp.MatchString("^[0-9]+.[0-9]+.[0-9]+$", str[l-1])
		if !match {
			match, _ := regexp.MatchString("^latest$", str[l-1])
			if match {
				return nil
			}
			return errors.New("the tag of the image must match '^[0-9]+.[0-9]+.[0-9]+$'")
		}
	}
	return nil
}
