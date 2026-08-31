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
	"reflect"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	. "github.com/emqx/emqx-operator/test/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = DescribeClientFaultMatrix("paused reconciliation", Ordered, func() {
	var (
		ns       *corev1.Namespace
		instance *crd.EMQX
	)

	BeforeAll(func() {
		ns = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			GenerateName: "controller-paused-reconciliation-test-",
		}}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, ns)).To(Succeed())
	})

	BeforeEach(func() {
		instance = emqx.DeepCopy()
		instance.Name = ""
		instance.GenerateName = "paused-reconciliation-"
		instance.Namespace = ns.Name
		instance.Labels = nil
	})

	AfterEach(func() {
		Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, instance))).To(Succeed())
	})

	reconcile := func(reconciler *EMQXReconciler, instance *crd.EMQX) (ctrl.Result, error) {
		return reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: instance.Namespace,
			Name:      instance.Name,
		}})
	}

	It("observes a new paused resource after creating only bootstrap Secrets", func() {
		util.AttachAnnotation(instance, crd.AnnotationPaused, "true")
		Expect(k8sClient.Create(ctx, instance)).To(Succeed())

		reconciler := emqxReconciler()
		Eventually(reconcile).WithArguments(reconciler, instance).
			Should(Equal(ctrl.Result{RequeueAfter: observationInterval}))

		Expect(actualize(instance)).To(Succeed())
		Expect(instance).To(HaveCondition(crd.Paused, And(
			HaveField("Status", Equal(metav1.ConditionTrue)),
			HaveField("Reason", Equal("AnnotationSet")),
		)))
		Expect(instance).To(HaveCondition(crd.EMQXAPIAvailable, And(
			HaveField("Status", Equal(metav1.ConditionFalse)),
			HaveField("Reason", Equal("NoRequester")),
		)))

		Expect(k8sClient.Get(ctx, instance.BootstrapAPIKeyNamespacedName(), &corev1.Secret{})).
			To(Succeed())
		Expect(k8sClient.Get(ctx, instance.NodeCookieNamespacedName(), &corev1.Secret{})).
			To(Succeed())

		for _, list := range []client.ObjectList{
			&corev1.ConfigMapList{},
			&corev1.ServiceList{},
			&appsv1.StatefulSetList{},
			&appsv1.ReplicaSetList{},
		} {
			Expect(k8sClient.List(ctx, list,
				client.InNamespace(instance.Namespace),
				client.MatchingLabels(instance.DefaultLabels()),
			)).To(Succeed())
			objects, err := meta.ExtractList(list)
			Expect(err).NotTo(HaveOccurred())
			Expect(objects).To(BeEmpty())
		}
	})

	It("refreshes status but leaves an existing workload unchanged", func() {
		util.AttachAnnotation(instance, crd.AnnotationPaused, "true")
		Expect(k8sClient.Create(ctx, instance)).To(Succeed())

		coreSet := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      instance.CoreName(),
				Namespace: instance.Namespace,
				Labels:    instance.DefaultLabelsWith(crd.CoreLabels()),
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: instance.Spec.CoreTemplate.Spec.Replicas,
				Selector: &metav1.LabelSelector{MatchLabels: instance.DefaultLabelsWith(crd.CoreLabels())},
				Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{
					Labels: instance.DefaultLabelsWith(crd.CoreLabels()),
				}},
				ServiceName: instance.HeadlessServiceNamespacedName().Name,
			},
		}
		Expect(k8sClient.Create(ctx, coreSet)).To(Succeed())
		coreSet.Status.Replicas = 3
		Expect(k8sClient.Status().Update(ctx, coreSet)).To(Succeed())
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(coreSet), coreSet)).To(Succeed())
		before := coreSet.DeepCopy()

		reconciler := emqxReconciler()
		Eventually(reconcile).WithArguments(reconciler, instance).
			Should(Equal(ctrl.Result{RequeueAfter: observationInterval}))

		after := &appsv1.StatefulSet{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(coreSet), after)).To(Succeed())
		Expect(reflect.DeepEqual(before, after)).To(BeTrue(), "paused reconciliation changed the StatefulSet")

		Expect(actualize(instance)).To(Succeed())
		Expect(instance.Status.CoreReplicas).To(Equal(int32(3)))
		Expect(instance).To(HaveCondition(crd.Paused, HaveField("Status", Equal(metav1.ConditionTrue))))
	})

	It("resumes normal convergence after removing the annotation", func() {
		util.AttachAnnotation(instance, crd.AnnotationPaused, "true")
		Expect(k8sClient.Create(ctx, instance)).To(Succeed())
		reconciler := emqxReconciler()
		Eventually(reconcile).WithArguments(reconciler, instance).
			Should(Equal(ctrl.Result{RequeueAfter: observationInterval}))

		Expect(actualize(instance)).To(Succeed())
		delete(instance.Annotations, crd.AnnotationPaused)
		Expect(k8sClient.Update(ctx, instance)).To(Succeed())

		reconciler = emqxReconciler()
		Eventually(func(g Gomega) {
			_, err := reconcile(reconciler, instance)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(actualize(instance)).To(Succeed())
			g.Expect(instance).To(HaveCondition(crd.Paused, And(
				HaveField("Status", Equal(metav1.ConditionFalse)),
				HaveField("Reason", Equal("AnnotationNotSet")),
			)))
			g.Expect(k8sClient.Get(ctx, instance.HeadlessServiceNamespacedName(), &corev1.Service{})).
				To(Succeed())
		}).Should(Succeed())
	})
})
