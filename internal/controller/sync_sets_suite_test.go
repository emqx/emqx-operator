package controller

import (
	"fmt"
	"time"

	appsv2beta1 "github.com/emqx/emqx-operator/api/v2beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Check sync sts and pvc", func() {
	var s *syncSets

	var instance *appsv2beta1.EMQX = new(appsv2beta1.EMQX)
	var ns *corev1.Namespace = &corev1.Namespace{}

	BeforeEach(func() {
		s = &syncSets{emqxReconciler}
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "controller-v2beta1-sync-sets-test-" + rand.String(5),
				Labels: map[string]string{
					"test": "e2e",
				},
			},
		}
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Spec.RevisionHistoryLimit = ptr.To(int32(3))
		instance.Status = appsv2beta1.EMQXStatus{
			Conditions: []metav1.Condition{
				{
					Type:               appsv2beta1.Ready,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
				},
			},
		}

		Expect(k8sClient.Create(ctx, ns)).To(Succeed())
		for i := 0; i < 5; i++ {
			name := fmt.Sprintf("%s-%d", instance.Name, i)

			rs := &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: instance.Namespace,
					Labels: appsv2beta1.CloneAndAddLabel(
						appsv2beta1.DefaultReplicantLabels(instance),
						appsv2beta1.LabelsPodTemplateHashKey,
						fmt.Sprintf("fake-%d", i),
					),
				},
				Spec: appsv1.ReplicaSetSpec{
					Replicas: ptr.To(int32(0)),
					Selector: &metav1.LabelSelector{
						MatchLabels: appsv2beta1.CloneAndAddLabel(
							appsv2beta1.DefaultReplicantLabels(instance),
							appsv2beta1.LabelsPodTemplateHashKey,
							fmt.Sprintf("fake-%d", i),
						),
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: appsv2beta1.CloneAndAddLabel(
								appsv2beta1.DefaultReplicantLabels(instance),
								appsv2beta1.LabelsPodTemplateHashKey,
								fmt.Sprintf("fake-%d", i),
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
			Expect(k8sClient.Status().Patch(ctx, rs.DeepCopy(), client.Merge)).Should(Succeed())

			sts := &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: instance.Namespace,
					Labels: appsv2beta1.CloneAndAddLabel(
						appsv2beta1.DefaultCoreLabels(instance),
						appsv2beta1.LabelsPodTemplateHashKey,
						fmt.Sprintf("fake-%d", i),
					),
				},
				Spec: appsv1.StatefulSetSpec{
					Replicas: ptr.To(int32(0)),
					Selector: &metav1.LabelSelector{
						MatchLabels: appsv2beta1.CloneAndAddLabel(
							appsv2beta1.DefaultCoreLabels(instance),
							appsv2beta1.LabelsPodTemplateHashKey,
							fmt.Sprintf("fake-%d", i),
						),
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: appsv2beta1.CloneAndAddLabel(
								appsv2beta1.DefaultCoreLabels(instance),
								appsv2beta1.LabelsPodTemplateHashKey,
								fmt.Sprintf("fake-%d", i),
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
			Expect(k8sClient.Create(ctx, sts.DeepCopy())).Should(Succeed())
			sts.Status.Replicas = 0
			sts.Status.ObservedGeneration = 1
			Expect(k8sClient.Status().Patch(ctx, sts.DeepCopy(), client.Merge)).Should(Succeed())

			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sts.Name,
					Namespace: sts.Namespace,
					Labels:    sts.Labels,
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, pvc.DeepCopy())).Should(Succeed())
		}
	})

	It("should delete rs sts and pvc", func() {
		Expect(s.reconcile(ctx, logger, instance, nil)).Should(Equal(subResult{}))

		Eventually(func() int {
			list := &appsv1.ReplicaSetList{}
			_ = k8sClient.List(ctx, list,
				client.InNamespace(instance.Namespace),
				client.MatchingLabels(appsv2beta1.DefaultReplicantLabels(instance)),
			)
			count := 0
			for _, i := range list.Items {
				item := i.DeepCopy()
				if item.DeletionTimestamp == nil {
					count++
				}
			}
			return count
		}).WithTimeout(timeout).WithPolling(interval).Should(BeEquivalentTo(*instance.Spec.RevisionHistoryLimit))

		Eventually(func() int {
			list := &appsv1.StatefulSetList{}
			_ = k8sClient.List(ctx, list,
				client.InNamespace(instance.Namespace),
				client.MatchingLabels(appsv2beta1.DefaultCoreLabels(instance)),
			)
			count := 0
			for _, i := range list.Items {
				item := i.DeepCopy()
				if item.DeletionTimestamp == nil {
					count++
				}
			}
			return count
		}).WithTimeout(timeout).WithPolling(interval).Should(BeEquivalentTo(*instance.Spec.RevisionHistoryLimit))

		Eventually(func() int {
			list := &corev1.PersistentVolumeClaimList{}
			_ = k8sClient.List(ctx, list,
				client.InNamespace(instance.Namespace),
				client.MatchingLabels(appsv2beta1.DefaultCoreLabels(instance)),
			)
			count := 0
			for _, i := range list.Items {
				item := i.DeepCopy()
				if item.DeletionTimestamp == nil {
					count++
				}
			}
			return count
		}).WithTimeout(timeout).WithPolling(interval).Should(BeEquivalentTo(*instance.Spec.RevisionHistoryLimit))
	})
})
