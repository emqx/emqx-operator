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
	"fmt"
	"reflect"

	emperror "emperror.dev/errors"

	// "github.com/gurkankaymak/hocon"
	hocon "github.com/rory-z/go-hocon"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
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
	r.defaultBootstrapConfig()
	r.defaultReplicas()
	r.defaultDashboardServiceTemplate()
	r.defaultProbe()
	r.defaultSecurityContext()
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-apps-emqx-io-v2alpha2-emqx,mutating=false,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=emqxes,verbs=create;update,versions=v2alpha2,name=validator.apps.emqx.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &EMQX{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *EMQX) ValidateCreate() error {
	emqxlog.Info("validate create", "name", r.Name)

	if _, err := hocon.ParseString(r.Spec.BootstrapConfig); err != nil {
		err = emperror.Wrap(err, "failed to parse bootstrap config")
		emqxlog.Error(err, "validate create failed")
		return err
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *EMQX) ValidateUpdate(old runtime.Object) error {
	emqxlog.Info("validate update", "name", r.Name)

	oldEMQX := old.(*EMQX)
	if !reflect.DeepEqual(oldEMQX.Spec.BootstrapAPIKeys, r.Spec.BootstrapAPIKeys) {
		err := emperror.Errorf("bootstrap APIKey cannot be updated")
		emqxlog.Error(err, "validate update failed")
		return err
	}

	config, err := hocon.ParseString(r.Spec.BootstrapConfig)
	if err != nil {
		err = emperror.Wrap(err, "failed to parse bootstrap config")
		emqxlog.Error(err, "validate update failed")
		return err
	}

	oldConfig, _ := hocon.ParseString(oldEMQX.Spec.BootstrapConfig)
	if !reflect.DeepEqual(oldConfig, config) {
		err := emperror.Errorf("bootstrap config cannot be updated, old bootstrap config: %s, new bootstrap config: %s", oldEMQX.Spec.BootstrapConfig, r.Spec.BootstrapConfig)
		emqxlog.Error(err, "validate update failed")
		return err
	}

	if !reflect.DeepEqual(oldEMQX.Spec.ReplicantTemplate.ObjectMeta, r.Spec.ReplicantTemplate.ObjectMeta) ||
		!reflect.DeepEqual(oldEMQX.Spec.CoreTemplate.ObjectMeta, r.Spec.CoreTemplate.ObjectMeta) {
		err := emperror.New(".spec.coreTemplate.metadata and .spec.replicantTemplate.metadata cannot be updated")
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

	if r.Spec.CoreTemplate.Name == "" {
		r.Spec.CoreTemplate.Name = r.CoreNodeNamespacedName().Name
	}

	if r.Spec.ReplicantTemplate.Name == "" {
		r.Spec.ReplicantTemplate.Name = r.ReplicantNodeNamespacedName().Name
	}

	if r.Spec.DashboardServiceTemplate.Name == "" {
		r.Spec.DashboardServiceTemplate.Name = r.DashboardServiceNamespacedName().Name
	}

	if r.Spec.ListenersServiceTemplate.Name == "" {
		r.Spec.ListenersServiceTemplate.Name = r.ListenersServiceNamespacedName().Name
	}
}

func (r *EMQX) defaultLabels() {
	if r.Labels == nil {
		r.Labels = make(map[string]string)
	}
	r.Labels["apps.emqx.io/managed-by"] = "emqx-operator"
	r.Labels["apps.emqx.io/instance"] = r.GetName()

	// Core
	if r.Spec.CoreTemplate.Labels == nil {
		r.Spec.CoreTemplate.Labels = make(map[string]string)
	}
	r.Spec.CoreTemplate.Labels["apps.emqx.io/instance"] = r.Name
	r.Spec.CoreTemplate.Labels["apps.emqx.io/managed-by"] = "emqx-operator"
	r.Spec.CoreTemplate.Labels["apps.emqx.io/db-role"] = "core"

	// Replicant
	if r.Spec.ReplicantTemplate.Labels == nil {
		r.Spec.ReplicantTemplate.Labels = make(map[string]string)
	}
	r.Spec.ReplicantTemplate.Labels["apps.emqx.io/instance"] = r.Name
	r.Spec.ReplicantTemplate.Labels["apps.emqx.io/managed-by"] = "emqx-operator"
	r.Spec.ReplicantTemplate.Labels["apps.emqx.io/db-role"] = "replicant"

	// Dashboard service
	if r.Spec.DashboardServiceTemplate.Labels == nil {
		r.Spec.DashboardServiceTemplate.Labels = make(map[string]string)
	}
	r.Spec.DashboardServiceTemplate.Labels["apps.emqx.io/instance"] = r.Name
	r.Spec.DashboardServiceTemplate.Labels["apps.emqx.io/managed-by"] = "emqx-operator"

	// Listeners service
	if r.Spec.ListenersServiceTemplate.Labels == nil {
		r.Spec.ListenersServiceTemplate.Labels = make(map[string]string)
	}
	r.Spec.ListenersServiceTemplate.Labels["apps.emqx.io/instance"] = r.Name
	r.Spec.ListenersServiceTemplate.Labels["apps.emqx.io/managed-by"] = "emqx-operator"
}

func (r *EMQX) defaultAnnotations() {
	annotations := r.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}
	delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")

	r.Spec.CoreTemplate.Annotations = mergeMap(r.Spec.CoreTemplate.Annotations, annotations)
	r.Spec.ReplicantTemplate.Annotations = mergeMap(r.Spec.ReplicantTemplate.Annotations, annotations)
	r.Spec.DashboardServiceTemplate.Annotations = mergeMap(r.Spec.DashboardServiceTemplate.Annotations, annotations)
	r.Spec.ListenersServiceTemplate.Annotations = mergeMap(r.Spec.ListenersServiceTemplate.Annotations, annotations)
}

func (r *EMQX) defaultBootstrapConfig() {
	dnsName := fmt.Sprintf("%s.%s.svc.cluster.local", r.HeadlessServiceNamespacedName().Name, r.Namespace)
	defaultBootstrapConfigStr := fmt.Sprintf(`
	node {
	  data_dir = data
	  etc_dir = etc
	}
	cluster {
		discovery_strategy = dns
		dns {
			record_type = srv
			name = "%s"
		}
	}
	dashboard {
	  listeners.http {
		bind = 18083
	  }
	  default_username = admin
	  default_password = public
	}
	listeners.tcp.default {
		bind = "0.0.0.0:1883"
		max_connections = 1024000
	}
	`, dnsName)

	bootstrapConfig := fmt.Sprintf("%s\n%s", defaultBootstrapConfigStr, r.Spec.BootstrapConfig)
	config, err := hocon.ParseString(bootstrapConfig)
	if err != nil {
		return
	}

	r.Spec.BootstrapConfig = config.String()
}

func (r *EMQX) defaultReplicas() {
	if r.Spec.CoreTemplate.Spec.Replicas == nil {
		r.Spec.CoreTemplate.Spec.Replicas = pointer.Int32Ptr(2)
	}
	if r.Spec.ReplicantTemplate.Spec.Replicas == nil {
		r.Spec.ReplicantTemplate.Spec.Replicas = pointer.Int32Ptr(3)
	}
}

func (r *EMQX) defaultDashboardServiceTemplate() {
	r.Spec.DashboardServiceTemplate.Spec.Selector = r.Spec.CoreTemplate.Labels
	dashboardPort, err := GetDashboardServicePort(r)
	if err != nil {
		emqxlog.Info("failed to get dashboard service port in bootstrap config, use 18083", "error", err)
		dashboardPort = &corev1.ServicePort{
			Name:       "dashboard-listeners-http-bind",
			Protocol:   corev1.ProtocolTCP,
			Port:       int32(18083),
			TargetPort: intstr.FromInt(18083),
		}
	}

	r.Spec.DashboardServiceTemplate.Spec.Ports = MergeServicePorts(
		r.Spec.DashboardServiceTemplate.Spec.Ports,
		[]corev1.ServicePort{
			*dashboardPort,
		},
	)
}

func (r *EMQX) defaultProbe() {
	dashboardPort, err := GetDashboardServicePort(r)
	if err != nil {
		emqxlog.Info("failed to get dashboard service port in bootstrap config, use 18083", "error", err)
		dashboardPort = &corev1.ServicePort{
			TargetPort: intstr.FromInt(18083),
		}
	}

	defaultReadinessProbe := &corev1.Probe{
		InitialDelaySeconds: 10,
		PeriodSeconds:       5,
		FailureThreshold:    12,
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/status",
				Port: dashboardPort.TargetPort,
			},
		},
	}

	defaultLivenessProbe := &corev1.Probe{
		InitialDelaySeconds: 60,
		PeriodSeconds:       30,
		FailureThreshold:    3,
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/status",
				Port: dashboardPort.TargetPort,
			},
		},
	}

	if r.Spec.CoreTemplate.Spec.ReadinessProbe == nil {
		r.Spec.CoreTemplate.Spec.ReadinessProbe = defaultReadinessProbe
	}
	if r.Spec.CoreTemplate.Spec.LivenessProbe == nil {
		r.Spec.CoreTemplate.Spec.LivenessProbe = defaultLivenessProbe
	}

	if r.Spec.ReplicantTemplate.Spec.ReadinessProbe == nil {
		r.Spec.ReplicantTemplate.Spec.ReadinessProbe = defaultReadinessProbe
	}
	if r.Spec.ReplicantTemplate.Spec.LivenessProbe == nil {
		r.Spec.ReplicantTemplate.Spec.LivenessProbe = defaultLivenessProbe
	}
}

