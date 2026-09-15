/*
Copyright 2026.

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

package controller

import (
	"fmt"

	emperror "emperror.dev/errors"
	crd "github.com/emqx/emqx-operator/api/v3beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type pauseReconciliation struct {
	*EMQXReconciler
}

// reconcile updates the Paused condition and stops the flat reconciliation
// pipeline before any post-bootstrap Kubernetes or EMQX state can be mutated.
func (p *pauseReconciliation) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	paused := reconciliationPaused(instance)
	changed := false
	if paused {
		changed = instance.Status.SetCondition(
			crd.Paused,
			metav1.ConditionTrue,
			"AnnotationSet",
			fmt.Sprintf("Reconciliation is paused by the %s annotation", crd.AnnotationPaused),
			instance.Generation,
		)
	} else {
		reason := "AnnotationNotSet"
		message := "Reconciliation is active"
		if _, present := instance.Annotations[crd.AnnotationPaused]; present {
			reason = "AnnotationFalse"
			message = fmt.Sprintf("Reconciliation is active because the %s annotation is false", crd.AnnotationPaused)
		}
		changed = instance.Status.SetCondition(
			crd.Paused,
			metav1.ConditionFalse,
			reason,
			message,
			instance.Generation,
		)
	}

	if changed {
		if err := p.Client.Status().Update(r.ctx, instance); err != nil {
			return reconcileError(emperror.Wrap(err, "failed to update pause status"))
		}
	}
	if paused {
		return reconcileRequeueAfter(observationInterval)
	}

	return subResult{}
}

func reconciliationPaused(instance *crd.EMQX) bool {
	value, present := instance.Annotations[crd.AnnotationPaused]
	if !present {
		return false
	}
	return value != "false"
}
