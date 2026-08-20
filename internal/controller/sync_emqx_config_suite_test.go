package controller

import (
	"net/http"
	"net/url"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	"github.com/emqx/emqx-operator/internal/controller/config"
	resources "github.com/emqx/emqx-operator/internal/controller/resources"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	req "github.com/emqx/emqx-operator/internal/requester"
	. "github.com/emqx/emqx-operator/test/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

var _ = DescribeClientFaultMatrix("Reconciler syncConfig", Ordered, func() {
	var ns *corev1.Namespace
	var instance *crd.EMQX

	BeforeEach(func() {
		ns = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			GenerateName: "controller-sync-config-test-",
		}}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Name = "emqx-sync-config"
		Expect(k8sClient.Create(ctx, instance)).To(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, ns)).To(Succeed())
	})

	createCoreSet := func() {
		Expect(k8sClient.Create(ctx, newStatefulSet(instance))).To(Succeed())
	}

	createReadyCorePod := func() *corev1.Pod {
		annotations := map[string]string{}
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        instance.CoreName() + "-0",
				Namespace:   instance.Namespace,
				Labels:      instance.DefaultLabelsWith(crd.CoreLabels()),
				Annotations: annotations,
			},
			Spec: corev1.PodSpec{Containers: []corev1.Container{{
				Name:  "emqx",
				Image: instance.Spec.Image,
			}}},
		}
		Expect(k8sClient.Create(ctx, pod)).To(Succeed())
		pod.Status.Conditions = []corev1.PodCondition{{
			Type:   corev1.ContainersReady,
			Status: corev1.ConditionTrue,
		}}
		Expect(k8sClient.Status().Update(ctx, pod)).To(Succeed())
		return pod
	}

	It("manages the ConfigMap and updates it when configuration roots change", func() {
		configMap := &corev1.ConfigMap{}

		instance.Spec.Config.Roots = crd.ConfigRoots{
			"log":       apiextv1.JSON{Raw: FromYAMLString(`console: { level: info }`)},
			"dashboard": apiextv1.JSON{Raw: FromYAMLString(`listeners: { http: { bind: 18083 } }`)},
		}
		Expect(k8sClient.Update(ctx, instance)).To(Succeed())

		round := newReconcileRoundWithRequester(nil)
		reconciler := &syncConfig{emqxReconciler()}
		Eventually(runRoundReconcile).WithArguments(round, instance, reconciler).
			Should(BeSuccessfulReconcile())
		Expect(k8sClient.Get(ctx, instance.ConfigsNamespacedName(), configMap)).To(Succeed())
		Expect(metav1.IsControlledBy(configMap, instance)).To(BeTrue())
		Expect(configMap.Data[resources.BaseConfigFile]).To(
			ContainSubstring(`"log" = {"console":{"level":"info"}}`),
		)
		Expect(configMap.Data[resources.StartupConfigFile]).To(And(
			ContainSubstring(`"dashboard" = {"listeners":{"http":{"bind":18083}}}`),
			Not(ContainSubstring(`"log" = {`)),
		))

		instance.Spec.Config.Roots = crd.ConfigRoots{
			"log": apiextv1.JSON{Raw: FromYAMLString(`console: { level: warning }`)},
		}
		Expect(k8sClient.Update(ctx, instance)).To(Succeed())

		Eventually(runRoundReconcile).WithArguments(round, instance, reconciler).
			Should(BeSuccessfulReconcile())
		Expect(k8sClient.Get(ctx, instance.ConfigsNamespacedName(), configMap)).To(Succeed())
		Expect(configMap.Data[resources.BaseConfigFile]).To(ContainSubstring(`"log" = {"console":{"level":"warning"}}`))
		Expect(configMap.Data[resources.StartupConfigFile]).To(BeEmpty())
	})

	It("propagates runtime-applicable configuration through the EMQX API", func() {
		roots := crd.ConfigRoots{
			"log": apiextv1.JSON{Raw: FromYAMLString(`console: {level: debug}`)},
		}
		instance.Spec.Config.Roots = roots
		Expect(k8sClient.Update(ctx, instance)).To(Succeed())

		createCoreSet()
		createReadyCorePod()

		var calls int
		var updatedConfig string
		requester := req.NewMockRequester(func(
			method string,
			requestURL url.URL,
			body []byte,
			header http.Header,
		) (*http.Response, []byte, error) {
			calls++
			Expect(method).To(Equal(http.MethodPut))
			Expect(requestURL.Path).To(Equal("api/v5/configs"))
			Expect(requestURL.RawQuery).To(Equal("mode=replace"))
			Expect(header.Get("Content-Type")).To(Equal("text/plain"))
			updatedConfig = string(body)
			return &http.Response{StatusCode: http.StatusOK}, nil, nil
		})

		reconciler := &syncConfig{emqxReconciler()}
		Eventually(runRoundReconcile).
			WithArguments(newReconcileRoundWithRequester(requester), instance, reconciler).
			Should(BeSuccessfulReconcile())
		Expect(calls).To(BeNumerically(">=", 1))
		Expect(updatedConfig).To(ContainSubstring(`"log" = {"console":{"level":"debug"}}`))

		Expect(actualize(instance)).To(Succeed())
		Expect(instance.Status.Config.RuntimeRevision).
			To(Equal(configStringHash(config.RenderRoots(roots))))
		Expect(instance).
			To(HaveCondition(crd.ConfigApplied, HaveField("Status", Equal(metav1.ConditionTrue))))
	})

	It("reports pending startup configuration until ready Pods carry its revision", func() {
		instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(2))
		roots := crd.ConfigRoots{
			"dashboard": apiextv1.JSON{Raw: FromYAMLString(`
				listeners:
				  http:
				    bind: 28083
			`)},
		}
		instance.Spec.Config.Roots = roots
		Expect(k8sClient.Update(ctx, instance)).To(Succeed())

		createCoreSet()
		pod := createReadyCorePod()

		runtimeRoots, startupRoots := config.SplitRoots(roots)
		startupRevision := startupConfigRevision(startupRoots)
		instance.Status.Config.RuntimeRevision = configRevision(runtimeRoots)
		Expect(k8sClient.Status().Update(ctx, instance)).To(Succeed())

		round := newReconcileRoundWithRequester(nil)
		reconciler := &syncConfig{emqxReconciler()}
		Eventually(runRoundReconcile).WithArguments(round, instance, reconciler).
			Should(BeSuccessfulReconcile())
		Expect(actualObject(instance)).To(HaveCondition(crd.ConfigApplied, And(
			HaveField("Status", Equal(metav1.ConditionFalse)),
			HaveField("Reason", Equal("StartupConfigPending")),
			HaveField("Message", Equal(
				"Configuration roots require rolling restart: [dashboard]",
			)),
		)))

		util.AttachAnnotation(pod, crd.AnnotationStartupConfigRevision, startupRevision)
		Expect(k8sClient.Update(ctx, pod)).To(Succeed())

		Eventually(runRoundReconcile).WithArguments(round, instance, reconciler).
			Should(BeSuccessfulReconcile())
		Expect(actualObject(instance)).To(HaveCondition(crd.ConfigApplied, And(
			HaveField("Status", Equal(metav1.ConditionTrue)),
			HaveField("Reason", Equal("Applied")),
		)))

		pod.Status.Conditions[0].Status = corev1.ConditionFalse
		Expect(k8sClient.Status().Update(ctx, pod)).To(Succeed())
		Eventually(runRoundReconcile).WithArguments(round, instance, reconciler).
			Should(BeSuccessfulReconcile())
		Expect(actualObject(instance)).To(HaveCondition(crd.ConfigApplied, And(
			HaveField("Status", Equal(metav1.ConditionFalse)),
			HaveField("Reason", Equal("PendingAvailability")),
			HaveField("Message", Equal(
				"Desired startup configuration needs ready EMQX pod for verification",
			)),
		)))
	})
})
