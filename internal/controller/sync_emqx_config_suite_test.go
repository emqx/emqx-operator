package controller

import (
	"net/http"
	"net/url"

	crdv2 "github.com/emqx/emqx-operator/api/v2"
	config "github.com/emqx/emqx-operator/internal/controller/config"
	resources "github.com/emqx/emqx-operator/internal/controller/resources"
	req "github.com/emqx/emqx-operator/internal/requester"
	"github.com/lithammer/dedent"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("Reconciler syncConfig", Ordered, func() {
	var ns *corev1.Namespace
	var instance *crdv2.EMQX
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

		s = &syncConfig{emqxReconciler}
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})

	It("creates the ConfigMap and records last-applied config", func() {
		Eventually(s.reconcile).WithArguments(newReconcileRound(), instance).
			WithTimeout(timeout).
			WithPolling(interval).
			Should(Equal(subResult{}))

		configMap := actualConfigMap(instance)
		Expect(configMap.Data).To(HaveKeyWithValue(
			resources.BaseConfigFile,
			config.WithDefaults(instance.Spec.Config.Data),
		))
		Expect(configMap.Data).To(HaveKeyWithValue(resources.OverridesConfigFile, ""))
		Expect(configMap.OwnerReferences).To(ContainElement(HaveField("Name", Equal(instance.Name))))

		Expect(actualInstance(instance).Annotations).To(HaveKeyWithValue(
			crdv2.AnnotationLastEMQXConfig,
			instance.Spec.Config.Data,
		))
	})

	It("updates the ConfigMap but defers runtime update until core nodes are ready", func() {
		lastConfig := dedent.Dedent(`
			listeners.tcp.default.bind = "0.0.0.0:1883"
		`)
		nextConfig := dedent.Dedent(`
			listeners.tcp.default.bind = "0.0.0.0:1884"
		`)
		instance = establishConfig(s, instance, lastConfig)
		instance = updateSpecConfig(instance, nextConfig)

		calls := []syncConfigAPIRequest{}
		round := newSyncConfigRound(&calls, 204)
		Eventually(s.reconcile).WithArguments(round, instance).
			WithTimeout(timeout).
			WithPolling(interval).
			Should(Equal(subResult{}))

		configMap := actualConfigMap(instance)
		Expect(configMap.Data[resources.BaseConfigFile]).To(Equal(config.WithDefaults(nextConfig)))
		Expect(actualInstance(instance).Annotations).To(HaveKeyWithValue(
			crdv2.AnnotationLastEMQXConfig,
			lastConfig,
		))
		Expect(calls).To(BeEmpty())
	})

	It("strips non-changeable dashboard listener update", func() {
		lastConfig := dedent.Dedent(`
			listeners.tcp.default.bind = "0.0.0.0:1883"
		`)
		nextConfig := dedent.Dedent(`
			dashboard.listeners.http.bind = 18084
			listeners.tcp.default.bind = "0.0.0.0:1884"
		`)
		instance = establishConfig(s, instance, lastConfig)
		instance = updateSpecConfig(instance, nextConfig)

		Eventually(s.reconcile).WithArguments(newReconcileRound(), instance).
			WithTimeout(timeout).
			WithPolling(interval).
			Should(Equal(subResult{}))

		configMap := actualConfigMap(instance)
		conf, err := config.EMQXConfig(configMap.Data[resources.BaseConfigFile])
		Expect(err).NotTo(HaveOccurred())
		Expect(configStringAt(conf, "dashboard.listeners.http.bind")).To(Equal("18083"))
		Expect(configStringAt(conf, "listeners.tcp.default.bind")).To(Equal("0.0.0.0:1884"))
	})

	It("strips non-changeable HTTPS dashboard listener update", func() {
		lastConfig := dedent.Dedent(`
			dashboard.listeners.https.bind = 18084
			listeners.tcp.default.bind = "0.0.0.0:1883"
		`)
		nextConfig := dedent.Dedent(`
			dashboard.listeners.https.bind = 18085
			listeners.tcp.default.bind = "0.0.0.0:1884"
		`)
		instance = establishConfig(s, instance, lastConfig)
		instance = updateSpecConfig(instance, nextConfig)

		Eventually(s.reconcile).WithArguments(newReconcileRound(), instance).
			WithTimeout(timeout).
			WithPolling(interval).
			Should(Equal(subResult{}))

		configMap := actualConfigMap(instance)
		conf, err := config.EMQXConfig(configMap.Data[resources.BaseConfigFile])
		Expect(err).NotTo(HaveOccurred())
		Expect(conf.Get("dashboard.listeners.https.bind")).To(BeNil())
		Expect(configStringAt(conf, "listeners.tcp.default.bind")).To(Equal("0.0.0.0:1884"))
	})

	It("allows enabling a previously disabled dashboard listener", func() {
		lastConfig := dedent.Dedent(`
			dashboard.listeners.http.bind = 0
			listeners.tcp.default.bind = "0.0.0.0:1883"
		`)
		nextConfig := dedent.Dedent(`
			dashboard.listeners.http.bind = 18084
			listeners.tcp.default.bind = "0.0.0.0:1884"
		`)
		instance = establishConfig(s, instance, lastConfig)
		instance = updateSpecConfig(instance, nextConfig)

		Eventually(s.reconcile).WithArguments(newReconcileRound(), instance).
			WithTimeout(timeout).
			WithPolling(interval).
			Should(Equal(subResult{}))

		configMap := actualConfigMap(instance)
		conf, err := config.EMQXConfig(configMap.Data[resources.BaseConfigFile])
		Expect(err).NotTo(HaveOccurred())
		Expect(configStringAt(conf, "dashboard.listeners.http.bind")).To(Equal("18084"))
		Expect(configStringAt(conf, "listeners.tcp.default.bind")).To(Equal("0.0.0.0:1884"))
	})

	It("applies runtime config when core nodes are ready and strips readonly runtime entries", func() {
		lastConfig := dedent.Dedent(`
			listeners.tcp.default.bind = "0.0.0.0:1883"
		`)
		nextConfig := dedent.Dedent(`
			node.name = "emqx@127.0.0.1"
			rpc.tcp_client_num = 1
			cluster.name = "readonly"
			cluster.links.remote.server = "remote:1883"
			listeners.tcp.default.bind = "0.0.0.0:1884"
		`)
		instance = establishConfig(s, instance, lastConfig)
		instance = updateSpecConfig(instance, nextConfig)
		instance.Status.SetTrueCondition(crdv2.CoreNodesReady)

		calls := []syncConfigAPIRequest{}
		round := newSyncConfigRound(&calls, 204)
		Eventually(s.reconcile).WithArguments(round, instance).
			WithTimeout(timeout).
			WithPolling(interval).
			Should(Equal(subResult{result: ctrl.Result{Requeue: true}}))

		Expect(calls).To(ConsistOf(And(
			HaveField("Method", Equal(http.MethodPut)),
			HaveField("Path", Equal("api/v5/configs")),
			HaveField("Header", HaveKeyWithValue("Content-Type", ConsistOf(Equal("text/plain")))),
		)))

		runtimeConfig, err := config.EMQXConfig(calls[0].Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(runtimeConfig.Find("node")).To(BeNil())
		Expect(runtimeConfig.Find("rpc")).To(BeNil())
		Expect(runtimeConfig.Find("cluster.name")).To(BeNil())
		Expect(runtimeConfig.Find("cluster.links")).NotTo(BeNil())
		Expect(runtimeConfig.Find("listeners")).NotTo(BeNil())

		Expect(actualInstance(instance).Annotations).To(HaveKeyWithValue(
			crdv2.AnnotationLastEMQXConfig,
			nextConfig,
		))
	})

	It("returns an error and keeps last-applied config when runtime update fails", func() {
		lastConfig := dedent.Dedent(`
			listeners.tcp.default.bind = "0.0.0.0:1883"
		`)
		nextConfig := dedent.Dedent(`
			listeners.tcp.default.bind = "0.0.0.0:1884"
		`)
		instance = establishConfig(s, instance, lastConfig)
		instance = updateSpecConfig(instance, nextConfig)
		instance.Status.SetTrueCondition(crdv2.CoreNodesReady)

		calls := []syncConfigAPIRequest{}
		result := s.reconcile(newSyncConfigRound(&calls, 500), instance)

		Expect(result.result).To(Equal(ctrl.Result{}))
		Expect(result.err).To(MatchError(ContainSubstring("failed to update emqx config through API")))
		Expect(calls).To(HaveLen(1))
		Expect(actualInstance(instance).Annotations).To(HaveKeyWithValue(
			crdv2.AnnotationLastEMQXConfig,
			lastConfig,
		))
	})
})

