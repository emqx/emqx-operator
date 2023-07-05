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
	r.defaultDashboardServiceTemplate()
	r.defaultContainerPort()
	r.defaultProbe()
	r.defaultSecurityContext()
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
	annotations := r.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}
	delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")

	r.Spec.DashboardServiceTemplate.Annotations = mergeMap(r.Spec.DashboardServiceTemplate.Annotations, annotations)
	r.Spec.ListenersServiceTemplate.Annotations = mergeMap(r.Spec.ListenersServiceTemplate.Annotations, annotations)
	r.Spec.CoreTemplate.Annotations = mergeMap(r.Spec.CoreTemplate.Annotations, annotations)
	if r.Spec.ReplicantTemplate != nil {
		r.Spec.ReplicantTemplate.Annotations = mergeMap(r.Spec.ReplicantTemplate.Annotations, annotations)
	}
}

func (r *EMQX) defaultBootstrapConfig() {
	dnsName := fmt.Sprintf("%s.%s.svc.%s", r.HeadlessServiceNamespacedName().Name, r.Namespace, r.Spec.ClusterDomain)
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
	listeners.tcp.default.bind = "0.0.0.0:1883"
	listeners.ssl.default.bind = "0.0.0.0:8883"
	listeners.ws.default.bind = "0.0.0.0:8083"
	listeners.wss.default.bind = "0.0.0.0:8084"
	`, dnsName)

	bootstrapConfig := fmt.Sprintf("%s\n%s", defaultBootstrapConfigStr, r.Spec.BootstrapConfig)
	config, err := hocon.ParseString(bootstrapConfig)
	if err != nil {
		return
	}

	r.Spec.BootstrapConfig = config.String()
}

func (r *EMQX) defaultDashboardServiceTemplate() {
	r.Spec.DashboardServiceTemplate.Spec.Selector = r.Spec.CoreTemplate.Labels
	dashboardPort, err := GetDashboardServicePort(r)
	if err != nil {
		emqxlog.Info("failed to get dashboard service port in bootstrap config, use 18083", "error", err)
		dashboardPort = &corev1.ServicePort{
			Name:       "dashboard-listeners-http-bind",
			Protocol:   corev1.ProtocolTCP,
			Port:       18083,
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

func (r *EMQX) defaultContainerPort() {
	var containerPort = corev1.ContainerPort{
		Name:          "dashboard-http",
		Protocol:      corev1.ProtocolTCP,
		ContainerPort: 18083,
	}

	svcPort, err := GetDashboardServicePort(r)
	if err != nil {
		emqxlog.Info("failed to get dashboard service port in bootstrap config, use 18083", "error", err)
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

	if r.Spec.ReplicantTemplate != nil {
		if r.Spec.ReplicantTemplate.Spec.ReadinessProbe == nil {
			r.Spec.ReplicantTemplate.Spec.ReadinessProbe = defaultReadinessProbe
		}
		if r.Spec.ReplicantTemplate.Spec.LivenessProbe == nil {
			r.Spec.ReplicantTemplate.Spec.LivenessProbe = defaultLivenessProbe
		}
	}
}

func (r *EMQX) defaultSecurityContext() {
	podSecurityContext := &corev1.PodSecurityContext{
		RunAsUser:           pointer.Int64(1000),
		RunAsGroup:          pointer.Int64(1000),
		FSGroup:             pointer.Int64(1000),
		FSGroupChangePolicy: (*corev1.PodFSGroupChangePolicy)(pointer.String("Always")),
		SupplementalGroups:  []int64{1000},
	}

	if r.Spec.CoreTemplate.Spec.PodSecurityContext == nil {
		r.Spec.CoreTemplate.Spec.PodSecurityContext = podSecurityContext.DeepCopy()
	}
	if r.Spec.ReplicantTemplate != nil {
		if r.Spec.ReplicantTemplate.Spec.PodSecurityContext == nil {
			r.Spec.ReplicantTemplate.Spec.PodSecurityContext = podSecurityContext.DeepCopy()
		}
	}
}