func (r *EMQX) defaultSecurityContext() {
	if r.Spec.CoreTemplate.Spec.PodSecurityContext == nil {
		r.Spec.CoreTemplate.Spec.PodSecurityContext = &corev1.PodSecurityContext{
			RunAsUser:  pointer.Int64(1000),
			RunAsGroup: pointer.Int64(1000),
			FSGroup:    pointer.Int64(1000),
		}

		r.Spec.CoreTemplate.Spec.PodSecurityContext.FSGroupChangePolicy = (*corev1.PodFSGroupChangePolicy)(pointer.String("Always"))
		r.Spec.CoreTemplate.Spec.PodSecurityContext.SupplementalGroups = []int64{1000}
	}

	if r.Spec.ReplicantTemplate.Spec.PodSecurityContext == nil {
		r.Spec.ReplicantTemplate.Spec.PodSecurityContext = &corev1.PodSecurityContext{
			RunAsUser:  pointer.Int64(1000),
			RunAsGroup: pointer.Int64(1000),
			FSGroup:    pointer.Int64(1000),
		}

		r.Spec.ReplicantTemplate.Spec.PodSecurityContext.FSGroupChangePolicy = (*corev1.PodFSGroupChangePolicy)(pointer.String("Always"))
		r.Spec.ReplicantTemplate.Spec.PodSecurityContext.SupplementalGroups = []int64{1000}
	}
}