type syncConfigAPIRequest struct {
	Method string
	Path   string
	Body   string
	Header http.Header
}

func newSyncConfigRound(calls *[]syncConfigAPIRequest, statusCode int) *reconcileRound {
	requester := req.NewMockRequester(
		func(method string, u url.URL, body []byte, header http.Header) (*http.Response, []byte, error) {
			*calls = append(*calls, syncConfigAPIRequest{
				Method: method,
				Path:   u.Path,
				Body:   string(body),
				Header: header,
			})
			return &http.Response{StatusCode: statusCode}, []byte("runtime update failed"), nil
		},
	)
	return newReconcileRoundWithRequester(requester)
}

func establishConfig(s *syncConfig, instance *crdv2.EMQX, configData string) *crdv2.EMQX {
	instance.Spec.Config.Data = configData
	Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
	Eventually(s.reconcile).WithArguments(newReconcileRound(), actualInstance(instance)).
		WithTimeout(timeout).
		WithPolling(interval).
		Should(Equal(subResult{}))
	return actualInstance(instance)
}

func updateSpecConfig(instance *crdv2.EMQX, configData string) *crdv2.EMQX {
	instance.Spec.Config.Data = configData
	Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
	return actualInstance(instance)
}

func actualConfigMap(instance *crdv2.EMQX) *corev1.ConfigMap {
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
