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
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	emperror "emperror.dev/errors"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"

	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	innerErr "github.com/emqx/emqx-operator/internal/errors"
	"github.com/emqx/emqx-operator/internal/handler"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const EmqxContainerName string = "emqx"

// subResult provides a wrapper around different results from a subreconciler.
type subResult struct {
	cont   bool // continue to next sub reconciler
	err    error
	result ctrl.Result
	args   any
}

type emqxSubReconciler interface {
	reconcile(ctx context.Context, logger logr.Logger, instance appsv1beta4.Emqx, args ...any) subResult
}

var _ reconcile.Reconciler = &EmqxBrokerReconciler{}

type EmqxReconciler struct {
	*handler.Handler
	Clientset     *kubernetes.Clientset
	Config        *rest.Config
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
}

func NewEmqxReconciler(mgr manager.Manager) *EmqxReconciler {
	return &EmqxReconciler{
		Handler:       handler.NewHandler(mgr),
		Clientset:     kubernetes.NewForConfigOrDie(mgr.GetConfig()),
		Config:        mgr.GetConfig(),
		Scheme:        mgr.GetScheme(),
		EventRecorder: mgr.GetEventRecorderFor("emqx-controller"),
	}
}

func (r *EmqxReconciler) Do(ctx context.Context, instance appsv1beta4.Emqx) (ctrl.Result, error) {
	if instance.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, nil
	}

	logger := log.FromContext(ctx)

	requester, err := newRequesterBySvc(ctx, r.Client, instance)
	if err != nil {
		if k8sErrors.IsNotFound(emperror.Cause(err)) {
			_ = addEmqxBootstrapUser{EmqxReconciler: r}.reconcile(ctx, logger, instance)
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		if !innerErr.IsCommonError(err) {
			return ctrl.Result{}, emperror.Wrap(err, "failed to create request Pod API")
		}
	}

	var subResult subResult
	var subReconcilers = []emqxSubReconciler{
		updateEmqxStatus{EmqxReconciler: r, Requester: requester},
		addEmqxBootstrapUser{EmqxReconciler: r},
		addEmqxPlugins{EmqxReconciler: r},
		addEmqxResources{EmqxReconciler: r, Requester: requester},
		addEmqxStatefulSet{EmqxReconciler: r, Requester: requester},
		addListener{EmqxReconciler: r, Requester: requester},
		updateEmqxStatus{EmqxReconciler: r, Requester: requester},
		updatePodConditions{EmqxReconciler: r, Requester: requester},
	}
	for i := range subReconcilers {
		if reflect.ValueOf(subResult).FieldByName("args").IsValid() {
			subResult = subReconcilers[i].reconcile(ctx, logger, instance, subResult.args)
		} else {
			subResult = subReconcilers[i].reconcile(ctx, logger, instance)
		}
		subResult, err := r.processResult(subResult, instance)
		if err != nil || !subResult.IsZero() {
			return subResult, err
		}
	}

	return ctrl.Result{RequeueAfter: 20 * time.Second}, nil
}

func (r *EmqxReconciler) processResult(subResult subResult, instance appsv1beta4.Emqx) (ctrl.Result, error) {
	if subResult.cont {
		if subResult.err != nil {
			r.EventRecorder.Event(instance, corev1.EventTypeWarning, "ReconcileError", subResult.err.Error())
		}
		return ctrl.Result{}, nil
	}

	if subResult.err != nil && innerErr.IsCommonError(subResult.err) {
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	return subResult.result, subResult.err
}

func NewRequesterByPod(ctx context.Context, k8sClient client.Client, instance appsv1beta4.Emqx) (innerReq.RequesterInterface, error) {
	username, password, err := getBootstrapUser(ctx, k8sClient, instance)
	if err != nil {
		return nil, err
	}

	podList := &corev1.PodList{}
	_ = k8sClient.List(ctx, podList,
		client.InNamespace(instance.GetNamespace()),
		client.MatchingLabels(instance.GetSpec().GetTemplate().Labels),
	)
	for _, pod := range podList.Items {
		for _, c := range pod.Status.Conditions {
			if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
				return &innerReq.Requester{
					Host:     fmt.Sprintf("%s:8081", pod.Status.PodIP),
					Username: username,
					Password: password,
				}, nil
			}
		}
	}
	return nil, emperror.New("failed to get ready pod")
}

func newRequesterBySvc(ctx context.Context, client client.Client, instance appsv1beta4.Emqx) (innerReq.RequesterInterface, error) {
	username, password, err := getBootstrapUser(ctx, client, instance)
	if err != nil {
		return nil, err
	}

	names := appsv1beta4.Names{Object: instance}
	return &innerReq.Requester{
		// TODO: the telepersence is not support `$service.$namespace.svc` format in Linux
		// Host:     fmt.Sprintf("%s.%s.svc:8081", names.HeadlessSvc(), instance.GetNamespace()),
		Host:     fmt.Sprintf("%s.%s.svc.%s:8081", names.HeadlessSvc(), instance.GetNamespace(), instance.GetSpec().GetClusterDomain()),
		Username: username,
		Password: password,
	}, nil
}

func getBootstrapUser(ctx context.Context, client client.Client, instance appsv1beta4.Emqx) (username, password string, err error) {
	bootstrapUser := &corev1.Secret{}
	if err = client.Get(ctx, types.NamespacedName{
		Namespace: instance.GetNamespace(),
		Name:      instance.GetName() + "-bootstrap-user",
	}, bootstrapUser); err != nil {
		err = emperror.Wrap(err, "get secret failed")
		return
	}

	if data, ok := bootstrapUser.Data["bootstrap_user"]; ok {
		users := strings.Split(string(data), "\n")
		for _, user := range users {
			index := strings.Index(user, ":")
			if index > 0 && user[:index] == defUsername {
				username = user[:index]
				password = user[index+1:]
				return
			}
		}
	}

	err = emperror.Errorf("the secret does not contain the bootstrap_user")
	return
}
