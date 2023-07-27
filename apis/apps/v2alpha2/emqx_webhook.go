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

package v2alpha2

import (
	"reflect"

	emperror "emperror.dev/errors"

	// "github.com/gurkankaymak/hocon"

	hocon "github.com/rory-z/go-hocon"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
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

//+kubebuilder:webhook:path=/mutate-apps-emqx-io-v2alpha2-emqx,mutating=true,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxes,verbs=create;update,versions=v2alpha2,name=mutating.apps.emqx.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &EMQX{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *EMQX) Default() {
	emqxlog.Info("default", "name", r.Name)

	r.defaultNames()
	r.defaultLabels()
	r.defaultAnnotations()
	r.defaultConfiguration()
	r.defaultListenersServiceTemplate()
	r.defaultDashboardServiceTemplate()
	r.defaultContainerPort()
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-apps-emqx-io-v2alpha2-emqx,mutating=false,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxes,verbs=create;update,versions=v2alpha2,name=validator.apps.emqx.io,admissionReviewVersions={v1,v1beta1}

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

func (r *EMQX) defaultNames() {
	if r.Name == "" {
		r.Name = "emqx"
	}

	if r.Spec.DashboardServiceTemplate.Name == "" {
		r.Spec.DashboardServiceTemplate.Name = r.Name + "-dashboard"
	}

	if r.Spec.ListenersServiceTemplate.Name == "" {
		r.Spec.ListenersServiceTemplate.Name = r.Name + "-listeners"
	}

	if r.Spec.CoreTemplate.Name == "" {
		r.Spec.CoreTemplate.Name = r.Name + "-core"
	}

	if r.Spec.ReplicantTemplate != nil {
		if r.Spec.ReplicantTemplate.Name == "" {
			r.Spec.ReplicantTemplate.Name = r.Name + "-replicant"
		}
	}

}

func (r *EMQX) defaultLabels() {
	r.Labels = AddLabel(r.Labels, ManagerByLabelKey, "emqx-operator")
	r.Labels = AddLabel(r.Labels, InstanceNameLabelKey, r.GetName())

	// Dashboard service
	r.Spec.DashboardServiceTemplate.Labels = AddLabel(r.Spec.DashboardServiceTemplate.Labels, ManagerByLabelKey, "emqx-operator")
	r.Spec.DashboardServiceTemplate.Labels = AddLabel(r.Spec.DashboardServiceTemplate.Labels, InstanceNameLabelKey, r.GetName())

	// Listeners service
	r.Spec.ListenersServiceTemplate.Labels = AddLabel(r.Spec.ListenersServiceTemplate.Labels, ManagerByLabelKey, "emqx-operator")
	r.Spec.ListenersServiceTemplate.Labels = AddLabel(r.Spec.ListenersServiceTemplate.Labels, InstanceNameLabelKey, r.GetName())

	// Core
	r.Spec.CoreTemplate.Labels = AddLabel(r.Spec.CoreTemplate.Labels, ManagerByLabelKey, "emqx-operator")
	r.Spec.CoreTemplate.Labels = AddLabel(r.Spec.CoreTemplate.Labels, InstanceNameLabelKey, r.GetName())
	r.Spec.CoreTemplate.Labels = AddLabel(r.Spec.CoreTemplate.Labels, DBRoleLabelKey, "core")

	// Replicant
	if r.Spec.ReplicantTemplate != nil {
		r.Spec.ReplicantTemplate.Labels = AddLabel(r.Spec.ReplicantTemplate.Labels, ManagerByLabelKey, "emqx-operator")
		r.Spec.ReplicantTemplate.Labels = AddLabel(r.Spec.ReplicantTemplate.Labels, InstanceNameLabelKey, r.GetName())
		r.Spec.ReplicantTemplate.Labels = AddLabel(r.Spec.ReplicantTemplate.Labels, DBRoleLabelKey, "replicant")
	}
}

func (r *EMQX) defaultAnnotations() {
	annotations := r.DeepCopy().Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}
	delete(annotations, "kubectl.kubernetes.io/last-applied-config")
	delete(annotations, LastEMQXConfigAnnotationKey)

	r.Spec.DashboardServiceTemplate.Annotations = mergeMap(r.Spec.DashboardServiceTemplate.Annotations, annotations)
	r.Spec.ListenersServiceTemplate.Annotations = mergeMap(r.Spec.ListenersServiceTemplate.Annotations, annotations)
	r.Spec.CoreTemplate.Annotations = mergeMap(r.Spec.CoreTemplate.Annotations, annotations)
	if r.Spec.ReplicantTemplate != nil {
		r.Spec.ReplicantTemplate.Annotations = mergeMap(r.Spec.ReplicantTemplate.Annotations, annotations)
	}
}

func (r *EMQX) defaultConfiguration() {
	configuration, _ := hocon.ParseString(r.Spec.Config.Data)
	if configuration.GetString("listeners.tcp.default.bind") == "" {
		r.Spec.Config.Data = "dashboard.listeners.http.bind = 18083\n" + r.Spec.Config.Data
	}
}

func (r *EMQX) defaultListenersServiceTemplate() {
	r.Spec.ListenersServiceTemplate.Spec.Selector = r.Spec.CoreTemplate.Labels
	if IsExistReplicant(r) {
		r.Spec.ListenersServiceTemplate.Spec.Selector = r.Spec.ReplicantTemplate.Labels
	}
}

func (r *EMQX) defaultDashboardServiceTemplate() {
	r.Spec.DashboardServiceTemplate.Spec.Selector = r.Spec.CoreTemplate.Labels
	dashboardPort, err := GetDashboardServicePort(r.Spec.Config.Data)
	if err != nil {
		emqxlog.Info("failed to get dashboard service port in config, use 18083", "error", err)
		dashboardPort = &corev1.ServicePort{
			Name:       "dashboard",
			Protocol:   corev1.ProtocolTCP,
			Port:       18083,
			TargetPort: intstr.Parse("18083"),
		}
	}

	r.Spec.DashboardServiceTemplate.Spec.Ports = MergeServicePorts(
		r.Spec.DashboardServiceTemplate.Spec.Ports,
		[]corev1.ServicePort{
			*dashboardPort,
		},
	)
}

func (r *EMQX) defaultContainerPort() {
	var containerPort = corev1.ContainerPort{
		Name:          "dashboard",
		Protocol:      corev1.ProtocolTCP,
		ContainerPort: 18083,
	}

	svcPort, err := GetDashboardServicePort(r.Spec.Config.Data)
	if err != nil {
		emqxlog.Info("failed to get dashboard service port in config, use 18083", "error", err)
	} else {
		containerPort.ContainerPort = svcPort.Port
	}

	r.Spec.CoreTemplate.Spec.Ports = MergeContainerPorts(
		r.Spec.CoreTemplate.Spec.Ports,
		[]corev1.ContainerPort{
			containerPort,
		},
	)
	if r.Spec.ReplicantTemplate != nil {
		r.Spec.ReplicantTemplate.Spec.Ports = MergeContainerPorts(
			r.Spec.ReplicantTemplate.Spec.Ports,
			[]corev1.ContainerPort{
				containerPort,
			},
		)
	}
}
