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
	"reflect"
	"time"

	emperror "emperror.dev/errors"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"

	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	innerErr "github.com/emqx/emqx-operator/internal/errors"
	"github.com/emqx/emqx-operator/internal/handler"
	ctrl "sigs.k8s.io/controller-runtime"
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
	reconcile(ctx context.Context, instance appsv1beta4.Emqx, args ...any) subResult
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

	e, err := newEmqxHttpAPI(r.Client, instance, nil)
	if err != nil {
		if k8sErrors.IsNotFound(emperror.Cause(err)) {
			_ = addEmqxBootstrapUser{EmqxReconciler: r}.reconcile(ctx, instance)
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		if !innerErr.IsCommonError(err) {
			return ctrl.Result{}, emperror.Wrap(err, "failed to create request Pod API")
		}
	}

	var subResult subResult
	var subReconcilers = []emqxSubReconciler{
		updateEmqxStatus{EmqxReconciler: r, EmqxHttpAPI: e},
		addEmqxBootstrapUser{EmqxReconciler: r},
		addEmqxPlugins{EmqxReconciler: r},
		addEmqxResources{EmqxReconciler: r, EmqxHttpAPI: e},
		addEmqxStatefulSet{EmqxReconciler: r, EmqxHttpAPI: e},
		addListener{EmqxReconciler: r, EmqxHttpAPI: e},
		updateEmqxStatus{EmqxReconciler: r, EmqxHttpAPI: e},
		updatePodConditions{EmqxReconciler: r, EmqxHttpAPI: e},
	}
	for i := range subReconcilers {
		if reflect.ValueOf(subResult).FieldByName("args").IsValid() {
			subResult = subReconcilers[i].reconcile(ctx, instance, subResult.args)
		} else {
			subResult = subReconcilers[i].reconcile(ctx, instance)
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
