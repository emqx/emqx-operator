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
	"errors"
	"reflect"
	"strconv"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var rebalancelog = logf.Log.WithName("rebalance-resource")

func (r *Rebalance) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-apps-emqx-io-v1beta4-rebalance,mutating=false,failurePolicy=fail,sideEffects=None,groups=apps.emqx.io,resources=rebalances,verbs=create;update,versions=v1beta4,name=validator.rebalance.emqx.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Rebalance{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Rebalance) ValidateCreate() error {
	rebalancelog.Info("validate create", "name", r.Name)

	if len(r.Spec.RebalanceStrategy.RelConnThreshold) > 0 {
		_, err := strconv.ParseFloat(r.Spec.RebalanceStrategy.RelConnThreshold, 64)
		if err != nil {
			return errors.New(`the field ".spec.rebalanceStrategy.relConnThreshold" must be float64`)
		}
	}

	if len(r.Spec.RebalanceStrategy.RelSessThreshold) > 0 {
		_, err := strconv.ParseFloat(r.Spec.RebalanceStrategy.RelSessThreshold, 64)
		if err != nil {
			return errors.New(`the field ".spec.rebalanceStrategy.relSessThreshold" must be float64`)
		}
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Rebalance) ValidateUpdate(old runtime.Object) error {
	rebalancelog.Info("validate update", "name", r.Name)
	oldRebalance := old.(*Rebalance)

	newCopyRebalance := r.DeepCopy()
	oldCopyRebalance := oldRebalance.DeepCopy()

	if !reflect.DeepEqual(oldCopyRebalance.Spec, newCopyRebalance.Spec) {
		return errors.New("the Rebalance spec don't allow update, you can delete this and create new one")
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Rebalance) ValidateDelete() error {
	rebalancelog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
