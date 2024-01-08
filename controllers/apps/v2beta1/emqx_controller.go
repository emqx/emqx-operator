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

package v2beta1

import (
	"context"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	emperror "emperror.dev/errors"
	innerErr "github.com/emqx/emqx-operator/internal/errors"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	"github.com/go-logr/logr"
	"github.com/rory-z/go-hocon"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	"github.com/emqx/emqx-operator/internal/handler"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
)

// subResult provides a wrapper around different results from a subreconciler.
type subResult struct {
	err    error
	result ctrl.Result
}

type subReconciler interface {
	reconcile(ctx context.Context, logger logr.Logger, instance *appsv2beta1.EMQX, r innerReq.RequesterInterface) subResult
}

// EMQXReconciler reconciles a EMQX object
type EMQXReconciler struct {
	*handler.Handler
	Clientset     *kubernetes.Clientset
	Config        *rest.Config
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
}

func NewEMQXReconciler(mgr manager.Manager) *EMQXReconciler {
	return &EMQXReconciler{
		Handler:       handler.NewHandler(mgr),
		Clientset:     kubernetes.NewForConfigOrDie(mgr.GetConfig()),
		Config:        mgr.GetConfig(),
		Scheme:        mgr.GetScheme(),
		EventRecorder: mgr.GetEventRecorderFor("emqx-controller"),
	}
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
	logger := log.FromContext(ctx)

	instance := &appsv2beta1.EMQX{}
	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if instance.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, nil
	}

	_, err := hocon.ParseString(instance.Spec.Config.Data)
	if err != nil {
		r.EventRecorder.Event(instance, corev1.EventTypeWarning, "InvalidConfig", "the .spec.config.data is not a valid HOCON config")
		return ctrl.Result{}, emperror.Wrap(err, "failed to parse config")
	}

	requester, err := newRequester(r.Client, instance)
	if err != nil {
		if k8sErrors.IsNotFound(emperror.Cause(err)) {
			_ = (&addBootstrap{r}).reconcile(ctx, logger, instance, nil)
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		return ctrl.Result{}, emperror.Wrap(err, "failed to get bootstrap user")
	}

	for _, subReconciler := range []subReconciler{
		&addBootstrap{r},
		&updateStatus{r},
		&syncConfig{r},
		&addSvc{r},
		&addCore{r},
		&addRepl{r},
		&updatePodConditions{r},
		&updateStatus{r},
		&syncPods{r},
		&syncSets{r},
	} {
		subResult := subReconciler.reconcile(ctx, logger, instance, requester)
		if !subResult.result.IsZero() {
			return subResult.result, nil
		}
		if subResult.err != nil {
			if innerErr.IsCommonError(subResult.err) {
				logger.V(1).Info("requeue reconcile", "reconciler", subReconciler, "reason", subResult.err)
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
			r.EventRecorder.Event(instance, corev1.EventTypeWarning, "ReconcilerFailed", emperror.Cause(subResult.err).Error())
			return ctrl.Result{}, subResult.err
		}
	}

	if !instance.Status.IsConditionTrue(appsv2beta1.Ready) {
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}
	return ctrl.Result{RequeueAfter: time.Duration(30) * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EMQXReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv2beta1.EMQX{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				// Ignore updates to CR status in which case metadata.Generation does not change
				return e.ObjectNew.GetGeneration() != e.ObjectOld.GetGeneration()
			},
		}).
		Complete(r)
}

func newRequester(k8sClient client.Client, instance *appsv2beta1.EMQX) (innerReq.RequesterInterface, error) {
	username, password, err := getBootstrapAPIKey(context.Background(), k8sClient, instance)
	if err != nil {
		return nil, err
	}

	portMap, err := appsv2beta1.GetDashboardPortMap(instance.Spec.Config.Data)
	if err != nil {
		return nil, err
	}

	var schema, port string
	if dashboardHttps, ok := portMap["dashboard-https"]; ok {
		schema = "https"
		port = strconv.FormatInt(int64(dashboardHttps), 10)
	}
	if dashboard, ok := portMap["dashboard"]; ok {
		schema = "http"
		port = strconv.FormatInt(int64(dashboard), 10)
	}

	podList := &corev1.PodList{}
	_ = k8sClient.List(context.Background(), podList,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(
			appsv2beta1.DefaultCoreLabels(instance),
		),
	)
	sort.Slice(podList.Items, func(i, j int) bool {
		return podList.Items[i].CreationTimestamp.Before(&podList.Items[j].CreationTimestamp)
	})

	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodRunning && pod.Status.PodIP != "" {
			return &innerReq.Requester{
				Schema:   schema,
				Host:     net.JoinHostPort(pod.Status.PodIP, port),
				Username: username,
				Password: password,
			}, nil
		}
	}

	return nil, nil
}

func getBootstrapAPIKey(ctx context.Context, client client.Client, instance *appsv2beta1.EMQX) (username, password string, err error) {
	bootstrapAPIKey := &corev1.Secret{}
	if err = client.Get(ctx, types.NamespacedName{
		Namespace: instance.GetNamespace(),
		Name:      instance.GetName() + "-bootstrap-api-key",
	}, bootstrapAPIKey); err != nil {
		err = emperror.Wrap(err, "get secret failed")
		return
	}

	if data, ok := bootstrapAPIKey.Data["bootstrap_api_key"]; ok {
		users := strings.Split(string(data), "\n")
		for _, user := range users {
			index := strings.Index(user, ":")
			if index > 0 && user[:index] == appsv2beta1.DefaultBootstrapAPIKey {
				username = user[:index]
				password = user[index+1:]
				return
			}
		}
	}

	err = emperror.Errorf("the secret does not contain the bootstrap_api_key")
	return
}
