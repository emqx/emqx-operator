package controller

import (
	crd "github.com/emqx/emqx-operator/api/v3beta1"
	req "github.com/emqx/emqx-operator/internal/requester"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = DescribeClientFaultMatrix("Reconciler addService", Ordered, func() {
	var instance *crd.EMQX
	var ns *corev1.Namespace

	validServiceRequester := req.MockRequests(
		"GET api/v5/listeners", `[ {"type":"tcp","name":"default","enable":true,"bind":"1883"} ]`,
		"GET api/v5/gateways", `[]`,
	)

	BeforeAll(func() {
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "controller-add-service-test",
				Labels:       map[string]string{"test": "e2e"},
			},
		}
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
	})

	BeforeEach(func() {
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Spec.CoreTemplate = crd.EMQXCoreTemplate{
			TemplateObjectMeta: crd.TemplateObjectMeta{
				Labels: map[string]string{"test": "label"},
			},
		}
	})

	AfterEach(func() {
		Expect(k8sClient.DeleteAllOf(ctx, &corev1.Service{}, client.InNamespace(ns.Name))).
			To(Succeed())
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})

	It("postpones when there is no usable API requester yet", func() {
		r := newReconcileRound()
		r.requester = &apiRequesterUnavailable{}
		a := &addService{emqxReconciler()}
		Expect(a.reconcile(r, instance)).To(Equal(subResult{needRequeue: true}))
	})

	It("creates the Dashboard Service from roots and the Listeners Service from APIs", func() {
		a := &addService{emqxReconciler()}
		round := newReconcileRoundWithRequester(validServiceRequester)
		Eventually(a.reconcile).WithArguments(round, instance).
			Should(BeSuccessfulReconcile())

		dashboard := &corev1.Service{}
		Expect(k8sClient.Get(ctx, instance.DashboardServiceNamespacedName(), dashboard)).To(Succeed())
		Expect(dashboard.Spec.Selector).To(Equal(instance.DefaultLabelsWith(crd.CoreLabels())))

		listeners := &corev1.Service{}
		Expect(k8sClient.Get(ctx, instance.ListenersServiceNamespacedName(), listeners)).To(Succeed())
		Expect(listeners.Spec.Selector).To(Equal(instance.DefaultLabelsWith(crd.CoreLabels())))
	})

	It("points the Listeners Service at replicants when any are available", func() {
		instance.Spec.ReplicantTemplate = crd.EMQXReplicantTemplate{
			Spec: crd.EMQXReplicantTemplateSpec{
				Replicas: ptr.To(int32(2)),
			},
		}

		round := newReconcileRoundWithRequester(validServiceRequester)
		round.state = &reconcileState{
			replicantSets: []*appsv1.ReplicaSet{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "emqx-rev-current",
						Labels: map[string]string{crd.LabelPodTemplateHash: "rev-current"},
					},
					Status: appsv1.ReplicaSetStatus{ReadyReplicas: 1},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "emqx-rev-update",
						Labels: map[string]string{crd.LabelPodTemplateHash: "rev-update"},
					},
					Status: appsv1.ReplicaSetStatus{ReadyReplicas: 0},
				},
			},
		}

		a := &addService{emqxReconciler()}
		Eventually(a.reconcile).WithArguments(round, instance).
			Should(BeSuccessfulReconcile())

		listeners := &corev1.Service{}
		Expect(k8sClient.Get(ctx, instance.ListenersServiceNamespacedName(), listeners)).To(Succeed())
		Expect(listeners.Spec.Selector).To(Equal(instance.DefaultLabelsWith(crd.ReplicantLabels())))
	})

	It("does not create the Dashboard Service when the template is disabled", func() {
		instance.Spec.DashboardServiceTemplate = &crd.ServiceTemplate{
			Enabled: ptr.To(false),
		}

		a := &addService{emqxReconciler()}
		round := newReconcileRoundWithRequester(validServiceRequester)
		Eventually(a.reconcile).WithArguments(round, instance).
			Should(BeSuccessfulReconcile())

		err := k8sClient.Get(ctx, instance.DashboardServiceNamespacedName(), &corev1.Service{})
		Expect(k8sErrors.IsNotFound(err)).To(BeTrue())

		listeners := &corev1.Service{}
		Expect(k8sClient.Get(ctx, instance.ListenersServiceNamespacedName(), listeners)).To(Succeed())
	})

	It("does not discover listeners when the Listeners Service template is disabled", func() {
		instance.Spec.ListenersServiceTemplate = &crd.ServiceTemplate{Enabled: ptr.To(false)}
		round := newReconcileRoundWithRequester(req.MockRequests())
		a := &addService{emqxReconciler()}
		Eventually(a.reconcile).WithArguments(round, instance).Should(BeSuccessfulReconcile())
		Expect(k8sClient.Get(ctx, instance.DashboardServiceNamespacedName(), &corev1.Service{})).To(Succeed())
		err := k8sClient.Get(ctx, instance.ListenersServiceNamespacedName(), &corev1.Service{})
		Expect(k8sErrors.IsNotFound(err)).To(BeTrue())
	})

	It("does not use the Configs API for listener discovery", func() {
		instance.Spec.DashboardServiceTemplate = &crd.ServiceTemplate{Enabled: ptr.To(false)}
		round := newReconcileRoundWithRequester(req.MockRequests(
			"GET api/v5/listeners", `[ {"type":"tcp","name":"custom","enable":true,"bind":"1884"} ]`,
			"GET api/v5/gateways", `[]`,
		))
		a := &addService{emqxReconciler()}
		Eventually(a.reconcile).WithArguments(round, instance).Should(BeSuccessfulReconcile())
		listeners := &corev1.Service{}
		Expect(k8sClient.Get(ctx, instance.ListenersServiceNamespacedName(), listeners)).To(Succeed())
		Expect(listeners.Spec.Ports).To(HaveLen(1))
		Expect(listeners.Spec.Ports[0].Name).To(Equal("tcp-custom"))
		Expect(listeners.Spec.Ports[0].Port).To(Equal(int32(1884)))
	})

	It("leaves the existing Listeners Service unchanged after partial gateway discovery", func() {
		a := &addService{emqxReconciler()}
		validRound := newReconcileRoundWithRequester(validServiceRequester)
		Eventually(a.reconcile).WithArguments(validRound, instance).Should(BeSuccessfulReconcile())
		before := &corev1.Service{}
		Expect(k8sClient.Get(ctx, instance.ListenersServiceNamespacedName(), before)).To(Succeed())

		round := newReconcileRoundWithRequester(req.MockRequests(
			"GET api/v5/listeners", `[ {"type":"tcp","name":"changed","enable":true,"bind":"1884"} ]`,
			"GET api/v5/gateways", `[ {"name":"lwm2m","status":"running"} ]`,
			"GET api/v5/gateways/lwm2m/listeners", req.MockUnavail(),
		))
		result := a.reconcile(round, instance)
		Expect(result.err).To(MatchError(ContainSubstring(`failed to get listeners for gateway "lwm2m"`)))

		after := &corev1.Service{}
		Expect(k8sClient.Get(ctx, instance.ListenersServiceNamespacedName(), after)).To(Succeed())
		Expect(after.Spec.Ports).To(Equal(before.Spec.Ports))
	})

	It("preserves Kubernetes validation errors and the existing Listeners Service", func() {
		a := &addService{emqxReconciler()}
		validRound := newReconcileRoundWithRequester(validServiceRequester)
		Eventually(a.reconcile).WithArguments(validRound, instance).Should(BeSuccessfulReconcile())
		before := &corev1.Service{}
		Expect(k8sClient.Get(ctx, instance.ListenersServiceNamespacedName(), before)).To(Succeed())

		invalidListenerResponses := []string{
			`[{"type":"tcp","name":"zero","enable":true,"bind":"0"}]`,
			`[{"type":"tcp","name":"too-large","enable":true,"bind":"65536"}]`,
			`[
				{"type":"tcp","name":"duplicate","enable":true,"bind":"1884"},
				{"type":"tcp","name":"duplicate","enable":true,"bind":"1885"}
			]`,
		}
		for _, listenersResponse := range invalidListenerResponses {
			invalidRound := newReconcileRoundWithRequester(req.MockRequests(
				"GET api/v5/listeners", listenersResponse,
				"GET api/v5/gateways", `[]`,
			))
			Eventually(func() string {
				result := a.reconcile(invalidRound, instance)
				if result.err == nil {
					return ""
				}
				return result.err.Error()
			}).Should(And(
				ContainSubstring("failed to create or update services"),
				ContainSubstring("spec.ports"),
			))

			after := &corev1.Service{}
			Expect(k8sClient.Get(ctx, instance.ListenersServiceNamespacedName(), after)).To(Succeed())
			Expect(after.Spec.Ports).To(Equal(before.Spec.Ports))
		}
	})
})
