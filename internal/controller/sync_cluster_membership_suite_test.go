package controller

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	crdv2 "github.com/emqx/emqx-operator/api/v2"
	req "github.com/emqx/emqx-operator/internal/requester"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

var _ = Describe("Reconciler syncClusterMembership", Ordered, func() {
	var ns *corev1.Namespace
	var instance *crdv2.EMQX
	var round *reconcileRound
	var coreSet *appsv1.StatefulSet
	var replicantSet *appsv1.ReplicaSet
	var corePod0 *corev1.Pod
	var replicantPod *corev1.Pod

	var forceLeftNodes []string

	const updateRevision = "update"

	emqxNodeName := func(podName string) string {
		return fmt.Sprintf("emqx@%s.%s.%s.svc.%s",
			podName,
			instance.HeadlessServiceNamespacedName().Name,
			instance.Namespace,
			instance.Spec.ClusterDomain)
	}

	BeforeAll(func() {
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "controller-sync-cluster-test",
				Labels: map[string]string{"test": "e2e"},
			},
		}
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})

	BeforeEach(func() {
		coreLabels := emqx.DefaultLabelsWith(crdv2.CoreLabels())
		coreSet = &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      emqx.Name + "-core",
				Namespace: ns.Name,
				Labels:    coreLabels,
			},
			Spec: appsv1.StatefulSetSpec{
				ServiceName: emqx.Name + "-core",
				Replicas:    ptr.To(int32(1)),
				UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
					Type: appsv1.OnDeleteStatefulSetStrategyType,
				},
				Selector: &metav1.LabelSelector{
					MatchLabels: coreLabels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: coreLabels},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "emqx", Image: "emqx"}},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, coreSet)).Should(Succeed())

		corePod0 = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      coreSet.Name + "-0",
				Namespace: ns.Name,
				Labels: map[string]string{
					crdv2.LabelInstance:                   "emqx",
					crdv2.LabelManagedBy:                  "emqx-operator",
					crdv2.LabelDBRole:                     "core",
					appsv1.ControllerRevisionHashLabelKey: updateRevision,
				},
				OwnerReferences: ownerReferences(coreSet),
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "emqx", Image: "emqx"}},
			},
		}
		Expect(k8sClient.Create(ctx, corePod0)).Should(Succeed())

		replicantLabels := emqx.DefaultLabelsWith(
			crdv2.ReplicantLabels(),
			map[string]string{crdv2.LabelPodTemplateHash: "rev1"},
		)
		replicantSet = &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      emqx.Name + "-replicant-rev1",
				Namespace: ns.Name,
				Labels:    replicantLabels,
			},
			Spec: appsv1.ReplicaSetSpec{
				Replicas: ptr.To(int32(1)),
				Selector: &metav1.LabelSelector{MatchLabels: replicantLabels},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: replicantLabels},
					Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "emqx", Image: "emqx"}}},
				},
			},
		}
		Expect(k8sClient.Create(ctx, replicantSet)).Should(Succeed())
		replicantPod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:            replicantSet.Name + "-xyz",
				Namespace:       ns.Name,
				Labels:          replicantLabels,
				OwnerReferences: ownerReferences(replicantSet),
			},
			Spec: replicantSet.Spec.Template.Spec,
		}
		Expect(k8sClient.Create(ctx, replicantPod)).Should(Succeed())

		replicantPod.Status.PodIP = "10.0.0.1"
		Expect(k8sClient.Status().Update(ctx, replicantPod)).Should(Succeed())

		coreSet.Status.Replicas = 1
		coreSet.Status.ReadyReplicas = 1
		coreSet.Status.CurrentRevision = updateRevision
		coreSet.Status.UpdateRevision = updateRevision
		Expect(k8sClient.Status().Update(ctx, coreSet)).Should(Succeed())

		replicantSet.Status.Replicas = 1
		replicantSet.Status.ReadyReplicas = 1
		Expect(k8sClient.Status().Update(ctx, replicantSet)).Should(Succeed())

		forceLeftNodes = nil
		mockRequester := req.NewMockRequester(
			func(method string, u url.URL, body []byte, header http.Header) (*http.Response, []byte, error) {
				if method == "DELETE" && strings.Contains(u.Path, "force_leave") {
					parts := strings.Split(u.Path, "/")
					forceLeftNodes = append(forceLeftNodes, parts[len(parts)-2])
					return &http.Response{StatusCode: 204}, nil, nil
				}
				return &http.Response{StatusCode: 501}, nil, nil
			},
		)

		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Spec.ClusterDomain = "local"
		instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(1))
		instance.Spec.ReplicantTemplate = &crdv2.EMQXReplicantTemplate{}
		instance.Spec.ReplicantTemplate.Spec.Replicas = ptr.To(int32(1))
		round = newReconcileRoundWithRequester(mockRequester)
		Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
	})

	AfterEach(func() {
		_ = k8sClient.Delete(ctx, corePod0)
		_ = k8sClient.Delete(ctx, replicantPod)
		Expect(k8sClient.Delete(ctx, coreSet)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, replicantSet)).Should(Succeed())
	})

	It("force-leaves scaled-down node whose pod is gone", func() {
		instance.Status.CoreNodes = []crdv2.EMQXNode{
			{Name: emqxNodeName(corePod0.Name), PodName: corePod0.Name, Status: "running"},
			{Name: emqxNodeName(coreSet.Name + "-1"), PodName: "", Status: "stopped"},
			{Name: emqxNodeName(coreSet.Name + "-10"), PodName: "", Status: "stopped"},
		}
		s := &syncClusterMembership{emqxReconciler}
		Expect(s.reconcile(round, instance)).Should(Equal(subResult{}))
		Expect(forceLeftNodes).To(ConsistOf(
			emqxNodeName(coreSet.Name+"-1"),
			emqxNodeName(coreSet.Name+"-10"),
		))
	})

	It("does NOT force-leave stopped node whose pod still exists", func() {
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Status.CoreNodes = []crdv2.EMQXNode{
			{Name: emqxNodeName(corePod0.Name), PodName: corePod0.Name, Status: "stopped"},
			{Name: emqxNodeName(coreSet.Name + "-1"), PodName: coreSet.Name + "-1", Status: "stopped"},
		}
		Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
		s := &syncClusterMembership{emqxReconciler}
		Expect(s.reconcile(round, instance)).Should(Equal(subResult{}))
		Expect(forceLeftNodes).To(BeEmpty())
	})

	It("does NOT force-leave running nodes without pods", func() {
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(1))
		instance.Status.CoreNodes = []crdv2.EMQXNode{
			{Name: emqxNodeName(corePod0.Name), PodName: corePod0.Name, Status: "running"},
			{Name: emqxNodeName(coreSet.Name + "-1"), PodName: "", Status: "running"},
		}
		Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
		s := &syncClusterMembership{emqxReconciler}
		Expect(s.reconcile(round, instance)).Should(Equal(subResult{}))
		Expect(forceLeftNodes).To(BeEmpty())
	})

	It("force-leaves stale replicant nodes whose pods are gone", func() {
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(1))
		instance.Status.CoreNodes = []crdv2.EMQXNode{
			{Name: emqxNodeName(corePod0.Name), PodName: corePod0.Name, Status: "running"},
		}
		instance.Status.ReplicantNodes = []crdv2.EMQXNode{
			{Name: "emqx@10.0.0.1", PodName: replicantPod.Name, Status: "stopped", Role: "replicant"},
			{Name: "emqx@10.0.0.11", PodName: "", Status: "stopped", Role: "replicant"},
		}
		s := &syncClusterMembership{emqxReconciler}
		Expect(s.reconcile(round, instance)).Should(Equal(subResult{}))
		Expect(forceLeftNodes).To(ConsistOf("emqx@10.0.0.11"))
	})

	It("does NOT force-leave stopped replicant whose pod still exists", func() {
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(1))
		instance.Status.CoreNodes = []crdv2.EMQXNode{
			{Name: emqxNodeName(corePod0.Name), PodName: corePod0.Name, Status: "running"},
		}
		instance.Status.ReplicantNodes = []crdv2.EMQXNode{
			{Name: "emqx@10.0.0.1", PodName: replicantPod.Name, Status: "stopped", Role: "replicant"},
		}
		Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
		s := &syncClusterMembership{emqxReconciler}
		Expect(s.reconcile(round, instance)).Should(Equal(subResult{}))
		Expect(forceLeftNodes).To(BeEmpty())
	})

	It("does NOT force-leave running replicant nodes without pods", func() {
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(1))
		instance.Status.CoreNodes = []crdv2.EMQXNode{
			{Name: emqxNodeName(corePod0.Name), PodName: corePod0.Name, Status: "running"},
		}
		instance.Status.ReplicantNodes = []crdv2.EMQXNode{
			{Name: "emqx@10.0.0.99", PodName: "", Status: "running", Role: "replicant"},
		}
		Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
		s := &syncClusterMembership{emqxReconciler}
		Expect(s.reconcile(round, instance)).Should(Equal(subResult{}))
		Expect(forceLeftNodes).To(BeEmpty())
	})
})
