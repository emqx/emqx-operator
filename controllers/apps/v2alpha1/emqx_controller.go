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

package apps

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	"github.com/emqx/emqx-operator/pkg/handler"
	appsv1 "k8s.io/api/apps/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
)

const EMQXContainerName string = "emqx"

var (
	username      string = "admin"
	password      string = "public"
	dashboardPort string = "18083"
)

// EMQXReconciler reconciles a EMQX object
type EMQXReconciler struct {
	handler.Handler
	Scheme *runtime.Scheme
	record.EventRecorder
}

//+kubebuilder:rbac:groups=apps.emqx.io,resources=emqxes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.emqx.io,resources=emqxes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.emqx.io,resources=emqxes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EMQX object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *EMQXReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	instance := &appsv2alpha1.EMQX{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	headlessSvc := generateHeadlessService(instance)
	sts := generateStatefulSet(instance)

	resources := []client.Object{
		headlessSvc, sts,
	}

	if deploy := generateDeployment(instance); deploy != nil {
		resources = append(resources, deploy)
	}

	listenerPorts, err := r.getListenerPortsByAPI(sts)
	if err != nil {
		r.EventRecorder.Event(sts, corev1.EventTypeWarning, "FailedToGetListenerPorts", err.Error())
	}

	svcList := generateService(instance, listenerPorts)
	for _, svc := range svcList {
		resources = append(resources, svc)
	}

	if err := r.CreateOrUpdateList(instance, r.Scheme, resources, func(client.Object) error { return nil }); err != nil {
		return ctrl.Result{}, err
	}

	if !isExistReplicant(instance) {
		name := fmt.Sprintf("%s-%s", instance.Name, "replicant")
		deploy := &appsv1.Deployment{}
		svc := &corev1.Service{}

		if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: instance.Namespace}, deploy); !k8sErrors.IsNotFound(err) {
			if deploy.OwnerReferences[0].UID == instance.UID {
				r.Delete(ctx, deploy)
				return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
			}
		}

		if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: instance.Namespace}, svc); !k8sErrors.IsNotFound(err) {
			if svc.OwnerReferences[0].UID == instance.UID {
				r.Delete(ctx, svc)
				return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
			}
		}
	}

	// Check cluster status
	condition := appsv2alpha1.NewCondition(
		appsv2alpha1.ClusterRunning,
		corev1.ConditionTrue,
		"ClusterReady",
		"All resources are ready",
	)
	instance.Status.SetCondition(*condition)
	_ = r.Status().Update(ctx, instance)

	return ctrl.Result{RequeueAfter: time.Duration(20) * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EMQXReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv2alpha1.EMQX{}).
		Complete(r)
}

type emqxListener struct {
	Enable bool   `json:"enable"`
	Bind   string `json:"bind"`
	ID     string `json:"id"`
}

func (r *EMQXReconciler) getListenerPortsByAPI(obj client.Object) ([]corev1.ServicePort, error) {
	resp, body, err := r.Handler.RequestAPI(obj, "GET", username, password, dashboardPort, "api/v5/listeners")
	if err != nil {
		return nil, fmt.Errorf("failed to get listeners: %v", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get listener, status : %s, body: %s", resp.Status, body)
	}
	ports := []corev1.ServicePort{}
	listeners := []emqxListener{}
	if err := json.Unmarshal(body, &listeners); err != nil {
		return nil, fmt.Errorf("failed to parse listeners: %v", err)
	}
	for _, listener := range listeners {
		if !listener.Enable {
			continue
		}

		_, strPort, _ := net.SplitHostPort(listener.Bind)
		intPort, _ := strconv.Atoi(strPort)

		ports = append(ports, corev1.ServicePort{
			Name:       strings.ReplaceAll(listener.ID, ":", "-"),
			Port:       int32(intPort),
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromInt(intPort),
		})
	}
	return ports, nil
}

func mergeServicePorts(ports1, ports2 []corev1.ServicePort) []corev1.ServicePort {
	ports := append(ports1, ports2...)

	result := make([]corev1.ServicePort, 0, len(ports))
	temp := map[string]struct{}{}

	for _, item := range ports {
		if _, ok := temp[item.Name]; !ok {
			temp[item.Name] = struct{}{}
			result = append(result, item)
		}
	}

	return result
}

func generateHeadlessService(instance *appsv2alpha1.EMQX) *corev1.Service {
	headlessSvc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-headless", instance.Name),
			Namespace: instance.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type:                     corev1.ServiceTypeClusterIP,
			ClusterIP:                corev1.ClusterIPNone,
			SessionAffinity:          corev1.ServiceAffinityNone,
			PublishNotReadyAddresses: true,
			Ports: []corev1.ServicePort{
				{
					Name:       "ekka",
					Port:       4370,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(4370),
				},
			},
			Selector: instance.Spec.CoreTemplate.Labels,
		},
	}
	return headlessSvc
}

