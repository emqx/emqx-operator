package v1beta4

import (
	"io"
	"time"

	emperror "emperror.dev/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

// requeue provides a wrapper around different results from a subreconciler.
type requeue struct {
	err    error
	result *ctrl.Result
}

func processRequeue(requeue *requeue) (ctrl.Result, error) {
	if requeue == nil {
		return ctrl.Result{}, nil
	}
	if requeue.result != nil {
		return *requeue.result, nil
	}
	// Common Errors
	err := emperror.Unwrap(requeue.err)
	if io.EOF == err {
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}
	if k8sErrors.IsNotFound(err) {
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}
	if k8sErrors.IsConflict(err) {
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}
	return ctrl.Result{}, requeue.err
}
