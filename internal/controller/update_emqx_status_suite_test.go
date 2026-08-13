package controller

import (
	"syscall"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	req "github.com/emqx/emqx-operator/internal/requester"
	. "github.com/emqx/emqx-operator/test/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = DescribeClientFaultMatrix("Reconciler updateStatus", Ordered, func() {
	var ns *corev1.Namespace
	var instance *crd.EMQX

	BeforeAll(func() {
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "controller-update-status-test",
				Labels:       map[string]string{"test": "e2e"},
			},
		}
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})

	BeforeEach(func() {
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Name = "emqx-update-status"
		Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
	})

	AfterEach(func() {
		_ = k8sClient.Delete(ctx, instance)
	})

	It("marks EMQX API unavailable when no core pod can serve API requests", func() {
		round := newReconcileRound()
		round.requester = &apiRequesterUnavailable{}
		s := &updateStatus{emqxReconciler()}

		Expect(runRoundReconcile(round, instance, s)).To(BeSuccessfulReconcile())

		Expect(actualObject(instance)).To(HaveCondition(crd.EMQXAPIAvailable, And(
			HaveField("Status", Equal(metav1.ConditionFalse)),
			HaveField("Reason", Equal("NoRequester")),
		)))
	})

	It("preserves unavailable transition time while API remains unavailable", func() {
		instance.Status.CoreNodes = []crd.EMQXNode{
			{Name: "emqx@emqx-core-0", PodName: "emqx-core-0", Status: "running"},
			{Name: "emqx@emqx-core-1", PodName: "emqx-core-1", Status: "running"},
		}
		instance.Status.ReplicantNodes = []crd.EMQXNode{
			{Name: "emqx@emqx-replicant-0", PodName: "emqx-replicant-0", Status: "running"},
		}
		instance.Status.NodeEvacuations = []crd.NodeEvacuationStatus{
			{NodeName: "emqx@emqx-core-1", State: "evicting_sessions"},
		}
		instance.Status.DSReplication = crd.DSReplicationStatus{
			DBs: []crd.DSDBReplicationStatus{
				{Name: "messages", NumShards: 16, NumShardReplicas: 32},
			},
		}
		instance.Status.SetCondition(
			crd.EMQXAPIAvailable,
			metav1.ConditionFalse,
			"RequestFailed",
			"previous failure",
		)
		Expect(k8sClient.Status().Update(ctx, instance)).Should(Succeed())

		Expect(actualize(instance)).NotTo(HaveOccurred())
		_, conditionBefore := instance.Status.GetCondition(crd.EMQXAPIAvailable)
		Expect(conditionBefore).NotTo(BeNil())

		round := newReconcileRoundWithRequester(req.MockRequests("*", syscall.ECONNREFUSED))
		s := &updateStatus{emqxReconciler()}
		Expect(runRoundReconcile(round, instance, s)).To(BeSuccessfulReconcile())

		Expect(actualize(instance)).NotTo(HaveOccurred())
		_, condition := instance.Status.GetCondition(crd.EMQXAPIAvailable)
		Expect(condition).NotTo(BeNil())
		Expect(condition.Status).To(Equal(metav1.ConditionFalse))
		Expect(condition.Reason).To(Equal("RequestFailed"))
		Expect(condition.Message).To(ContainSubstring("connection refused"))
		Expect(condition.LastTransitionTime).To(Equal(conditionBefore.LastTransitionTime))
		Expect(instance.Status.CoreNodes).To(BeEmpty())
		Expect(instance.Status.ReplicantNodes).To(BeEmpty())
		Expect(instance.Status.NodeEvacuations).To(BeEmpty())
		Expect(instance.Status.DSReplication.DBs).To(BeEmpty())
	})
})
