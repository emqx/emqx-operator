package controller

import (
	"net/http"
	"net/url"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	config "github.com/emqx/emqx-operator/internal/controller/config"
	resources "github.com/emqx/emqx-operator/internal/controller/resources"
	req "github.com/emqx/emqx-operator/internal/requester"
	"github.com/lithammer/dedent"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Reconciler syncConfig", Ordered, func() {
	var ns *corev1.Namespace
	var instance *crd.EMQX
	var s *syncConfig

	BeforeEach(func() {
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "sync-emqx-config-",
			},
		}
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Spec.Config.Data = dedent.Dedent(`
			listeners.tcp.default.bind = "0.0.0.0:1883"
		`)
		Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

		s = &syncConfig{emqxReconciler()}
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})

	It("creates the ConfigMap and records last-applied config", func() {
		Eventually(s.reconcile).WithArguments(newReconcileRound(), instance).
			Should(BeSuccessfulReconcile())

		configMap := actualConfigMap(instance)
		Expect(configMap.OwnerReferences).To(ContainElement(HaveField("Name", Equal(instance.Name))))
		Expect(configMap.Data).To(And(
			HaveKeyWithValue(resources.BaseConfigFile, config.WithDefaults(instance.Spec.Config.Data)),
			HaveKeyWithValue(resources.OverridesConfigFile, ""),
		))

		Expect(actualize(instance)).To(Succeed())
		Expect(instance.Annotations).To(HaveKeyWithValue(
			crd.AnnotationLastEMQXConfig,
			instance.Spec.Config.Data,
		))
	})

	It("updates the ConfigMap but defers runtime update until core nodes are ready", func() {
		lastConfig := dedent.Dedent(`
			listeners.tcp.default.bind = "0.0.0.0:1883"
		`)
		updateSpecConfig(instance, lastConfig)
		Eventually(s.reconcile).WithArguments(newReconcileRound(), instance).
			Should(BeSuccessfulReconcile())

		nextConfig := dedent.Dedent(`
			listeners.tcp.default.bind = "0.0.0.0:1884"
		`)
		updateSpecConfig(instance, nextConfig)

		intercept := mkAPIRequesterInterceptor(req.NewMockRequester(
			func(method string, url url.URL, body []byte, _ http.Header) (*http.Response, []byte, error) {
				return &http.Response{StatusCode: 204}, []byte{}, nil
			},
		))
		Eventually(func() subResult {
			return s.reconcile(newSyncConfigReconcileRound(0, intercept.apiRequester()), instance)
		}).
			Should(Equal(reconcilePostpone()))

		configMap := actualConfigMap(instance)
		Expect(configMap.Data).To(
			HaveKeyWithValue(resources.BaseConfigFile, Equal(config.WithDefaults(nextConfig))),
		)
		Expect(intercept.listCaptured()).To(BeEmpty())
		Expect(actualize(instance)).To(Succeed())
		Expect(instance.Annotations).To(
			HaveKeyWithValue(crd.AnnotationLastEMQXConfig, lastConfig),
		)
	})

	It("strips non-changeable dashboard listener update", func() {
		updateSpecConfig(instance, dedent.Dedent(`
			listeners.tcp.default.bind = "0.0.0.0:1883"
		`))
		Eventually(s.reconcile).WithArguments(newReconcileRound(), instance).
			Should(BeSuccessfulReconcile())

		updateSpecConfig(instance, dedent.Dedent(`
			dashboard.listeners.http.bind = 18084
			listeners.tcp.default.bind = "0.0.0.0:1884"
		`))
		Eventually(s.reconcile).WithArguments(newReconcileRound(), instance).
			Should(Equal(reconcilePostpone()))

		configMap := actualConfigMap(instance)
		conf, err := config.EMQXConfig(configMap.Data[resources.BaseConfigFile])
		Expect(err).NotTo(HaveOccurred())
		Expect(configStringAt(conf, "dashboard.listeners.http.bind")).To(Equal("18083"))
		Expect(configStringAt(conf, "listeners.tcp.default.bind")).To(Equal("0.0.0.0:1884"))
	})

	It("strips non-changeable HTTPS dashboard listener update", func() {
		updateSpecConfig(instance, dedent.Dedent(`
			dashboard.listeners.https.bind = 18084
			listeners.tcp.default.bind = "0.0.0.0:1883"
		`))
		Eventually(s.reconcile).WithArguments(newReconcileRound(), instance).
			Should(BeSuccessfulReconcile())

		updateSpecConfig(instance, dedent.Dedent(`
			dashboard.listeners.https.bind = 18085
			listeners.tcp.default.bind = "0.0.0.0:1884"
		`))
		Eventually(s.reconcile).WithArguments(newReconcileRound(), instance).
			Should(Equal(reconcilePostpone()))

		configMap := actualConfigMap(instance)
		conf, err := config.EMQXConfig(configMap.Data[resources.BaseConfigFile])
		Expect(err).NotTo(HaveOccurred())
		Expect(configStringAt(conf, "dashboard.listeners.https.bind")).To(Equal("18084"))
		Expect(configStringAt(conf, "listeners.tcp.default.bind")).To(Equal("0.0.0.0:1884"))
	})

	It("allows enabling a previously disabled dashboard listener", func() {
		updateSpecConfig(instance, dedent.Dedent(`
			dashboard.listeners.http.bind = 0
			listeners.tcp.default.bind = "0.0.0.0:1883"
		`))
		Eventually(s.reconcile).WithArguments(newReconcileRound(), instance).
			Should(BeSuccessfulReconcile())

		updateSpecConfig(instance, dedent.Dedent(`
			dashboard.listeners.http.bind = 18084
			listeners.tcp.default.bind = "0.0.0.0:1884"
		`))
		Eventually(s.reconcile).WithArguments(newReconcileRound(), instance).
			Should(Equal(reconcilePostpone()))

		configMap := actualConfigMap(instance)
		conf, err := config.EMQXConfig(configMap.Data[resources.BaseConfigFile])
		Expect(err).NotTo(HaveOccurred())
		Expect(configStringAt(conf, "dashboard.listeners.http.bind")).To(Equal("18084"))
		Expect(configStringAt(conf, "listeners.tcp.default.bind")).To(Equal("0.0.0.0:1884"))
	})

	It("applies runtime config when core nodes are ready and strips readonly runtime entries", func() {
		instance.Spec.Config.Mode = "Replace"
		updateSpecConfig(instance, dedent.Dedent(`
			listeners.tcp.default.bind = "0.0.0.0:1883"
		`))
		Eventually(s.reconcile).WithArguments(newReconcileRound(), instance).
			Should(BeSuccessfulReconcile())

		nextConfig := dedent.Dedent(`
			node.name = "emqx@127.0.0.1"
			rpc.tcp_client_num = 1
			cluster.name = "readonly"
			cluster.links.remote.server = "remote:1883"
			listeners.tcp.default.bind = "0.0.0.0:1884"
		`)
		updateSpecConfig(instance, nextConfig)

		intercept := mkAPIRequesterInterceptor(req.NewMockRequester(
			func(method string, url url.URL, body []byte, _ http.Header) (*http.Response, []byte, error) {
				return &http.Response{StatusCode: 204}, []byte{}, nil
			},
		))
		Eventually(func() subResult {
			return s.reconcile(newSyncConfigReconcileRound(1, intercept.apiRequester()), instance)
		}).
			Should(Equal(reconcileRequeue()))

		Expect(intercept.listCaptured()).To(ConsistOf(And(
			HaveField("Method", Equal(http.MethodPut)),
			HaveField("URL.Path", Equal("api/v5/configs")),
			HaveField("Header", HaveKeyWithValue("Content-Type", ConsistOf(Equal("text/plain")))),
		)))

		runtimeConfig, err := config.EMQXConfig(intercept.captured(0).Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(runtimeConfig.Find("node")).To(BeNil())
		Expect(runtimeConfig.Find("rpc")).To(BeNil())
		Expect(runtimeConfig.Find("cluster.name")).To(BeNil())
		Expect(runtimeConfig.Find("cluster.links")).NotTo(BeNil())
		Expect(runtimeConfig.Find("listeners")).NotTo(BeNil())

		Expect(actualize(instance)).To(Succeed())
		Expect(instance.Annotations).To(
			HaveKeyWithValue(crd.AnnotationLastEMQXConfig, nextConfig),
		)
	})

	It("returns an error and keeps last-applied config when runtime update fails", func() {
		lastConfig := dedent.Dedent(`
			listeners.tcp.default.bind = "0.0.0.0:1883"
		`)
		updateSpecConfig(instance, lastConfig)
		Eventually(s.reconcile).WithArguments(newReconcileRound(), instance).
			Should(BeSuccessfulReconcile())

		nextConfig := dedent.Dedent(`
			listeners.tcp.default.bind = "0.0.0.0:1884"
		`)
		updateSpecConfig(instance, nextConfig)

		intercept := mkAPIRequesterInterceptor(req.NewMockRequester(
			func(method string, url url.URL, body []byte, _ http.Header) (*http.Response, []byte, error) {
				return &http.Response{StatusCode: 500}, []byte("runtime update failed"), nil
			},
		))
		Eventually(func() subResult {
			return s.reconcile(newSyncConfigReconcileRound(1, intercept.apiRequester()), instance)
		}).
			Should(BeReconcileError(
				MatchError(ContainSubstring("failed to update emqx config through API")),
			))

		Expect(intercept.listCaptured()).To(HaveLen(1))
		Expect(actualize(instance)).To(Succeed())
		Expect(instance.Annotations).To(
			HaveKeyWithValue(crd.AnnotationLastEMQXConfig, lastConfig),
		)
	})
})

func newSyncConfigReconcileRound(readyReplicas int32, requester apiRequester) *reconcileRound {
	round := newReconcileRound()
	round.state.coreSets = []*appsv1.StatefulSet{{
		Status: appsv1.StatefulSetStatus{ReadyReplicas: readyReplicas},
	}}
	round.requester = requester
	return round
}

func updateSpecConfig(instance *crd.EMQX, configData string) {
	instance.Spec.Config.Data = configData
	Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
	Expect(actualize(instance)).To(Succeed())
}

func actualConfigMap(instance *crd.EMQX) *corev1.ConfigMap {
	configMap := &corev1.ConfigMap{}
	Expect(k8sClient.Get(ctx, instance.ConfigsNamespacedName(), configMap)).Should(Succeed())
	return configMap
}

func configStringAt(conf *config.EMQX, path string) string {
	v, ok := conf.Get(path)
	s, err := config.AsString(v)
	Expect(ok).To(BeTrue())
	Expect(err).To(Succeed())
	return s
}
