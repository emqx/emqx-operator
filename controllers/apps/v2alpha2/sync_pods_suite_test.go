package v2alpha2

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
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

var _ = Describe("Check update emqx nodes controller", Ordered, Label("node"), func() {
	var s *syncPods
	var fakeR *innerReq.FakeRequester = &innerReq.FakeRequester{}
	var instance *appsv2alpha2.EMQX = new(appsv2alpha2.EMQX)
	var ns *corev1.Namespace = &corev1.Namespace{}

	var currentSts, storageSts *appsv1.StatefulSet
	var currentRs, storageRs *appsv1.ReplicaSet
	var storageRsPod *corev1.Pod

	BeforeEach(func() {
		fakeR.ReqFunc = func(method, path string, body []byte, otps ...innerReq.HeaderOpt) (resp *http.Response, respBody []byte, err error) {
			resp = &http.Response{
				StatusCode: 200,
			}
			respBody, _ = json.Marshal(&appsv2alpha2.EMQXNode{
				Edition: "Opensource",
			})
			return resp, respBody, nil
		}

		s = &syncPods{emqxReconciler}
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "controller-v2alpha2-update-emqx-nodes-test-" + rand.String(5),
				Labels: map[string]string{
					"test": "e2e",
				},
			},
		}
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Spec.CoreTemplate.Labels = instance.Labels
		instance.Spec.ReplicantTemplate = &appsv2alpha2.EMQXReplicantTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Labels: instance.Labels,
			},
			Spec: appsv2alpha2.EMQXReplicantTemplateSpec{
				Replicas: pointer.Int32Ptr(1),
			},
		}
		instance.Status = appsv2alpha2.EMQXStatus{
			Conditions: []metav1.Condition{
				{
					Type:               appsv2alpha2.Available,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
				},
			},
			CoreNodesStatus: appsv2alpha2.EMQXNodesStatus{
				CurrentRevision: "current",
				ReadyReplicas:   2,
				Replicas:        1,
			},
			ReplicantNodesStatus: &appsv2alpha2.EMQXNodesStatus{
				CurrentRevision: "current",
				ReadyReplicas:   2,
				Replicas:        1,
			},
		}

		Expect(k8sClient.Create(context.TODO(), ns)).Should(Succeed())

		currentSts = &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: instance.Name + "-",
				Namespace:    instance.Namespace,
				Labels: appsv2alpha2.CloneAndAddLabel(
					instance.Labels,
					appsv2alpha2.PodTemplateHashLabelKey,
					"current",
				),
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: pointer.Int32Ptr(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: appsv2alpha2.CloneAndAddLabel(
						instance.Labels,
						appsv2alpha2.PodTemplateHashLabelKey,
						"current",
					),
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: appsv2alpha2.CloneAndAddLabel(
							instance.Labels,
							appsv2alpha2.PodTemplateHashLabelKey,
							"current",
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

		storageSts = currentSts.DeepCopy()
		storageSts.Labels[appsv2alpha2.PodTemplateHashLabelKey] = "storage"
		storageSts.Spec.Selector.MatchLabels[appsv2alpha2.PodTemplateHashLabelKey] = "storage"
		storageSts.Spec.Template.Labels[appsv2alpha2.PodTemplateHashLabelKey] = "storage"

		Expect(k8sClient.Create(context.Background(), currentSts)).Should(Succeed())
		currentSts.Status.Replicas = 1
		currentSts.Status.ReadyReplicas = 1
		Expect(k8sClient.Status().Update(context.Background(), currentSts)).Should(Succeed())

		Expect(k8sClient.Create(context.Background(), storageSts)).Should(Succeed())
		storageSts.Status.Replicas = 1
		storageSts.Status.ReadyReplicas = 1
		Expect(k8sClient.Status().Update(context.Background(), storageSts)).Should(Succeed())

		currentRs = &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: instance.Name + "-",
				Namespace:    instance.Namespace,
				Labels: appsv2alpha2.CloneAndAddLabel(
					instance.Labels,
					appsv2alpha2.PodTemplateHashLabelKey,
					"current",
				),
			},
			Spec: appsv1.ReplicaSetSpec{
				Replicas: pointer.Int32Ptr(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: appsv2alpha2.CloneAndAddLabel(
						instance.Labels,
						appsv2alpha2.PodTemplateHashLabelKey,
						"current",
					),
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: appsv2alpha2.CloneAndAddLabel(
							instance.Labels,
							appsv2alpha2.PodTemplateHashLabelKey,
							"current",
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

		storageRs = currentRs.DeepCopy()
		storageRs.Labels[appsv2alpha2.PodTemplateHashLabelKey] = "storage"
		storageRs.Spec.Selector.MatchLabels[appsv2alpha2.PodTemplateHashLabelKey] = "storage"
		storageRs.Spec.Template.Labels[appsv2alpha2.PodTemplateHashLabelKey] = "storage"

		Expect(k8sClient.Create(context.Background(), currentRs)).Should(Succeed())
		currentRs.Status.Replicas = 1
		currentRs.Status.ReadyReplicas = 1
		Expect(k8sClient.Status().Update(context.Background(), currentRs)).Should(Succeed())

		Expect(k8sClient.Create(context.Background(), storageRs)).Should(Succeed())
		storageRs.Status.Replicas = 1
		storageRs.Status.ReadyReplicas = 1
		Expect(k8sClient.Status().Update(context.Background(), storageRs)).Should(Succeed())

		storageRsPod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: storageRs.Name + "-",
				Namespace:    storageRs.Namespace,
				Labels:       storageRs.Spec.Template.Labels,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "apps/v1",
						Kind:       "ReplicaSet",
						Name:       storageRs.Name,
						UID:        storageRs.UID,
						Controller: pointer.BoolPtr(true),
					},
				},
			},
			Spec: storageRs.Spec.Template.Spec,
		}
		Expect(k8sClient.Create(context.Background(), storageRsPod)).Should(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.DeleteAllOf(context.Background(), &appsv1.ReplicaSet{}, client.InNamespace(instance.Namespace))).Should(Succeed())
		Expect(k8sClient.DeleteAllOf(context.Background(), &appsv1.StatefulSet{}, client.InNamespace(instance.Namespace))).Should(Succeed())
		Expect(k8sClient.DeleteAllOf(context.Background(), &appsv2alpha2.EMQX{}, client.InNamespace(instance.Namespace))).Should(Succeed())
		Expect(k8sClient.Delete(context.Background(), ns)).Should(Succeed())
	})

	It("running update emqx node controller", func() {
		Expect(s.reconcile(ctx, instance, fakeR)).Should(Equal(subResult{}))

		By("should add pod deletion cost annotation")
		Eventually(func() map[string]string {
			_ = k8sClient.Get(context.Background(), client.ObjectKeyFromObject(storageRsPod), storageRsPod)
			return storageRsPod.Annotations
		}).Should(HaveKeyWithValue("controller.kubernetes.io/pod-deletion-cost", "-99999"))

		By("should scale down rs")
		Eventually(func() int32 {
			_ = k8sClient.Get(context.Background(), client.ObjectKeyFromObject(storageRs), storageRs)
			return *storageRs.Spec.Replicas
		}).Should(Equal(int32(0)))

		By("before scale down rs, do nothing for sts")
		Eventually(func() int32 {
			_ = k8sClient.Get(context.Background(), client.ObjectKeyFromObject(storageSts), storageSts)
			return *storageSts.Spec.Replicas
		}).Should(Equal(int32(1)))

		instance.Status.ReplicantNodesStatus.ReadyReplicas = instance.Status.ReplicantNodesStatus.Replicas
		Expect(s.reconcile(ctx, instance, fakeR)).Should(Equal(subResult{}))
		By("should scale down sts")
		Eventually(func() int32 {
			_ = k8sClient.Get(context.Background(), client.ObjectKeyFromObject(storageSts), storageSts)
			return *storageSts.Spec.Replicas
		}).Should(Equal(int32(0)))
	})

})

