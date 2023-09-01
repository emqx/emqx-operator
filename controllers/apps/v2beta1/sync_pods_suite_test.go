package v2beta1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Check sync pods controller", Ordered, Label("node"), func() {
	var s *syncPods
	var fakeR *innerReq.FakeRequester = &innerReq.FakeRequester{}
	var instance *appsv2beta1.EMQX = new(appsv2beta1.EMQX)
	var ns *corev1.Namespace = &corev1.Namespace{}

	var updateSts, currentSts *appsv1.StatefulSet
	var updateRs, currentRs *appsv1.ReplicaSet
	var currentRsPod *corev1.Pod

	BeforeEach(func() {
		fakeR.ReqFunc = func(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error) {
			resp = &http.Response{
				StatusCode: 200,
			}
			respBody, _ = json.Marshal(&appsv2beta1.EMQXNode{
				Edition: "Opensource",
			})
			return resp, respBody, nil
		}

		s = &syncPods{emqxReconciler}
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "controller-v2beta1-update-emqx-nodes-test-" + rand.String(5),
				Labels: map[string]string{
					"test": "e2e",
				},
			},
		}
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Spec.ReplicantTemplate = &appsv2beta1.EMQXReplicantTemplate{
			Spec: appsv2beta1.EMQXReplicantTemplateSpec{
				Replicas: pointer.Int32Ptr(1),
			},
		}
		instance.Status = appsv2beta1.EMQXStatus{
			Conditions: []metav1.Condition{
				{
					Type:               appsv2beta1.Available,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
				},
			},
			CoreNodesStatus: appsv2beta1.EMQXNodesStatus{
				UpdateRevision:  "update",
				UpdateReplicas:  1,
				CurrentRevision: "current",
				CurrentReplicas: 1,
				ReadyReplicas:   2,
				Replicas:        1,
			},
			ReplicantNodesStatus: &appsv2beta1.EMQXNodesStatus{
				UpdateRevision:  "update",
				UpdateReplicas:  1,
				CurrentRevision: "current",
				CurrentReplicas: 1,
				ReadyReplicas:   2,
				Replicas:        1,
			},
		}

		Expect(k8sClient.Create(context.TODO(), ns)).Should(Succeed())

		updateSts = &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: instance.Name + "-",
				Namespace:    instance.Namespace,
				Labels: appsv2beta1.CloneAndAddLabel(
					appsv2beta1.DefaultCoreLabels(instance),
					appsv2beta1.LabelsPodTemplateHashKey,
					"update",
				),
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: pointer.Int32Ptr(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: appsv2beta1.CloneAndAddLabel(
						appsv2beta1.DefaultCoreLabels(instance),
						appsv2beta1.LabelsPodTemplateHashKey,
						"update",
					),
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: appsv2beta1.CloneAndAddLabel(
							appsv2beta1.DefaultCoreLabels(instance),
							appsv2beta1.LabelsPodTemplateHashKey,
							"update",
						),
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "emqx", Image: "emqx"},
						},
					},
				},
			},
		}

		currentSts = updateSts.DeepCopy()
		currentSts.Labels[appsv2beta1.LabelsPodTemplateHashKey] = "current"
		currentSts.Spec.Selector.MatchLabels[appsv2beta1.LabelsPodTemplateHashKey] = "current"
		currentSts.Spec.Template.Labels[appsv2beta1.LabelsPodTemplateHashKey] = "current"

		Expect(k8sClient.Create(context.Background(), updateSts)).Should(Succeed())
		updateSts.Status.Replicas = 1
		updateSts.Status.ReadyReplicas = 1
		Expect(k8sClient.Status().Update(context.Background(), updateSts)).Should(Succeed())

		Expect(k8sClient.Create(context.Background(), currentSts)).Should(Succeed())
		currentSts.Status.Replicas = 1
		currentSts.Status.ReadyReplicas = 1
		Expect(k8sClient.Status().Update(context.Background(), currentSts)).Should(Succeed())

		updateRs = &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: instance.Name + "-",
				Namespace:    instance.Namespace,
				Labels: appsv2beta1.CloneAndAddLabel(
					appsv2beta1.DefaultReplicantLabels(instance),
					appsv2beta1.LabelsPodTemplateHashKey,
					"update",
				),
			},
			Spec: appsv1.ReplicaSetSpec{
				Replicas: pointer.Int32Ptr(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: appsv2beta1.CloneAndAddLabel(
						appsv2beta1.DefaultReplicantLabels(instance),
						appsv2beta1.LabelsPodTemplateHashKey,
						"update",
					),
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: appsv2beta1.CloneAndAddLabel(
							appsv2beta1.DefaultReplicantLabels(instance),
							appsv2beta1.LabelsPodTemplateHashKey,
							"update",
						),
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "emqx", Image: "emqx"},
						},
					},
				},
			},
		}

		currentRs = updateRs.DeepCopy()
		currentRs.Labels[appsv2beta1.LabelsPodTemplateHashKey] = "current"
		currentRs.Spec.Selector.MatchLabels[appsv2beta1.LabelsPodTemplateHashKey] = "current"
		currentRs.Spec.Template.Labels[appsv2beta1.LabelsPodTemplateHashKey] = "current"

		Expect(k8sClient.Create(context.Background(), updateRs)).Should(Succeed())
		updateRs.Status.Replicas = 1
		updateRs.Status.ReadyReplicas = 1
		Expect(k8sClient.Status().Update(context.Background(), updateRs)).Should(Succeed())

		Expect(k8sClient.Create(context.Background(), currentRs)).Should(Succeed())
		currentRs.Status.Replicas = 1
		currentRs.Status.ReadyReplicas = 1
		Expect(k8sClient.Status().Update(context.Background(), currentRs)).Should(Succeed())

		currentRsPod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: currentRs.Name + "-",
				Namespace:    currentRs.Namespace,
				Labels:       currentRs.Spec.Template.Labels,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "apps/v1",
						Kind:       "ReplicaSet",
						Name:       currentRs.Name,
						UID:        currentRs.UID,
						Controller: pointer.BoolPtr(true),
					},
				},
			},
			Spec: currentRs.Spec.Template.Spec,
		}
		Expect(k8sClient.Create(context.Background(), currentRsPod)).Should(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.DeleteAllOf(context.Background(), &appsv1.ReplicaSet{}, client.InNamespace(instance.Namespace))).Should(Succeed())
		Expect(k8sClient.DeleteAllOf(context.Background(), &appsv1.StatefulSet{}, client.InNamespace(instance.Namespace))).Should(Succeed())
		Expect(k8sClient.DeleteAllOf(context.Background(), &appsv2beta1.EMQX{}, client.InNamespace(instance.Namespace))).Should(Succeed())
		Expect(k8sClient.Delete(context.Background(), ns)).Should(Succeed())
	})

	It("running update emqx node controller", func() {
		Eventually(func() *appsv2beta1.EMQX {
			_ = s.reconcile(ctx, instance, fakeR)
			return instance
		}).WithTimeout(timeout).WithPolling(interval).Should(And(
			WithTransform(
				// should add pod deletion cost
				func(instance *appsv2beta1.EMQX) map[string]string {
					_ = k8sClient.Get(context.Background(), client.ObjectKeyFromObject(currentRsPod), currentRsPod)
					return currentRsPod.Annotations
				},
				HaveKeyWithValue("controller.kubernetes.io/pod-deletion-cost", "-99999"),
			),
			WithTransform(
				// should scale down rs
				func(instance *appsv2beta1.EMQX) int32 {
					_ = k8sClient.Get(context.Background(), client.ObjectKeyFromObject(currentRs), currentRs)
					return *currentRs.Spec.Replicas
				},
				Equal(int32(0)),
			),
			WithTransform(
				// before rs not ready, do nothing for sts
				func(instance *appsv2beta1.EMQX) int32 {
					_ = k8sClient.Get(context.Background(), client.ObjectKeyFromObject(currentSts), currentSts)
					return *currentSts.Spec.Replicas
				},
				Equal(int32(1)),
			),
		))

		By("mock rs ready, should scale down sts")
		instance.Status.ReplicantNodesStatus.CurrentRevision = instance.Status.ReplicantNodesStatus.UpdateRevision
		Eventually(func() *appsv2beta1.EMQX {
			_ = s.reconcile(ctx, instance, fakeR)
			return instance
		}).WithTimeout(timeout).WithPolling(interval).Should(
			WithTransform(
				func(instance *appsv2beta1.EMQX) int32 {
					_ = k8sClient.Get(context.Background(), client.ObjectKeyFromObject(currentSts), currentSts)
					return *currentSts.Spec.Replicas
				}, Equal(int32(0)),
			),
		)
	})

})

