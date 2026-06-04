package controller

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	crdv2 "github.com/emqx/emqx-operator/api/v2"
	req "github.com/emqx/emqx-operator/internal/requester"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Reconciler syncClusterMembership", Ordered, func() {
	var ns *corev1.Namespace
	var instance *crdv2.EMQX
	var round *reconcileRound
	var coreSetUpdate, coreSetCurrent *appsv1.StatefulSet
	var replicantSet *appsv1.ReplicaSet
	var corePod0, corePod1 *corev1.Pod
	var replicantPod *corev1.Pod
	var forceLeftNodes []string

	emqxNodeName := func(podName string) string {
		return constructNodeName(podName, instance)
	}

	BeforeAll(func() {
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "controller-sync-cluster-test-",
				Labels:       map[string]string{"test": "e2e"},
			},
		}
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})

	mkCorePod := func(sts *appsv1.StatefulSet, ordinal int) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sts.Name + "-" + strconv.Itoa(ordinal),
				Namespace: sts.Namespace,
				Labels:    sts.Spec.Template.Labels,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "apps/v1",
						Kind:       "StatefulSet",
						Name:       sts.Name,
						UID:        sts.UID,
						Controller: ptr.To(true),
					},
				},
			},
			Spec: sts.Spec.Template.Spec,
		}
	}

	BeforeEach(func() {
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Spec.ClusterDomain = "local"
		instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(1))
		instance.Spec.ReplicantTemplate = &crdv2.EMQXReplicantTemplate{}
		instance.Spec.ReplicantTemplate.Spec.Replicas = ptr.To(int32(1))
		instance.Status.CoreNodesStatus.CurrentRevision = currentRevision
		instance.Status.CoreNodesStatus.UpdateRevision = updateRevision
		instance.Status.ReplicantNodesStatus.CurrentRevision = updateRevision
		instance.Status.ReplicantNodesStatus.UpdateRevision = updateRevision

		coreLabels := instance.DefaultLabelsWith(
			crdv2.CoreLabels(),
			map[string]string{crdv2.LabelPodTemplateHash: updateRevision},
		)
		coreLabelsCurrent := instance.DefaultLabelsWith(
			crdv2.CoreLabels(),
			map[string]string{crdv2.LabelPodTemplateHash: currentRevision},
		)
		coreSetUpdate = &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      instance.Name + "-core-" + updateRevision,
				Namespace: ns.Name,
				Labels:    coreLabels,
			},
			Spec: appsv1.StatefulSetSpec{
				ServiceName: instance.HeadlessServiceNamespacedName().Name,
				Replicas:    ptr.To(int32(1)),
				Selector:    &metav1.LabelSelector{MatchLabels: coreLabels},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: coreLabels},
					Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "emqx", Image: "emqx"}}},
				},
			},
		}
		coreSetCurrent = &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      instance.Name + "-core-" + currentRevision,
				Namespace: ns.Name,
				Labels:    coreLabelsCurrent,
			},
			Spec: appsv1.StatefulSetSpec{
				ServiceName: instance.HeadlessServiceNamespacedName().Name,
				Replicas:    ptr.To(int32(0)),
				Selector:    &metav1.LabelSelector{MatchLabels: coreLabelsCurrent},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: coreLabelsCurrent},
					Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "emqx", Image: "emqx"}}},
				},
			},
		}
		Expect(k8sClient.Create(ctx, coreSetUpdate)).Should(Succeed())
		Expect(k8sClient.Create(ctx, coreSetCurrent)).Should(Succeed())

		corePod0 = mkCorePod(coreSetUpdate, 0)
		Expect(k8sClient.Create(ctx, corePod0)).Should(Succeed())
		corePod1 = mkCorePod(coreSetUpdate, 1)
		Expect(k8sClient.Create(ctx, corePod1)).Should(Succeed())

		replicantLabels := instance.DefaultLabelsWith(
			crdv2.ReplicantLabels(),
			map[string]string{crdv2.LabelPodTemplateHash: updateRevision},
		)
		replicantSet = &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      instance.Name + "-replicant-" + updateRevision,
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
				Name:      replicantSet.Name + "-xyz",
				Namespace: ns.Name,
				Labels:    replicantLabels,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "apps/v1",
						Kind:       "ReplicaSet",
						Name:       replicantSet.Name,
						UID:        replicantSet.UID,
						Controller: ptr.To(true),
					},
				},
			},
			Spec: replicantSet.Spec.Template.Spec,
		}
		Expect(k8sClient.Create(ctx, replicantPod)).Should(Succeed())

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
		round = newReconcileRoundWithRequester(mockRequester)
	})

	AfterEach(func() {
		Expect(k8sClient.DeleteAllOf(ctx, &corev1.Pod{}, client.InNamespace(ns.Name))).Should(Succeed())
		Expect(k8sClient.DeleteAllOf(ctx, &appsv1.ReplicaSet{}, client.InNamespace(ns.Name))).Should(Succeed())
		Expect(k8sClient.DeleteAllOf(ctx, &appsv1.StatefulSet{}, client.InNamespace(ns.Name))).Should(Succeed())
	})

	It("force-leaves scaled-down core nodes whose pods are gone", func() {
		instance.Status.CoreNodes = []crdv2.EMQXNode{
			{Name: emqxNodeName(corePod0.Name), PodName: corePod0.Name, Status: "running", Role: "core"},
			{Name: emqxNodeName(corePod1.Name), PodName: corePod1.Name, Status: "stopped", Role: "core"},
			{Name: emqxNodeName(coreSetUpdate.Name + "-10"), PodName: "", Status: "stopped", Role: "core"},
			{Name: emqxNodeName(coreSetUpdate.Name + "-11"), PodName: "", Status: "stopped", Role: "core"},
		}
		s := &syncClusterMembership{emqxReconciler}
		round.state = loadReconcileState(ctx, k8sClient, instance)
		Expect(s.reconcile(round, instance)).Should(Equal(subResult{}))
		Expect(forceLeftNodes).To(ConsistOf(
			emqxNodeName(coreSetUpdate.Name+"-10"),
			emqxNodeName(coreSetUpdate.Name+"-11"),
		))
	})

	It("does NOT force-leave stopped core node whose pod still exists", func() {
		instance.Status.CoreNodes = []crdv2.EMQXNode{
			{Name: emqxNodeName(corePod0.Name), PodName: corePod0.Name, Status: "stopped", Role: "core"},
			{Name: emqxNodeName(corePod1.Name), PodName: corePod1.Name, Status: "stopped", Role: "core"},
		}
		s := &syncClusterMembership{emqxReconciler}
		round.state = loadReconcileState(ctx, k8sClient, instance)
		Expect(s.reconcile(round, instance)).Should(Equal(subResult{}))
		Expect(forceLeftNodes).To(BeEmpty())
	})

	It("does NOT force-leave stopped core node whose ordinal is still desired", func() {
		Expect(k8sClient.Delete(ctx, corePod0)).Should(Succeed())
		instance.Status.CoreNodes = []crdv2.EMQXNode{
			{Name: emqxNodeName(corePod0.Name), PodName: "", Status: "stopped", Role: "core"},
		}
		s := &syncClusterMembership{emqxReconciler}
		round.state = loadReconcileState(ctx, k8sClient, instance)
		Expect(s.reconcile(round, instance)).Should(Equal(subResult{}))
		Expect(forceLeftNodes).To(BeEmpty())
	})

	It("force-leaves stopped old current core node after it is scaled to zero", func() {
		instance.Status.CoreNodes = []crdv2.EMQXNode{
			{Name: emqxNodeName(corePod0.Name), PodName: corePod0.Name, Status: "running", Role: "core"},
			{Name: emqxNodeName(corePod1.Name), PodName: corePod1.Name, Status: "stopped", Role: "core"},
			{Name: emqxNodeName(coreSetCurrent.Name + "-0"), PodName: "", Status: "stopped", Role: "core"},
		}
		s := &syncClusterMembership{emqxReconciler}
		round.state = loadReconcileState(ctx, k8sClient, instance)
		Expect(s.reconcile(round, instance)).Should(Equal(subResult{}))
		Expect(forceLeftNodes).To(ConsistOf(emqxNodeName(coreSetCurrent.Name + "-0")))
	})

	It("force-leaves core nodes belonging to removed coreSet", func() {
		instance.Status.CoreNodes = []crdv2.EMQXNode{
			{Name: emqxNodeName(corePod0.Name), PodName: corePod0.Name, Status: "running", Role: "core"},
			{Name: emqxNodeName(corePod1.Name), PodName: corePod1.Name, Status: "stopped", Role: "core"},
			{Name: emqxNodeName("emqx-core-ancient-0"), PodName: "", Status: "stopped", Role: "core"},
		}
		s := &syncClusterMembership{emqxReconciler}
		round.state = loadReconcileState(ctx, k8sClient, instance)
		Expect(s.reconcile(round, instance)).Should(Equal(subResult{}))
		Expect(forceLeftNodes).To(ConsistOf(emqxNodeName("emqx-core-ancient-0")))
	})

	It("does NOT force-leave running core nodes without pods", func() {
		instance.Status.CoreNodes = []crdv2.EMQXNode{
			{Name: emqxNodeName(corePod0.Name), PodName: corePod0.Name, Status: "running", Role: "core"},
			{Name: emqxNodeName(corePod1.Name), PodName: "", Status: "running", Role: "core"},
		}
		s := &syncClusterMembership{emqxReconciler}
		round.state = loadReconcileState(ctx, k8sClient, instance)
		Expect(s.reconcile(round, instance)).Should(Equal(subResult{}))
		Expect(forceLeftNodes).To(BeEmpty())
	})
})