var _ = Describe("check can be scale down", func() {
	var s *syncPods
	var fakeR *innerReq.FakeRequester = &innerReq.FakeRequester{}
	var instance *appsv2alpha2.EMQX = new(appsv2alpha2.EMQX)
	var ns *corev1.Namespace = &corev1.Namespace{}

	BeforeEach(func() {
		s = &syncPods{emqxReconciler}
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "controller-v2alpha2-update-emqx-nodes-test-" + rand.String(5),
				Labels: map[string]string{
					"test": "e2e",
				},
			},
		}
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Status.Conditions = []metav1.Condition{
			{
				Type:               appsv2alpha2.Available,
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
		Expect(k8sClient.DeleteAllOf(context.Background(), &appsv2alpha2.EMQX{}, client.InNamespace(instance.Namespace))).Should(Succeed())
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

		It("have more than 1 of replicaSet", func() {
			instance.Spec.ReplicantTemplate = &appsv2alpha2.EMQXReplicantTemplate{
				Spec: appsv2alpha2.EMQXReplicantTemplateSpec{
					Replicas: pointer.Int32Ptr(3),
				},
			}
			instance.Status.ReplicantNodesStatus = &appsv2alpha2.EMQXNodesStatus{
				ReadyReplicas: 6,
				Replicas:      3,
			}

			canBeScaledDown, err := s.canBeScaleDownSts(ctx, instance, nil, oldSts, []string{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(canBeScaledDown).Should(BeFalse())
			Eventually(s.reconcile(ctx, instance, nil)).Should(Equal(subResult{}))
		})

		It("emqx is enterprise, and node session more than 0", func() {
			fakeR.ReqFunc = func(method, path string, body []byte, opts ...innerReq.HeaderOpt) (resp *http.Response, respBody []byte, err error) {
				resp = &http.Response{
					StatusCode: 200,
				}

				if method == "GET" {
					respBody, _ = json.Marshal(&appsv2alpha2.EMQXNode{
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
			fakeR.ReqFunc = func(method, path string, body []byte, opts ...innerReq.HeaderOpt) (resp *http.Response, respBody []byte, err error) {
				resp = &http.Response{
					StatusCode: 200,
				}
				respBody, _ = json.Marshal(&appsv2alpha2.EMQXNode{
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
			fakeR.ReqFunc = func(method, path string, body []byte, opts ...innerReq.HeaderOpt) (resp *http.Response, respBody []byte, err error) {
				resp = &http.Response{
					StatusCode: 200,
				}
				respBody, _ = json.Marshal(&appsv2alpha2.EMQXNode{
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
					Labels:       instance.Labels,
				},
				Spec: appsv1.ReplicaSetSpec{
					Replicas: pointer.Int32Ptr(1),
					Selector: &metav1.LabelSelector{
						MatchLabels: instance.Labels,
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: instance.Labels,
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

		It("emqx is enterprise, and node session more than 0", func() {
			fakeR.ReqFunc = func(method, path string, body []byte, opts ...innerReq.HeaderOpt) (resp *http.Response, respBody []byte, err error) {
				resp = &http.Response{
					StatusCode: 200,
				}

				if method == "GET" {
					respBody, _ = json.Marshal(&appsv2alpha2.EMQXNode{
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
			fakeR.ReqFunc = func(method, path string, body []byte, opts ...innerReq.HeaderOpt) (resp *http.Response, respBody []byte, err error) {
				resp = &http.Response{
					StatusCode: 200,
				}
				respBody, _ = json.Marshal(&appsv2alpha2.EMQXNode{
					Edition: "Enterprise",
					Session: 0,
				})
				return resp, respBody, nil
			}

			canBeScaledDown, err := s.canBeScaleDownRs(ctx, instance, fakeR, oldRs, []string{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(canBeScaledDown).ShouldNot(BeNil())
		})

		It("emqx is open source", func() {
			fakeR.ReqFunc = func(method, path string, body []byte, opts ...innerReq.HeaderOpt) (resp *http.Response, respBody []byte, err error) {
				resp = &http.Response{
					StatusCode: 200,
				}
				respBody, _ = json.Marshal(&appsv2alpha2.EMQXNode{
					Edition: "Opensource",
				})
				return resp, respBody, nil
			}

			canBeScaledDown, err := s.canBeScaleDownRs(ctx, instance, fakeR, oldRs, []string{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(canBeScaledDown).ShouldNot(BeNil())
		})
	})
})