var _ = Describe("check can be scale down", func() {
	var s *syncPods
	var fakeR *innerReq.FakeRequester = &innerReq.FakeRequester{}
	var instance *appsv2beta1.EMQX = new(appsv2beta1.EMQX)
	var ns *corev1.Namespace = &corev1.Namespace{}

	BeforeEach(func() {
		s = &syncPods{emqxReconciler}
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "controller-v2beta1-update-emqx-nodes-test-" + rand.String(5),
				Labels: map[string]string{
					"test": "e2e",
				},
			},
		}
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Status.Conditions = []metav1.Condition{
			{
				Type:               appsv2beta1.Available,
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
			},
		}

		Expect(k8sClient.Create(context.TODO(), ns)).Should(Succeed())

	})

	AfterEach(func() {
		Expect(k8sClient.DeleteAllOf(context.Background(), &corev1.Pod{}, client.InNamespace(instance.Namespace))).Should(Succeed())
		Expect(k8sClient.DeleteAllOf(context.Background(), &appsv1.ReplicaSet{}, client.InNamespace(instance.Namespace))).Should(Succeed())
		Expect(k8sClient.DeleteAllOf(context.Background(), &appsv1.StatefulSet{}, client.InNamespace(instance.Namespace))).Should(Succeed())
		Expect(k8sClient.DeleteAllOf(context.Background(), &appsv2beta1.EMQX{}, client.InNamespace(instance.Namespace))).Should(Succeed())
		Expect(k8sClient.Delete(context.Background(), ns)).Should(Succeed())
	})

	Context("check can be scale down sts", func() {
		var oldSts *appsv1.StatefulSet
		JustBeforeEach(func() {
			oldSts = &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instance.Name + "-fake",
					Namespace: instance.Namespace,
				},
				Spec: appsv1.StatefulSetSpec{
					ServiceName: instance.Name + "-fake",
					Replicas:    pointer.Int32Ptr(1),
				},
			}

		})
		It("emqx is not available", func() {
			instance.Status.Conditions = []metav1.Condition{}
			canBeScaledDown, err := s.canBeScaleDownSts(ctx, instance, nil, oldSts, []string{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(canBeScaledDown).Should(BeFalse())
		})

		It("emqx is available, but is not initial delay seconds", func() {
			instance.Spec.UpdateStrategy.InitialDelaySeconds = 99999999
			canBeScaledDown, err := s.canBeScaleDownSts(ctx, instance, nil, oldSts, []string{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(canBeScaledDown).Should(BeFalse())
		})

		It("the replicaSet didn't ready", func() {
			instance.Spec.ReplicantTemplate = &appsv2beta1.EMQXReplicantTemplate{
				Spec: appsv2beta1.EMQXReplicantTemplateSpec{
					Replicas: pointer.Int32Ptr(3),
				},
			}
			instance.Status.ReplicantNodesStatus = &appsv2beta1.EMQXNodesStatus{
				UpdateRevision:  "update",
				CurrentRevision: "current",
			}

			canBeScaledDown, err := s.canBeScaleDownSts(ctx, instance, nil, oldSts, []string{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(canBeScaledDown).Should(BeFalse())
			Eventually(s.reconcile(ctx, instance, nil)).WithTimeout(timeout).WithPolling(interval).WithTimeout(timeout).WithPolling(interval).Should(Equal(subResult{}))
		})

		It("emqx is enterprise, and node session more than 0", func() {
			fakeR.ReqFunc = func(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error) {
				resp = &http.Response{
					StatusCode: 200,
				}

				if method == "GET" {
					respBody, _ = json.Marshal(&appsv2beta1.EMQXNode{
						Edition: "Enterprise",
						Session: 99999,
					})
				}

				return resp, respBody, nil
			}

			canBeScaledDown, err := s.canBeScaleDownSts(ctx, instance, fakeR, oldSts, []string{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(canBeScaledDown).Should(BeFalse())
		})

		It("emqx is enterprise, and node session is 0", func() {
			fakeR.ReqFunc = func(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error) {
				resp = &http.Response{
					StatusCode: 200,
				}
				respBody, _ = json.Marshal(&appsv2beta1.EMQXNode{
					Edition: "Enterprise",
					Session: 0,
				})
				return resp, respBody, nil
			}

			canBeScaledDown, err := s.canBeScaleDownSts(ctx, instance, fakeR, oldSts, []string{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(canBeScaledDown).Should(BeTrue())
		})

		It("emqx is open source", func() {
			fakeR.ReqFunc = func(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error) {
				resp = &http.Response{
					StatusCode: 200,
				}
				respBody, _ = json.Marshal(&appsv2beta1.EMQXNode{
					Edition: "Opensource",
				})
				return resp, respBody, nil
			}

			canBeScaledDown, err := s.canBeScaleDownSts(ctx, instance, fakeR, oldSts, []string{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(canBeScaledDown).Should(BeTrue())
		})
	})

	Context("check can be scale down rs", func() {
		var oldRs *appsv1.ReplicaSet
		JustBeforeEach(func() {
			oldRs = &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: instance.Name + "-",
					Namespace:    instance.Namespace,
					Labels:       appsv2beta1.DefaultReplicantLabels(instance),
				},
				Spec: appsv1.ReplicaSetSpec{
					Replicas: pointer.Int32Ptr(1),
					Selector: &metav1.LabelSelector{
						MatchLabels: appsv2beta1.DefaultReplicantLabels(instance),
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: appsv2beta1.DefaultReplicantLabels(instance),
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "emqx", Image: "emqx"},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, oldRs)).Should(Succeed())
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(oldRs), oldRs)).Should(Succeed())

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: oldRs.Name + "-",
					Namespace:    oldRs.Namespace,
					Labels:       oldRs.Spec.Selector.MatchLabels,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "apps/v1",
							Kind:       "ReplicaSet",
							Name:       oldRs.Name,
							UID:        oldRs.UID,
							Controller: pointer.BoolPtr(true),
						},
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "emqx", Image: "emqx"},
					},
				},
			}
			Expect(k8sClient.Create(ctx, pod)).Should(Succeed())
		})
		It("emqx is not available", func() {
			instance.Status.Conditions = []metav1.Condition{}
			canBeScaledDown, err := s.canBeScaleDownRs(ctx, instance, nil, oldRs, []string{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(canBeScaledDown).Should(BeNil())
		})

		It("emqx is available, but is not initial delay seconds", func() {
			instance.Spec.UpdateStrategy.InitialDelaySeconds = 99999999
			canBeScaledDown, err := s.canBeScaleDownRs(ctx, instance, nil, oldRs, []string{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(canBeScaledDown).Should(BeNil())
		})

		It("emqx is in node evacuations", func() {
			instance.Status.NodeEvacuationsStatus = []appsv2beta1.NodeEvacuationStatus{
				{
					State: "fake",
				},
			}
			canBeScaledDown, err := s.canBeScaleDownRs(ctx, instance, fakeR, oldRs, []string{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(canBeScaledDown).Should(BeNil())
		})

		It("emqx is enterprise, and node session more than 0", func() {
			fakeR.ReqFunc = func(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error) {
				resp = &http.Response{
					StatusCode: 200,
				}

				if method == "GET" {
					respBody, _ = json.Marshal(&appsv2beta1.EMQXNode{
						Edition: "Enterprise",
						Session: 99999,
					})
				}

				return resp, respBody, nil
			}

			canBeScaledDown, err := s.canBeScaleDownRs(ctx, instance, fakeR, oldRs, []string{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(canBeScaledDown).Should(BeNil())
		})

		It("emqx is enterprise, and node session is 0", func() {
			fakeR.ReqFunc = func(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error) {
				resp = &http.Response{
					StatusCode: 200,
				}
				respBody, _ = json.Marshal(&appsv2beta1.EMQXNode{
					Edition: "Enterprise",
					Session: 0,
				})
				return resp, respBody, nil
			}

			Eventually(func() *corev1.Pod {
				canBeScaledDown, err := s.canBeScaleDownRs(ctx, instance, fakeR, oldRs, []string{})
				if err != nil {
					return nil
				}
				return canBeScaledDown
			}).WithTimeout(timeout).WithPolling(interval).ShouldNot(BeNil())
		})

		It("emqx is open source", func() {
			fakeR.ReqFunc = func(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error) {
				resp = &http.Response{
					StatusCode: 200,
				}
				respBody, _ = json.Marshal(&appsv2beta1.EMQXNode{
					Edition: "Opensource",
				})
				return resp, respBody, nil
			}
			Eventually(func() *corev1.Pod {
				canBeScaledDown, err := s.canBeScaleDownRs(ctx, instance, fakeR, oldRs, []string{})
				if err != nil {
					return nil
				}
				return canBeScaledDown
			}).WithTimeout(timeout).WithPolling(interval).ShouldNot(BeNil())
		})
	})
})
