package controller

import (
	"net/http"
	"net/url"
	"strings"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	config "github.com/emqx/emqx-operator/internal/controller/config"
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

func mockConfigsRequester(configBody string) req.RequesterInterface {
	return req.NewMockRequester(
		func(method string, u url.URL, body []byte, header http.Header) (*http.Response, []byte, error) {
			if method == "GET" && strings.Contains(u.Path, "api/v5/configs") {
				return &http.Response{StatusCode: http.StatusOK}, []byte(configBody), nil
			}
			return &http.Response{StatusCode: http.StatusNotImplemented}, nil, nil
		},
	)
}

var _ = Describe("Reconciler addService", Ordered, func() {
	var a *addService
	var instance *crd.EMQX
	var ns *corev1.Namespace
	var round *reconcileRound

	validConfig := config.WithDefaults("")

	BeforeAll(func() {
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "controller-add-service-test",
				Labels: map[string]string{
					"test": "e2e",
				},
			},
		}
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
	})

	BeforeEach(func() {
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Spec.CoreTemplate = crd.EMQXCoreTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"test": "label"},
			},
		}
		a = &addService{emqxReconciler}
		round = newReconcileRoundWithRequester(mockConfigsRequester(validConfig))
	})

	AfterEach(func() {
		var serviceList corev1.ServiceList
		Expect(k8sClient.List(ctx, &serviceList, client.InNamespace(ns.Name))).To(Succeed())
		for _, service := range serviceList.Items {
			_ = k8sClient.Delete(ctx, &service)
		}
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})

	It("postpones when there is no usable API requester yet", func() {
		r := &reconcileRound{
			ctx:       ctx,
			log:       logger,
			conf:      emqxConf,
			requester: &apiRequesterUnavailable{},
			state:     &reconcileState{},
		}
		Expect(a.reconcile(r, instance)).To(Equal(subResult{needRequeue: true}))
	})

	It("creates Dashboard and Listeners Services from EMQX config", func() {
		Eventually(a.reconcile).WithArguments(round, instance).
			Should(Equal(subResult{}))

		dashboard := &corev1.Service{}
		Expect(k8sClient.Get(ctx, instance.DashboardServiceNamespacedName(), dashboard)).To(Succeed())
		Expect(dashboard.Spec.Selector).To(Equal(instance.DefaultLabelsWith(crd.CoreLabels())))

		listeners := &corev1.Service{}
		Expect(k8sClient.Get(ctx, instance.ListenersServiceNamespacedName(), listeners)).To(Succeed())
		Expect(listeners.Spec.Selector).To(Equal(instance.DefaultLabelsWith(crd.CoreLabels())))
	})

	It("points the Listeners Service at replicants when any are available", func() {
		instance.Spec.ReplicantTemplate = &crd.EMQXReplicantTemplate{
			Spec: crd.EMQXReplicantTemplateSpec{
				Replicas: ptr.To(int32(2)),
			},
		}
		round.state.replicantSets = []*appsv1.ReplicaSet{
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
		}

		Eventually(a.reconcile).WithArguments(round, instance).
			Should(Equal(subResult{}))

		listeners := &corev1.Service{}
		Expect(k8sClient.Get(ctx, instance.ListenersServiceNamespacedName(), listeners)).To(Succeed())
		Expect(listeners.Spec.Selector).To(Equal(instance.DefaultLabelsWith(crd.ReplicantLabels())))
	})

	It("does not create the Dashboard Service when the template is disabled", func() {
		disabled := false
		instance.Spec.DashboardServiceTemplate = &crd.ServiceTemplate{
			Enabled: &disabled,
		}

		r := newReconcileRoundWithRequester(mockConfigsRequester(validConfig))
		Expect(a.reconcile(r, instance)).To(Equal(subResult{}))

		err := k8sClient.Get(ctx, instance.DashboardServiceNamespacedName(), &corev1.Service{})
		Expect(k8sErrors.IsNotFound(err)).To(BeTrue())

		listeners := &corev1.Service{}
		Expect(k8sClient.Get(ctx, instance.ListenersServiceNamespacedName(), listeners)).To(Succeed())
	})
})
