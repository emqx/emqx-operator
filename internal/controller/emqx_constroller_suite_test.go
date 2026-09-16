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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	"github.com/emqx/emqx-operator/internal/handler"
)

var _ = Describe("CRD Defaults", Ordered, func() {
	var ns *corev1.Namespace

	BeforeAll(func() {
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "api-defaults-test-ns",
			},
		}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, ns)).To(Succeed())
	})

	It("defaults coreTemplate.spec when spec is missing", func() {
		instance := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": crd.GroupVersion.String(),
				"kind":       "EMQX",
				"metadata": map[string]interface{}{
					"name":      "emqx",
					"namespace": ns.Name,
				},
				"spec": map[string]interface{}{
					"image": "emqx",
					"coreTemplate": map[string]interface{}{
						"metadata": map[string]interface{}{},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, instance)).To(Succeed())

		actual := &crd.EMQX{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: "emqx"}, actual)).To(Succeed())
		Expect(actual.Spec.CoreTemplate.Spec).To(HaveField("Replicas", And(
			Not(BeNil()),
			HaveValue(BeEquivalentTo(1)),
		)))
		Expect(actual.Spec.CoreTemplate.Spec).To(HaveField("PodSecurityContext", And(
			Not(BeNil()),
			HaveValue(HaveField("RunAsUser", HaveValue(BeEquivalentTo(1000)))),
		)))
	})

	It("defaults the replicant scale path for core-only instances", func() {
		name := "emqx-core-only-scale"
		Expect(k8sClient.Create(ctx, &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": crd.GroupVersion.String(),
				"kind":       "EMQX",
				"metadata": map[string]interface{}{
					"name":      name,
					"namespace": ns.Name,
				},
				"spec": map[string]interface{}{
					"image": "emqx",
				},
			},
		})).To(Succeed())

		instance := &crd.EMQX{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: name}, instance)).
			To(Succeed())

		// Materialize status.replicantReplicas, which the scale subresource also requires.
		instance.Status.ReplicantReplicas = 0
		Expect(k8sClient.Status().Update(ctx, instance)).To(Succeed())

		scale := &autoscalingv1.Scale{}
		Expect(k8sClient.SubResource("scale").Get(ctx, instance, scale)).To(Succeed())
		Expect(scale.Spec.Replicas).To(BeEquivalentTo(0))
		Expect(scale.Status.Replicas).To(BeEquivalentTo(0))
	})

	It("defaults updateStrategy.evacuationStrategy for minimal instances", func() {
		instance := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": crd.GroupVersion.String(),
				"kind":       "EMQX",
				"metadata": map[string]interface{}{
					"name":      "emqx-update-strategy-defaults",
					"namespace": ns.Name,
				},
				"spec": map[string]interface{}{
					"image": "emqx",
				},
			},
		}
		Expect(k8sClient.Create(ctx, instance)).To(Succeed())

		actual := &crd.EMQX{}
		key := client.ObjectKey{Namespace: ns.Name, Name: "emqx-update-strategy-defaults"}
		Expect(k8sClient.Get(ctx, key, actual)).To(Succeed())
		Expect(actual.Spec.UpdateStrategy.Type).To(Equal("RollingUpdate"))
		Expect(actual.Spec.UpdateStrategy.EvacuationStrategy).To(Equal(crd.EvacuationStrategy{
			Type:            crd.NodeEvacuationStrategy,
			ConnEvictRate:   1000,
			SessEvictRate:   1000,
			WaitTakeover:    10,
			WaitHealthCheck: 60,
		}))

		actual.Annotations = map[string]string{"apps.emqx.io/test": "updated"}
		Expect(k8sClient.Update(ctx, actual)).To(Succeed())
	})

	It("preserves arbitrary JSON values under config roots", func() {
		instance := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": crd.GroupVersion.String(),
				"kind":       "EMQX",
				"metadata": map[string]interface{}{
					"name":      "emqx-config-roots",
					"namespace": ns.Name,
				},
				"spec": map[string]interface{}{
					"image": "emqx",
					"config": map[string]interface{}{
						"roots": map[string]interface{}{
							"listeners": map[string]interface{}{
								"tcp": map[string]interface{}{
									"default": map[string]interface{}{
										"bind":    int64(1883),
										"enabled": true,
									},
								},
							},
							"authentication": []interface{}{
								map[string]interface{}{"mechanism": "password_based"},
							},
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, instance)).To(Succeed())

		actual := &crd.EMQX{}
		key := client.ObjectKey{Namespace: ns.Name, Name: "emqx-config-roots"}
		Expect(k8sClient.Get(ctx, key, actual)).To(Succeed())

		Expect(actual.Spec.Config.Roots).To(HaveLen(2))
		Expect(actual.Spec.Config.Roots).To(HaveKey("listeners"))
		Expect(actual.Spec.Config.Roots["listeners"].Raw).To(MatchJSON(
			`{"tcp":{"default":{"bind":1883,"enabled":true}}}`,
		))
		Expect(actual.Spec.Config.Roots).To(HaveKey("authentication"))
		Expect(actual.Spec.Config.Roots["authentication"].Raw).To(MatchJSON(
			`[{"mechanism":"password_based"}]`,
		))
	})

})

