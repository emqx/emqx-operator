package v2beta1

import (
	"fmt"

	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/ptr"

	// "k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var rebalance = appsv2beta1.Rebalance{
	TypeMeta: metav1.TypeMeta{
		Kind:       "Rebalance",
		APIVersion: appsv2beta1.GroupVersion.String(),
	},
	ObjectMeta: metav1.ObjectMeta{
		Name: "rebalance",
	},
	Spec: appsv2beta1.RebalanceSpec{
		RebalanceStrategy: appsv2beta1.RebalanceStrategy{
			WaitTakeover:     10,
			ConnEvictRate:    10,
			SessEvictRate:    10,
			WaitHealthCheck:  10,
			AbsSessThreshold: 100,
			RelConnThreshold: "1.2",
			AbsConnThreshold: 100,
			RelSessThreshold: "1.2",
		},
	},
}

var _ = Describe("EMQX 5 Rebalance Test", Label("rebalance"), func() {
	var instance *appsv2beta1.EMQX
	var r *appsv2beta1.Rebalance
	BeforeEach(func() {
		instance = genEMQX().DeepCopy()
		instance.Spec.Image = "emqx/emqx-enterprise:5.1"
	})

	Context("EMQX is not found", func() {
		BeforeEach(func() {
			r = rebalance.DeepCopy()
			r.Namespace = instance.GetNamespace()
			r.Spec.InstanceKind = instance.GroupVersionKind().Kind
			r.Spec.InstanceName = "no-exist"

			By("Creating namespace", func() {
				// create namespace
				Eventually(func() bool {
					err := k8sClient.Create(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: instance.GetNamespace(),
							Labels: map[string]string{
								"test": "e2e",
							},
						},
					})
					return err == nil || k8sErrors.IsAlreadyExists(err)
				}).Should(BeTrue())
			})

			By("Creating Rebalance CR", func() {
				Expect(k8sClient.Create(ctx, r)).Should(Succeed())
			})
		})

		AfterEach(func() {
			By("Deleting Rebalance CR, can be successful", func() {
				Eventually(func() error {
					return k8sClient.Delete(ctx, r)
				}, timeout, interval).Should(Succeed())
				Eventually(func() bool {
					return k8sErrors.IsNotFound(k8sClient.Get(ctx, client.ObjectKeyFromObject(r), r))
				}).Should(BeTrue())
			})
		})

		It("Rebalance will failed, because the EMQX is not found", func() {
			Eventually(func() appsv2beta1.RebalanceStatus {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(r), r)
				return r.Status
			}, timeout, interval).Should(And(
				HaveField("Phase", appsv2beta1.RebalancePhaseFailed),
				HaveField("RebalanceStates", BeNil()),
				HaveField("Conditions", ContainElements(
					HaveField("Type", appsv2beta1.RebalanceConditionFailed),
				)),
				HaveField("Conditions", ContainElements(
					HaveField("Status", corev1.ConditionTrue),
				)),
				HaveField("Conditions", ContainElements(
					HaveField("Message", ContainSubstring(fmt.Sprintf("%s is not found", r.Spec.InstanceName)))),
				),
			))
		})
	})

	Context("EMQX is not enterprise", func() {
		BeforeEach(func() {
			instance.Spec.Image = "emqx/emqx:5.1"
			r = rebalance.DeepCopy()
			r.Namespace = instance.GetNamespace()
			r.Spec.InstanceKind = instance.GroupVersionKind().Kind
			r.Spec.InstanceName = instance.Name

			By("Creating namespace", func() {
				// create namespace
				Eventually(func() bool {
					err := k8sClient.Create(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: instance.GetNamespace(),
							Labels: map[string]string{
								"test": "e2e",
							},
						},
					})
					return err == nil || k8sErrors.IsAlreadyExists(err)
				}).Should(BeTrue())
			})

			By("Creating EMQX CR", func() {
				// create EMQX CR
				instance.Spec.ReplicantTemplate = nil
				instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(2))
				Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

				// check EMQX CR if created successfully
				Eventually(func() *appsv2beta1.EMQX {
					_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
					return instance
				}).WithTimeout(timeout).WithPolling(interval).Should(
					WithTransform(func(instance *appsv2beta1.EMQX) bool {
						return instance.Status.IsConditionTrue(appsv2beta1.Ready)
					}, BeTrue()),
				)
			})

			By("Creating Rebalance CR", func() {
				Expect(k8sClient.Create(ctx, r)).Should(Succeed())
			})
		})

		AfterEach(func() {
			By("Deleting Rebalance CR, can be successful", func() {
				Eventually(func() error {
					return k8sClient.Delete(ctx, r)
				}, timeout, interval).Should(Succeed())
				Eventually(func() bool {
					return k8sErrors.IsNotFound(k8sClient.Get(ctx, client.ObjectKeyFromObject(r), r))
				}).Should(BeTrue())
			})
		})

		It("Rebalance will failed, because the EMQX is not enterprise", func() {
			Eventually(func() appsv2beta1.RebalanceStatus {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(r), r)
				return r.Status
			}, timeout, interval).Should(And(
				HaveField("Phase", appsv2beta1.RebalancePhaseFailed),
				HaveField("RebalanceStates", BeNil()),
				HaveField("Conditions", ContainElements(
					HaveField("Type", appsv2beta1.RebalanceConditionFailed),
				)),
				HaveField("Conditions", ContainElements(
					HaveField("Status", corev1.ConditionTrue),
				)),
				HaveField("Conditions", ContainElements(
					HaveField("Message", ContainSubstring("Only enterprise edition can be rebalanced"))),
				),
			))
		})
	})

	Context("EMQX is exist", func() {
		BeforeEach(func() {
			r = rebalance.DeepCopy()

			By("Creating namespace", func() {
				// create namespace
				Eventually(func() bool {
					err := k8sClient.Create(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: instance.GetNamespace(),
							Labels: map[string]string{
								"test": "e2e",
							},
						},
					})
					return err == nil || k8sErrors.IsAlreadyExists(err)
				}).Should(BeTrue())
			})

			By("Creating EMQX CR", func() {
				// create EMQX CR
				instance.Spec.ReplicantTemplate = nil
				instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(2))
				Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

				// check EMQX CR if created successfully
				Eventually(func() *appsv2beta1.EMQX {
					_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
					return instance
				}).WithTimeout(timeout).WithPolling(interval).Should(
					WithTransform(func(instance *appsv2beta1.EMQX) bool {
						return instance.Status.IsConditionTrue(appsv2beta1.Ready)
					}, BeTrue()),
				)
			})
		})

		AfterEach(func() {
			By("Deleting EMQX CR", func() {
				// delete emqx cr
				Eventually(func() error {
					return k8sClient.Delete(ctx, instance)
				}, timeout, interval).Should(Succeed())
				Eventually(func() bool {
					return k8sErrors.IsNotFound(k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance))
				}).Should(BeTrue())
			})

			// delete rebalance cr
			By("Deleting Rebalance CR", func() {
				Eventually(func() error {
					return k8sClient.Delete(ctx, r)
				}, timeout, interval).Should(Succeed())
				Eventually(func() bool {
					return k8sErrors.IsNotFound(k8sClient.Get(ctx, client.ObjectKeyFromObject(r), r))
				}).Should(BeTrue())
			})
		})

		It("Check rebalance status", func() {
			By("Create rebalance", func() {
				r.Namespace = instance.GetNamespace()
				r.Spec.InstanceName = instance.GetName()
				r.Spec.InstanceKind = instance.GroupVersionKind().Kind

				Expect(k8sClient.Create(ctx, r)).Should(Succeed())
			})

			By("Rebalance should have finalizer", func() {
				Eventually(func() []string {
					_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(r), r)
					return r.GetFinalizers()
				}, timeout, interval).Should(ContainElements("apps.emqx.io/finalizer"))
			})

			By("Rebalance will failed, because the EMQX is nothing to balance", func() {
				Eventually(func() appsv2beta1.RebalanceStatus {
					_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(r), r)
					return r.Status
				}, timeout, interval).Should(And(
					HaveField("Phase", appsv2beta1.RebalancePhaseFailed),
					HaveField("RebalanceStates", BeNil()),
					HaveField("Conditions", ContainElements(
						And(
							HaveField("Type", appsv2beta1.RebalanceConditionFailed),
							HaveField("Status", corev1.ConditionTrue),
							HaveField("Message", ContainSubstring("Failed to start rebalance")),
						),
					)),
				))
			})

			By("Mock rebalance is in progress", func() {
				// mock rebalance processing
				r.Status.Phase = appsv2beta1.RebalancePhaseProcessing
				r.Status.Conditions = []appsv2beta1.RebalanceCondition{}
				Expect(k8sClient.Status().Update(ctx, r)).Should(Succeed())

				// update annotations for target reconciler
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(r), r)).Should(Succeed())
				r.Annotations = map[string]string{"test": "e2e"}
				Expect(k8sClient.Update(ctx, r)).Should(Succeed())
			})

			By("Rebalance should completed", func() {
				Eventually(func() appsv2beta1.RebalanceStatus {
					_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(r), r)
					return r.Status
				}, timeout, interval).Should(And(
					HaveField("Phase", appsv2beta1.RebalancePhaseCompleted),
					HaveField("RebalanceStates", BeNil()),
					HaveField("Conditions", ContainElements(
						HaveField("Type", appsv2beta1.RebalanceConditionCompleted),
					)),
					HaveField("Conditions", ContainElements(
						HaveField("Status", corev1.ConditionTrue),
					)),
				))
			})
		})
	})
})

