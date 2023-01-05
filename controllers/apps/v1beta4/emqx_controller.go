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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	"github.com/emqx/emqx-operator/pkg/apiclient"
	innerErr "github.com/emqx/emqx-operator/pkg/errors"
	"github.com/emqx/emqx-operator/pkg/handler"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const EmqxContainerName string = "emqx"

var _ reconcile.Reconciler = &EmqxBrokerReconciler{}

type EmqxReconciler struct {
	*handler.Handler
	APIClient     *apiclient.APIClient
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
}

// subResult provides a wrapper around different results from a subreconciler.
type subResult struct {
	err    error
	result *ctrl.Result
	args   any
}

type emqxSubReconciler interface {
	reconcile(ctx context.Context, instance appsv1beta4.Emqx, args ...any) subResult
}

func NewEmqxReconciler(mgr manager.Manager) *EmqxReconciler {
	return &EmqxReconciler{
		Handler:       handler.NewHandler(mgr),
		APIClient:     apiclient.NewAPIClient(mgr),
		Scheme:        mgr.GetScheme(),
		EventRecorder: mgr.GetEventRecorderFor("emqx-controller"),
	}
}

func (r *EmqxReconciler) Do(ctx context.Context, instance appsv1beta4.Emqx) (ctrl.Result, error) {
	requestAPI := newRequestAPI(r.Client, r.APIClient, instance)

	var subResult subResult
	var subReconcilers = []emqxSubReconciler{
		updateEmqxStatus{EmqxReconciler: r, requestAPI: requestAPI},
		addEmqxInitResources{EmqxReconciler: r},
		addEmqxResources{EmqxReconciler: r, requestAPI: requestAPI},
		addEmqxStatefulSet{EmqxReconciler: r, requestAPI: requestAPI},
		updateEmqxStatus{EmqxReconciler: r, requestAPI: requestAPI},
	}
	for i := range subReconcilers {
		if reflect.ValueOf(subResult).FieldByName("args").IsValid() {
			subResult = subReconcilers[i].reconcile(ctx, instance, subResult.args)
		} else {
			subResult = subReconcilers[i].reconcile(ctx, instance)
		}
		subResult, err, _ := processResult(subResult)
		if err != nil || !subResult.IsZero() {
			return subResult, err
		}
	}

	return ctrl.Result{RequeueAfter: 20 * time.Second}, nil
}

func processResult(subResult subResult) (ctrl.Result, error, any) {
	if !subResult.result.IsZero() {
		return *subResult.result, subResult.err, subResult.args
	}
	// Common Errors
	if innerErr.IsCommonError(subResult.err) {
		return ctrl.Result{RequeueAfter: time.Second}, nil, subResult.args
	}
	return ctrl.Result{}, subResult.err, subResult.args
}
