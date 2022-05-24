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

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	"github.com/emqx/emqx-operator/pkg/service"
)

var _ reconcile.Reconciler = &EmqxBrokerReconciler{}

type EmqxReconciler struct {
	Handler
}

func (r *EmqxReconciler) DoReconcile(ctx context.Context, obj v1beta3.Emqx) error {
	for _, resource := range service.Generate(obj) {
		var err error
		names := v1beta3.Names{Object: obj}
		switch resource.GetName() {
		case names.MQTTSCertificate():
			postUpdate := func() error {
				return r.Handler.ExecToPods(obj, "emqx", "emqx_ctl listeners restart mqtt:wss:external")
			}
			err = r.Handler.CreateOrUpdate(resource, postUpdate)
		case names.WSSCertificate():
			postUpdate := func() error {
				return r.Handler.ExecToPods(obj, "emqx", "emqx_ctl listeners restart mqtt:wss:external")
			}
			err = r.Handler.CreateOrUpdate(resource, postUpdate)
		default:
			err = r.Handler.CreateOrUpdate(resource, func() error { return nil })
		}

		if err != nil {
			r.EventRecorder.Event(obj, corev1.EventTypeWarning, "Reconciled", err.Error())
			obj.SetFailedCondition(err.Error())
			obj.DescConditionsByTime()
			_ = r.Status().Update(ctx, obj)
			return err
		}
	}

	obj.SetRunningCondition("Reconciled")
	obj.DescConditionsByTime()
	_ = r.Status().Update(ctx, obj)
	return nil
}
