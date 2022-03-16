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

package v1beta2

import (
	"regexp"
	"strings"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	corev1 "k8s.io/api/core/v1"
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

//+kubebuilder:webhook:path=/mutate-apps-emqx-io-v1beta2-emqxenterprise,mutating=true,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxenterprises,verbs=create;update,versions=v1beta2,name=mutating.emqxenterprise.v1beta2.emqx.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &EmqxEnterprise{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *EmqxEnterprise) Default() {
	emqxenterpriselog.Info("default", "name", r.Name)

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

	if r.Spec.EmqxTemplate.ACL == nil {
		acls := &v1beta3.ACLList{}
		acls.Default()
		r.Spec.EmqxTemplate.ACL = acls.Items
	}

	plugins := &v1beta3.PluginList{
		Items: r.Spec.EmqxTemplate.Plugins,
	}
	plugins.Default()
	r.Spec.EmqxTemplate.Plugins = plugins.Items

	modules := &v1beta3.EmqxEnterpriseModuleList{
		Items: r.Spec.EmqxTemplate.Modules,
	}
	modules.Default()
	r.Spec.EmqxTemplate.Modules = modules.Items

	listener := &r.Spec.EmqxTemplate.Listener
	listener.Default()
	r.Spec.EmqxTemplate.Listener = *listener

	env := &v1beta3.EnvList{
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

	e, _ := env.Lookup("EMQX_CLUSTER__DISCOVERY")
	if e != nil && e.Value == "dns" {
		r.Spec.ServiceAccountName = ""
	}
	r.Spec.Env = env.Items

	if r.Spec.SecurityContext == nil {
		emqxUserGroup := int64(1000)
		runAsNonRoot := true
		fsGroupChangeAlways := corev1.FSGroupChangeAlways

		r.Spec.SecurityContext = &corev1.PodSecurityContext{
			FSGroup:             &emqxUserGroup,
			FSGroupChangePolicy: &fsGroupChangeAlways,
			RunAsNonRoot:        &runAsNonRoot,
			RunAsUser:           &emqxUserGroup,
			SupplementalGroups:  []int64{emqxUserGroup},
		}
	}
}

//+kubebuilder:webhook:path=/validate-apps-emqx-io-v1beta2-emqxenterprise,mutating=false,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxenterprises,verbs=create;update,versions=v1beta2,name=validator.emqxenterprise.v1beta2.emqx.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &EmqxEnterprise{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *EmqxEnterprise) ValidateCreate() error {
	emqxenterpriselog.Info("validate create", "name", r.Name)

	if err := validateTag(r.Spec.Image); err != nil {
		return err
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *EmqxEnterprise) ValidateUpdate(old runtime.Object) error {
	emqxenterpriselog.Info("validate update", "name", r.Name)

	if err := validateTag(r.Spec.Image); err != nil {
		return err
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *EmqxEnterprise) ValidateDelete() error {
	emqxenterpriselog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