var _ = Describe("CRD Validation", func() {
	It("rejects the dashboard port name in the core template", func() {
		instance := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": crd.GroupVersion.String(),
				"kind":       "EMQX",
				"metadata": map[string]interface{}{
					"name":      "reserved-dashboard-core",
					"namespace": "default",
				},
				"spec": map[string]interface{}{
					"image": "emqx",
					"coreTemplate": map[string]interface{}{
						"spec": map[string]interface{}{
							"ports": []interface{}{
								map[string]interface{}{
									"name":          "dashboard",
									"containerPort": int64(18083),
								},
							},
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, instance, client.DryRunAll)).To(MatchError(ContainSubstring(
			"port names dashboard and dashboard-https are reserved by the Operator",
		)))
	})

	It("rejects the dashboard-https port name in the replicant template", func() {
		instance := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": crd.GroupVersion.String(),
				"kind":       "EMQX",
				"metadata": map[string]interface{}{
					"name":      "reserved-dashboard-https-replicant",
					"namespace": "default",
				},
				"spec": map[string]interface{}{
					"image": "emqx",
					"coreTemplate": map[string]interface{}{
						"spec": map[string]interface{}{
							"replicas": int64(2),
						},
					},
					"replicantTemplate": map[string]interface{}{
						"spec": map[string]interface{}{
							"ports": []interface{}{
								map[string]interface{}{
									"name":          "dashboard-https",
									"containerPort": int64(18084),
								},
							},
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, instance, client.DryRunAll)).To(MatchError(ContainSubstring(
			"port names dashboard and dashboard-https are reserved by the Operator",
		)))
	})

	Context("Replicants update strategy", func() {
		DescribeTable("validates rollout budgets", func(unavailable, surge *intstr.IntOrString, valid bool, field string) {
			instance := &crd.EMQX{
				ObjectMeta: metav1.ObjectMeta{GenerateName: "rollout-validation-", Namespace: "default"},
				Spec: crd.EMQXSpec{
					Image: "emqx",
					UpdateStrategy: crd.UpdateStrategy{
						EvacuationStrategy: crd.EvacuationStrategy{Type: crd.NodeEvacuationStrategy},
						Replicants: &crd.ReplicantsUpdateStrategy{
							MaxUnavailable: unavailable,
							MaxSurge:       surge,
						},
					},
				},
			}
			err := k8sClient.Create(ctx, instance, client.DryRunAll)
			if valid {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(k8sErrors.IsInvalid(err)).To(BeTrue(), "%v", err)
				Expect(err.Error()).To(ContainSubstring("spec.updateStrategy.replicants" + field))
			}
		},
			Entry("omitted budgets", nil, nil, true, ""),
			Entry("non-negative integers", ptr.To(intstr.FromInt(1)), ptr.To(intstr.FromInt(0)), true, ""),
			Entry("negative unavailable", ptr.To(intstr.FromInt(-1)), nil, false, ".maxUnavailable"),
			Entry("negative surge", nil, ptr.To(intstr.FromInt(-1)), false, ".maxSurge"),
			Entry("percentages", ptr.To(intstr.FromString("25%")), ptr.To(intstr.FromString("25%")), true, ""),
			Entry("leading zeros", ptr.To(intstr.FromString("00099%")), ptr.To(intstr.FromString("00150%")), true, ""),
			Entry("100 percent unavailable with surge", ptr.To(intstr.FromString("00100%")), ptr.To(intstr.FromInt(1)), true, ""),
			Entry("100 percent unavailable with zero surge", ptr.To(intstr.FromString("00100%")), ptr.To(intstr.FromString("00%")), false, ".maxSurge"),
			Entry("100 percent unavailable with omitted surge", ptr.To(intstr.FromString("100%")), nil, false, ".maxSurge"),
			Entry("unavailable above 100 percent", ptr.To(intstr.FromString("101%")), nil, false, ".maxUnavailable"),
			Entry("surge above 100 percent", nil, ptr.To(intstr.FromString("150%")), true, ""),
			Entry("string count unavailable", ptr.To(intstr.FromString("1")), nil, false, ".maxUnavailable"),
			Entry("string count surge", nil, ptr.To(intstr.FromString("1")), false, ".maxSurge"),
			Entry("negative percentage", ptr.To(intstr.FromString("-1%")), nil, false, ".maxUnavailable"),
			Entry("fractional percentage", nil, ptr.To(intstr.FromString("1.5%")), false, ".maxSurge"),
			Entry("both integer zero", ptr.To(intstr.FromInt(0)), ptr.To(intstr.FromInt(0)), false, ".maxUnavailable"),
			Entry("both percentage zero", ptr.To(intstr.FromString("00%")), ptr.To(intstr.FromString("000%")), false, ".maxUnavailable"),
			Entry("mixed zeros", ptr.To(intstr.FromInt(0)), ptr.To(intstr.FromString("00%")), false, ".maxUnavailable"),
			Entry("reverse mixed zeros", ptr.To(intstr.FromString("00%")), ptr.To(intstr.FromInt(0)), false, ".maxUnavailable"),
			Entry("zero unavailable with omitted surge", ptr.To(intstr.FromString("00%")), nil, false, ".maxUnavailable"),
			Entry("omitted unavailable with zero surge", nil, ptr.To(intstr.FromString("00%")), true, ""),
			Entry("zero unavailable with surge", ptr.To(intstr.FromString("00%")), ptr.To(intstr.FromInt(1)), true, ""),
		)
	})
})

var _ = Describe("EMQX Reconciler / namespace termination", func() {
	var ns *corev1.Namespace
	var instance *crd.EMQX
	var reconciler *EMQXReconciler
	var recorder *record.FakeRecorder

	reconcileRequest := func() ctrl.Request {
		return ctrl.Request{NamespacedName: client.ObjectKeyFromObject(instance)}
	}

	BeforeEach(func() {
		ns = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "reconcile-termination-"}}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())
		DeferCleanup(func() {
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, ns))).To(Succeed())
		})
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		Expect(k8sClient.Create(ctx, instance)).To(Succeed())
		recorder = record.NewFakeRecorder(10)
		reconciler = emqxReconcilerDefault()
		reconciler.EventRecorder = recorder
	})

	It("stops without requeue or warning when the namespace is terminating", func() {
		Expect(k8sClient.Delete(ctx, ns)).To(Succeed())
		// Envtest has no namespace controller, so the EMQX remains undeleted.
		Expect(actualize(instance)).To(Succeed())
		Expect(instance.DeletionTimestamp).To(BeNil())
		Eventually(k8sClient.Create).WithArguments(ctx, generateNodeCookieSecret(instance)).
			Should(Satisfy(func(err error) bool {
				return k8sErrors.HasStatusCause(err, corev1.NamespaceTerminatingCause)
			}))
		result, err := reconciler.Reconcile(ctx, reconcileRequest())
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(ctrl.Result{}))
		Expect(recorder.Events).To(BeEmpty())
		Expect(k8sClient.Get(ctx, instance.NodeCookieNamespacedName(), &corev1.Secret{})).
			To(Satisfy(k8sErrors.IsNotFound))
	})

	It("still reports ordinary forbidden errors", func() {
		forbidden := k8sErrors.NewForbidden(
			schema.GroupResource{Resource: "secrets"},
			instance.NodeCookieNamespacedName().Name,
			fmt.Errorf("creation forbidden"),
		)
		// Allow the initial EMQX Get, then reject bootstrap Secret creation.
		reconciler.Handler = handler.NewHandler(newFaultyClient(k8sClient, &faultSequence{nil, forbidden}))
		_, err := reconciler.Reconcile(ctx, reconcileRequest())
		Expect(err).To(MatchError(forbidden))
		Expect(recorder.Events).To(Receive(ContainSubstring("Warning ReconcilerFailed")))
	})
})
