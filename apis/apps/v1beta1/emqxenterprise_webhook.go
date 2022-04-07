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
var emqxenterpriselog = logf.Log.WithName("emqxenterprise-resource")

func (r *EmqxEnterprise) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-apps-emqx-io-v1beta1-emqxenterprise,mutating=true,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxenterprises,verbs=create;update,versions=v1beta1,name=mutating.enterprise.emqx.io,admissionReviewVersions={v1,v1beta1}

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

	if r.Spec.Replicas == nil {
		defaultReplicas := int32(3)
		r.Spec.Replicas = &defaultReplicas
	}

	if r.Spec.ServiceAccountName == "" {
		r.Spec.ServiceAccountName = r.Name
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

	modules := &v1beta2.EmqxEnterpriseModulesList{
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

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-apps-emqx-io-v1beta1-emqxenterprise,mutating=false,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxenterprises,verbs=create;update,versions=v1beta1,name=validator.enterprise.emqx.io,admissionReviewVersions={v1,v1beta1}

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
