package controller

import (
	"fmt"
	"time"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = DescribeClientFaultMatrix("Reconciler cleanupOutdatedSets", Ordered, func() {
	var instance *crd.EMQX
	var ns *corev1.Namespace

	BeforeAll(func() {
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "controller-cleanup-outdated-test",
				Labels:       map[string]string{"test": "e2e"},
			},
		}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, ns)).To(Succeed())
	})

	BeforeEach(func() {
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Spec.RevisionHistoryLimit = 3
		instance.Status = crd.EMQXStatus{
			Conditions: []metav1.Condition{
				{
					Type:               crd.Ready,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
				},
			},
		}
	})

	It("should delete outdated replicant sets", func() {
		numReplicaSets := 5
		for i := 0; i < numReplicaSets; i++ {
			name := fmt.Sprintf("%s-%d", instance.Name, i)
			rs := &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: instance.Namespace,
					Labels: instance.DefaultLabelsWith(
						crd.ReplicantLabels(),
						map[string]string{crd.LabelPodTemplateHash: fmt.Sprintf("fake-%d", i)},
					),
				},
				Spec: appsv1.ReplicaSetSpec{
					Replicas: ptr.To(int32(0)),
					Selector: &metav1.LabelSelector{
						MatchLabels: instance.DefaultLabelsWith(
							crd.ReplicantLabels(),
							map[string]string{crd.LabelPodTemplateHash: fmt.Sprintf("fake-%d", i)},
						),
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: instance.DefaultLabelsWith(
								crd.ReplicantLabels(),
								map[string]string{crd.LabelPodTemplateHash: fmt.Sprintf("fake-%d", i)},
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
			Expect(k8sClient.Create(ctx, rs.DeepCopy())).Should(Succeed())
			rs.Status.Replicas = 0
			rs.Status.ObservedGeneration = 1
			Expect(k8sClient.Status().Update(ctx, rs)).Should(Succeed())
		}

		s := &cleanupOutdatedSets{emqxReconciler()}
		round := newReconcileRound()
		Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
		Eventually(s.reconcile).WithArguments(round, instance).
			Should(BeSuccessfulReconcile())

		Eventually(func() *appsv1.ReplicaSetList {
			list := &appsv1.ReplicaSetList{}
			_ = k8sClient.List(ctx, list,
				client.InNamespace(instance.Namespace),
				client.MatchingLabels(instance.DefaultLabelsWith(crd.ReplicantLabels())),
			)
			return list
		}).
			Should(HaveField("Items", HaveLen(int(instance.Spec.RevisionHistoryLimit))))
	})
})