var _ = Describe("EMQX 4 Rebalance Test", func() {
	var instance *appsv1beta4.EmqxEnterprise
	var r *appsv2beta1.Rebalance
	BeforeEach(func() {
		instance = &appsv1beta4.EmqxEnterprise{
			TypeMeta: metav1.TypeMeta{
				Kind:       "EmqxEnterprise",
				APIVersion: "apps.emqx.io/v1beta4",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "emqx-ee",
				Namespace: "e2e-test-v2beta1" + "-" + rand.String(5),
				Labels: map[string]string{
					"test": "e2e",
				},
			},
			Spec: appsv1beta4.EmqxEnterpriseSpec{
				Replicas:      ptr.To(int32(1)),
				ClusterDomain: "cluster.local",
				Template: appsv1beta4.EmqxTemplate{
					Spec: appsv1beta4.EmqxTemplateSpec{
						EmqxContainer: appsv1beta4.EmqxContainer{
							Image: appsv1beta4.EmqxImage{
								Repository: "emqx/emqx-ee",
								Version:    "4.4.19",
							},
						},
					},
				},
			},
		}
		instance.Default()
	})

	Context("EMQX is not found", func() {
		BeforeEach(func() {
			r = rebalance.DeepCopy()
			r.Namespace = instance.GetNamespace()
			r.Spec.InstanceKind = instance.GroupVersionKind().Kind
			r.Spec.InstanceName = "no-exist"

			By("Creating namespace", func() {
				// create namespace
				Eventually(func() bool {
					err := k8sClient.Create(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: instance.GetNamespace(),
							Labels: map[string]string{
								"test": "e2e",
							},
						},
					})
					return err == nil || k8sErrors.IsAlreadyExists(err)
				}).Should(BeTrue())
			})

			By("Creating Rebalance CR", func() {
				Expect(k8sClient.Create(ctx, r)).Should(Succeed())
			})
		})

		AfterEach(func() {
			By("Deleting Rebalance CR, can be successful", func() {
				Eventually(func() error {
					return k8sClient.Delete(ctx, r)
				}, timeout, interval).Should(Succeed())
				Eventually(func() bool {
					return k8sErrors.IsNotFound(k8sClient.Get(ctx, client.ObjectKeyFromObject(r), r))
				}).Should(BeTrue())
			})
		})

		It("Rebalance will failed, because the EMQX is not found", func() {
			Eventually(func() appsv2beta1.RebalanceStatus {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(r), r)
				return r.Status
			}, timeout, interval).Should(And(
				HaveField("Phase", appsv2beta1.RebalancePhaseFailed),
				HaveField("RebalanceStates", BeNil()),
				HaveField("Conditions", ContainElements(
					HaveField("Type", appsv2beta1.RebalanceConditionFailed),
				)),
				HaveField("Conditions", ContainElements(
					HaveField("Status", corev1.ConditionTrue),
				)),
				HaveField("Conditions", ContainElements(
					HaveField("Message", ContainSubstring(fmt.Sprintf("%s is not found", r.Spec.InstanceName)))),
				),
			))
		})
	})

	Context("EMQX is exist", func() {
		BeforeEach(func() {
			r = rebalance.DeepCopy()

			By("Creating namespace", func() {
				// create namespace
				Eventually(func() bool {
					err := k8sClient.Create(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: instance.GetNamespace(),
							Labels: map[string]string{
								"test": "e2e",
							},
						},
					})
					return err == nil || k8sErrors.IsAlreadyExists(err)
				}).Should(BeTrue())
			})

			By("Creating EMQX CR", func() {
				// create EMQX CR
				Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

				// check EMQX CR if created successfully
				Eventually(func() *appsv1beta4.EmqxEnterprise {
					_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
					return instance
				}).WithTimeout(timeout).WithPolling(interval).Should(
					WithTransform(func(instance *appsv1beta4.EmqxEnterprise) bool {
						return instance.Status.IsConditionTrue(appsv1beta4.ConditionRunning)
					}, BeTrue()),
				)
			})
		})

		AfterEach(func() {
			By("Deleting EMQX CR", func() {
				// delete emqx cr
				Eventually(func() error {
					return k8sClient.Delete(ctx, instance)
				}, timeout, interval).Should(Succeed())
				Eventually(func() bool {
					return k8sErrors.IsNotFound(k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance))
				}).Should(BeTrue())
			})

			// delete rebalance cr
			By("Deleting Rebalance CR", func() {
				Eventually(func() error {
					return k8sClient.Delete(ctx, r)
				}, timeout, interval).Should(Succeed())
				Eventually(func() bool {
					return k8sErrors.IsNotFound(k8sClient.Get(ctx, client.ObjectKeyFromObject(r), r))
				}).Should(BeTrue())
			})
		})

		It("Check rebalance status", func() {
			By("Create rebalance", func() {
				r.Namespace = instance.GetNamespace()
				r.Spec.InstanceName = instance.GetName()
				r.Spec.InstanceKind = "EmqxEnterprise"

				Expect(k8sClient.Create(ctx, r)).Should(Succeed())
			})

			By("Rebalance should have finalizer", func() {
				Eventually(func() []string {
					_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(r), r)
					return r.GetFinalizers()
				}, timeout, interval).Should(ContainElements("apps.emqx.io/finalizer"))
			})

			By("Rebalance will failed, because the EMQX is nothing to balance", func() {
				Eventually(func() appsv2beta1.RebalanceStatus {
					_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(r), r)
					return r.Status
				}, timeout, interval).Should(And(
					HaveField("Phase", appsv2beta1.RebalancePhaseFailed),
					HaveField("RebalanceStates", BeNil()),
					HaveField("Conditions", ContainElements(
						And(
							HaveField("Type", appsv2beta1.RebalanceConditionFailed),
							HaveField("Status", corev1.ConditionTrue),
							HaveField("Message", ContainSubstring("Failed to start rebalance")),
						),
					)),
				))
			})

			By("Mock rebalance is in progress", func() {
				// mock rebalance processing
				r.Status.Phase = appsv2beta1.RebalancePhaseProcessing
				r.Status.Conditions = []appsv2beta1.RebalanceCondition{}
				Expect(k8sClient.Status().Update(ctx, r)).Should(Succeed())

				// update annotations for target reconciler
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(r), r)).Should(Succeed())
				r.Annotations = map[string]string{"test": "e2e"}
				Expect(k8sClient.Update(ctx, r)).Should(Succeed())
			})

			By("Rebalance should completed", func() {
				Eventually(func() appsv2beta1.RebalanceStatus {
					_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(r), r)
					return r.Status
				}, timeout, interval).Should(And(
					HaveField("Phase", appsv2beta1.RebalancePhaseCompleted),
					HaveField("RebalanceStates", BeNil()),
					HaveField("Conditions", ContainElements(
						HaveField("Type", appsv2beta1.RebalanceConditionCompleted),
					)),
					HaveField("Conditions", ContainElements(
						HaveField("Status", corev1.ConditionTrue),
					)),
				))
			})
		})
	})
})