func generateService(instance *appsv2alpha1.EMQX, listenerPorts []corev1.ServicePort) (svcList []*corev1.Service) {
	instance.Spec.CoreTemplate.Spec.ServiceTemplate.Spec.Ports = mergeServicePorts(
		instance.Spec.CoreTemplate.Spec.ServiceTemplate.Spec.Ports,
		[]corev1.ServicePort{
			{
				Name:       "dashboard",
				Protocol:   corev1.ProtocolTCP,
				Port:       18083,
				TargetPort: intstr.FromInt(18083),
			},
		},
	)

	if isExistReplicant(instance) {
		instance.Spec.ReplicantTemplate.Spec.ServiceTemplate.Spec.Ports = mergeServicePorts(instance.Spec.ReplicantTemplate.Spec.ServiceTemplate.Spec.Ports, listenerPorts)
	} else {
		instance.Spec.CoreTemplate.Spec.ServiceTemplate.Spec.Ports = mergeServicePorts(instance.Spec.CoreTemplate.Spec.ServiceTemplate.Spec.Ports, listenerPorts)
	}

	coreSvc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-core", instance.Name),
			Namespace:   instance.Namespace,
			Labels:      instance.Spec.CoreTemplate.Spec.ServiceTemplate.Labels,
			Annotations: instance.Spec.CoreTemplate.Spec.ServiceTemplate.Annotations,
		},
		Spec: instance.Spec.CoreTemplate.Spec.ServiceTemplate.Spec,
	}
	svcList = append(svcList, coreSvc)

	if len(instance.Spec.ReplicantTemplate.Spec.ServiceTemplate.Spec.Ports) == 0 {
		return svcList
	}

	replicantSvc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-replicant", instance.Name),
			Namespace:   instance.Namespace,
			Labels:      instance.Spec.ReplicantTemplate.Spec.ServiceTemplate.Labels,
			Annotations: instance.Spec.ReplicantTemplate.Spec.ServiceTemplate.Annotations,
		},
		Spec: instance.Spec.ReplicantTemplate.Spec.ServiceTemplate.Spec,
	}

	return []*corev1.Service{coreSvc, replicantSvc}
}

func generateStatefulSet(instance *appsv2alpha1.EMQX) *appsv1.StatefulSet {
	sts := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-core", instance.Name),
			Namespace: instance.GetNamespace(),
			Labels:    instance.Spec.CoreTemplate.Labels,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: fmt.Sprintf("%s-headless", instance.Name),
			Replicas:    instance.Spec.CoreTemplate.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: instance.Spec.CoreTemplate.Labels,
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      instance.Spec.CoreTemplate.Labels,
					Annotations: instance.Spec.CoreTemplate.Annotations,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: instance.Spec.ImagePullSecrets,
					SecurityContext:  instance.Spec.SecurityContext,
					Affinity:         instance.Spec.CoreTemplate.Spec.Affinity,
					Tolerations:      instance.Spec.CoreTemplate.Spec.ToleRations,
					NodeName:         instance.Spec.CoreTemplate.Spec.NodeName,
					NodeSelector:     instance.Spec.CoreTemplate.Spec.NodeSelector,
					InitContainers:   instance.Spec.CoreTemplate.Spec.InitContainers,
					Containers: append(instance.Spec.CoreTemplate.Spec.ExtraContainers, corev1.Container{
						Name:            EMQXContainerName,
						Image:           instance.Spec.CoreTemplate.Spec.Image,
						ImagePullPolicy: corev1.PullPolicy(instance.Spec.CoreTemplate.Spec.ImagePullPolicy),
						Env: []corev1.EnvVar{
							{
								Name:  "EMQX_NODE__DB_ROLE",
								Value: "core",
							},
							{
								Name:  "EMQX_CLUSTER__DISCOVERY_STRATEGY",
								Value: "dns",
							},
							{
								Name:  "EMQX_CLUSTER__DNS__NAME",
								Value: fmt.Sprintf("%s-headless.%s.svc.cluster.local", instance.Name, instance.Namespace),
							},
							{
								Name:  "EMQX_CLUSTER__DNS__RECORD_TYPE",
								Value: "srv",
							},
							{
								Name:  "EMQX_CLUSTER__K8S__ADDRESS_TYPE",
								Value: "hostname",
							},
							{
								Name:  "EMQX_CLUSTER__K8S__NAMESPACE",
								Value: instance.Namespace,
							},
						},
						Args:           instance.Spec.CoreTemplate.Spec.Args,
						Resources:      instance.Spec.CoreTemplate.Spec.Resources,
						ReadinessProbe: instance.Spec.CoreTemplate.Spec.ReadinessProbe,
						LivenessProbe:  instance.Spec.CoreTemplate.Spec.LivenessProbe,
						StartupProbe:   instance.Spec.CoreTemplate.Spec.StartupProbe,
					}),
				},
			},
		},
	}
	return sts
}

