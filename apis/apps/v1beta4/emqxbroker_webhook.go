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
	"fmt"
	"reflect"

	semver "github.com/Masterminds/semver/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
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

//+kubebuilder:webhook:path=/mutate-apps-emqx-io-v1beta4-emqxbroker,mutating=true,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxbrokers,verbs=create;update,versions=v1beta4,name=mutating.broker.emqx.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &EmqxBroker{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *EmqxBroker) Default() {
	emqxbrokerlog.Info("default", "name", r.Name)

	defaultLabels(r)
	defaultEmqxImage(r)
	defaultEmqxACL(r)
	defaultEmqxConfig(r)
	defaultServiceTemplate(r)
}

//+kubebuilder:webhook:path=/validate-apps-emqx-io-v1beta4-emqxbroker,mutating=false,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxbrokers,verbs=create;update,versions=v1beta4,name=validator.broker.emqx.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &EmqxBroker{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *EmqxBroker) ValidateCreate() error {
	emqxbrokerlog.Info("validate create", "name", r.Name)

	if err := validateImageVersion(r); err != nil {
		emqxbrokerlog.Error(err, "validate create failed")
		return err
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *EmqxBroker) ValidateUpdate(old runtime.Object) error {
	emqxbrokerlog.Info("validate update", "name", r.Name)

	if err := validateImageVersion(r); err != nil {
		emqxbrokerlog.Error(err, "validate create failed")
		return err
	}

	oldEmqx := old.(*EmqxBroker)
	if err := validateVolumeClaimTemplates(r, oldEmqx); err != nil {
		emqxbrokerlog.Error(err, "validate update failed")
		return err
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *EmqxBroker) ValidateDelete() error {
	emqxbrokerlog.Info("validate delete", "name", r.Name)

	return nil
}

func defaultLabels(r Emqx) {
	labels := r.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	labels["apps.emqx.io/managed-by"] = "emqx-operator"
	labels["apps.emqx.io/instance"] = r.GetName()
	r.SetLabels(labels)

	template := r.GetSpec().GetTemplate()
	if template.Labels == nil {
		template.Labels = make(map[string]string)
	}
	for k, v := range labels {
		template.Labels[k] = v
	}
	r.GetSpec().SetTemplate(template)
}

func defaultEmqxImage(r Emqx) {
	template := r.GetSpec().GetTemplate()
	if template.Spec.EmqxContainer.Image.Repository == "" {
		if _, ok := r.(*EmqxBroker); ok {
			template.Spec.EmqxContainer.Image.Repository = "emqx/emqx"
		}
		if _, ok := r.(*EmqxEnterprise); ok {
			template.Spec.EmqxContainer.Image.Repository = "emqx/emqx-ee"
		}
	}
	r.GetSpec().SetTemplate(template)
}

func defaultEmqxACL(r Emqx) {
	template := r.GetSpec().GetTemplate()
	if len(template.Spec.EmqxContainer.EmqxACL) == 0 {
		template.Spec.EmqxContainer.EmqxACL = []string{
			`{allow, {user, "dashboard"}, subscribe, ["$SYS/#"]}.`,
			`{allow, {ipaddr, "127.0.0.1"}, pubsub, ["$SYS/#", "#"]}.`,
			`{deny, all, subscribe, ["$SYS/#", {eq, "#"}]}.`,
			`{allow, all}.`,
		}
	}
	r.GetSpec().SetTemplate(template)
}

func defaultEmqxConfig(r Emqx) {
	names := &Names{r}

	template := r.GetSpec().GetTemplate()
	if template.Spec.EmqxContainer.EmqxConfig == nil {
		template.Spec.EmqxContainer.EmqxConfig = make(map[string]string)
	}

	clusterConfig := make(map[string]string)
	clusterConfig["name"] = r.GetName()
	clusterConfig["log.to"] = "console"
	clusterConfig["cluster.discovery"] = "dns"
	clusterConfig["cluster.dns.type"] = "srv"
	clusterConfig["cluster.dns.app"] = r.GetName()
	clusterConfig["cluster.dns.name"] = fmt.Sprintf("%s.%s.svc.cluster.local", names.HeadlessSvc(), r.GetNamespace())
	clusterConfig["listener.tcp.internal"] = ""
	for k, v := range clusterConfig {
		if _, ok := template.Spec.EmqxContainer.EmqxConfig[k]; !ok {
			template.Spec.EmqxContainer.EmqxConfig[k] = v
		}
	}
	r.GetSpec().SetTemplate(template)
}

func defaultServiceTemplate(r Emqx) {
	s := r.GetSpec().GetServiceTemplate()

	s.ObjectMeta.Namespace = r.GetNamespace()
	if s.ObjectMeta.Name == "" {
		s.ObjectMeta.Name = r.GetName()
	}
	if s.ObjectMeta.Labels == nil {
		s.ObjectMeta.Labels = make(map[string]string)
	}
	for key, value := range r.GetLabels() {
		if _, ok := s.ObjectMeta.Labels[key]; !ok {
			s.ObjectMeta.Labels[key] = value
		}
	}
	if s.ObjectMeta.Annotations == nil {
		s.ObjectMeta.Annotations = map[string]string{}
	}
	for key, value := range r.GetAnnotations() {
		if key == "kubectl.kubernetes.io/last-applied-configuration" {
			continue
		}
		if _, ok := s.ObjectMeta.Annotations[key]; !ok {
			s.ObjectMeta.Annotations[key] = value
		}
	}

	s.Spec.Selector = r.GetLabels()
	s.Spec.Ports = MergeServicePorts(
		s.Spec.Ports,
		[]corev1.ServicePort{
			{
				Name:       "http-management-8081",
				Port:       8081,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt(8081),
			},
		},
	)

	r.GetSpec().SetServiceTemplate(s)
}

func validateImageVersion(r Emqx) error {
	version := r.GetSpec().GetTemplate().Spec.EmqxContainer.Image.Version
	if version == "latest" {
		return fmt.Errorf("image version can not be latest")
	}

	v, err := semver.NewVersion(version)
	if err != nil {
		return fmt.Errorf("invalid image version: %s", version)
	}
	if v.Compare(semver.MustParse("4.4.8")) < 0 {
		return fmt.Errorf("image version %s is too old, please upgrade to 4.4.8 or later", version)
	}
	if v.Compare(semver.MustParse("5.0.0")) >= 0 {
		return fmt.Errorf("image version %s is too new, please downgrade to 5.0.0 earlier", version)
	}

	return nil
}

func validateVolumeClaimTemplates(new, old Emqx) error {
	if !reflect.DeepEqual(new.GetSpec().GetPersistent(), old.GetSpec().GetPersistent()) {
		return errors.New("refuse to update Persistent ")
	}
	return nil
}