func generateDeployment(instance *appsv2alpha1.EMQX) *appsv1.Deployment {
	if !isExistReplicant(instance) {
		return nil
		// replicas := int32(0)
		// return &appsv1.Deployment{
		// 	TypeMeta: metav1.TypeMeta{
		// 		APIVersion: "apps/v1",
		// 		Kind:       "Deployment",
		// 	},
		// 	ObjectMeta: metav1.ObjectMeta{
		// 		Name:      fmt.Sprintf("%s-replicant", instance.Name),
		// 		Namespace: instance.GetNamespace(),
		// 	},
		// 	Spec: appsv1.DeploymentSpec{
		// 		Replicas: &replicas,
		// 		Selector: &metav1.LabelSelector{
		// 			MatchLabels: map[string]string{
		// 				"apps.emqx.io/instance": "emqx",
		// 			},
		// 		},
		// 		Template: corev1.PodTemplateSpec{
		// 			ObjectMeta: metav1.ObjectMeta{
		// 				Labels: map[string]string{
		// 					"apps.emqx.io/instance": "emqx",
		// 				},
		// 			},
		// 			Spec: corev1.PodSpec{
		// 				Containers: []corev1.Container{
		// 					{
		// 						Name:  "emqx",
		// 						Image: "emqx",
		// 					},
		// 				},
		// 			},
		// 		},
		// 	},
		// }
	}

	hostSuffix := fmt.Sprintf("%s.%s.svc.cluster.local", fmt.Sprintf("%s-headless", instance.Name), instance.Namespace)

	coreNodes := []string{}
	for i := int32(0); i < *instance.Spec.CoreTemplate.Spec.Replicas; i++ {
		coreNodes = append(coreNodes,
			fmt.Sprintf(
				"%s@%s-%d.%s",
				EMQXContainerName, fmt.Sprintf("%s-core", instance.Name), i, hostSuffix,
			),
		)
	}
	coreNodesStr, _ := json.Marshal(coreNodes)

	deploy := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-replicant", instance.Name),
			Namespace: instance.GetNamespace(),
			Labels:    instance.Spec.ReplicantTemplate.Labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: instance.Spec.ReplicantTemplate.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: instance.Spec.ReplicantTemplate.Labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      instance.Spec.ReplicantTemplate.Labels,
					Annotations: instance.Spec.ReplicantTemplate.Annotations,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: instance.Spec.ImagePullSecrets,
					SecurityContext:  instance.Spec.SecurityContext,
					Affinity:         instance.Spec.ReplicantTemplate.Spec.Affinity,
					Tolerations:      instance.Spec.ReplicantTemplate.Spec.ToleRations,
					NodeName:         instance.Spec.ReplicantTemplate.Spec.NodeName,
					NodeSelector:     instance.Spec.ReplicantTemplate.Spec.NodeSelector,
					InitContainers:   instance.Spec.ReplicantTemplate.Spec.InitContainers,
					Containers: append(instance.Spec.ReplicantTemplate.Spec.ExtraContainers, corev1.Container{
						Name:            EMQXContainerName,
						Image:           instance.Spec.CoreTemplate.Spec.Image,
						ImagePullPolicy: corev1.PullPolicy(instance.Spec.CoreTemplate.Spec.ImagePullPolicy),
						Env: []corev1.EnvVar{
							{
								Name:  "EMQX_NODE__DB_ROLE",
								Value: "replicant",
							},
							{
								Name: "EMQX_HOST",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "status.podIP",
									},
								},
							},
							{
								Name:  "EMQX_CLUSTER__DISCOVERY_STRATEGY",
								Value: "static",
							},
							{
								Name:  "EMQX_CLUSTER__STATIC__SEEDS",
								Value: string(coreNodesStr),
							},
						},
						Args:           instance.Spec.CoreTemplate.Spec.Args,
						Resources:      instance.Spec.CoreTemplate.Spec.Resources,
						ReadinessProbe: instance.Spec.CoreTemplate.Spec.ReadinessProbe,
						LivenessProbe:  instance.Spec.CoreTemplate.Spec.LivenessProbe,
						StartupProbe:   instance.Spec.CoreTemplate.Spec.StartupProbe,
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "emqx-data",
								MountPath: "/opt/emqx/data",
							},
						},
					}),
					Volumes: []corev1.Volume{
						{
							Name: "emqx-data",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}
	return deploy
}

func isExistReplicant(instance *appsv2alpha1.EMQX) bool {
	return instance.Spec.ReplicantTemplate.Spec.Replicas != nil && *instance.Spec.ReplicantTemplate.Spec.Replicas > 0
}
