/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	appscontrollersv1beta3 "github.com/emqx/emqx-operator/controllers/apps/v1beta3"
	"github.com/emqx/emqx-operator/pkg/handler"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var timeout, interval time.Duration
var k8sClient client.Client
var testEnv *envtest.Environment

var broker *v1beta3.EmqxBroker = new(v1beta3.EmqxBroker)
var enterprise *v1beta3.EmqxEnterprise = new(v1beta3.EmqxEnterprise)

func TestSuites(t *testing.T) {
	RegisterFailHandler(Fail)

	// fetch the current config
	suiteConfig, reporterConfig := GinkgoConfiguration()
	// adjust it
	suiteConfig.SkipStrings = []string{"NEVER-RUN"}
	reporterConfig.FullTrace = true
	// pass it in to RunSpecs
	RunSpecs(t, "Controller Suite", suiteConfig, reporterConfig)
}

var _ = BeforeSuite(func() {
	timeout = time.Minute * 7
	interval = time.Second * 1

	Expect(os.Setenv("USE_EXISTING_CLUSTER", "true")).To(Succeed())

	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = v1beta3.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	clientset, _ := kubernetes.NewForConfig(cfg)
	handler := handler.Handler{
		Client:        k8sClient,
		Clientset:     *clientset,
		Config:        *cfg,
		EventRecorder: k8sManager.GetEventRecorderFor("emqx-operator"),
	}

	err = (&appscontrollersv1beta3.EmqxBrokerReconciler{
		Handler: handler,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&appscontrollersv1beta3.EmqxEnterpriseReconciler{
		Handler: handler,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&appscontrollersv1beta3.EmqxPluginReconciler{
		Handler: handler,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = k8sManager.AddHealthzCheck("healthz", healthz.Ping)
	Expect(err).ToNot(HaveOccurred())

	err = k8sManager.AddReadyzCheck("readyz", healthz.Ping)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

	broker = &v1beta3.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "broker",
			Labels: map[string]string{
				"cluster": "emqx",
			},
		},
		Spec: v1beta3.EmqxBrokerSpec{
			EmqxTemplate: v1beta3.EmqxBrokerTemplate{
				Image: "emqx/emqx:4.4.4",
			},
		},
	}
	broker.Default()

	enterprise = &v1beta3.EmqxEnterprise{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx-ee",
			Namespace: "enterprise",
			Labels: map[string]string{
				"cluster": "emqx",
			},
		},
		Spec: v1beta3.EmqxEnterpriseSpec{
			EmqxTemplate: v1beta3.EmqxEnterpriseTemplate{
				Image: "emqx/emqx-ee:4.4.4",
			},
		},
	}
	enterprise.Default()

	emqxReady := make(chan string)
	for _, emqx := range []v1beta3.Emqx{broker, enterprise} {
		go func(emqx v1beta3.Emqx) {
			Expect(k8sClient.Create(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: emqx.GetNamespace()}})).Should(Succeed())
			Expect(k8sClient.Create(context.Background(), emqx)).Should(Succeed())

			var instance v1beta3.Emqx
			switch emqx.(type) {
			case *v1beta3.EmqxBroker:
				instance = &v1beta3.EmqxBroker{}
			case *v1beta3.EmqxEnterprise:
				instance = &v1beta3.EmqxEnterprise{}
			}
			Eventually(func() corev1.ConditionStatus {
				_ = k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Name:      emqx.GetName(),
						Namespace: emqx.GetNamespace(),
					},
					instance,
				)
				running := corev1.ConditionFalse
				for _, c := range instance.GetStatus().Conditions {
					if c.Type == v1beta3.ConditionRunning {
						running = c.Status
					}
				}
				return running
			}, timeout, interval).Should(Equal(corev1.ConditionTrue))

			lwm2m := &v1beta3.EmqxPlugin{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps.emqx.io/v1beta3",
					Kind:       "EmqxPlugin",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "lwm2m"),
					Namespace: emqx.GetNamespace(),
					Labels:    emqx.GetLabels(),
				},
				Spec: v1beta3.EmqxPluginSpec{
					PluginName: "emqx_lwm2m",
					Selector:   emqx.GetLabels(),
					Config: map[string]string{
						"lwm2m.lifetime_min": "1s",
						"lwm2m.lifetime_max": "86400s",
						"lwm2m.bind.udp.1":   "0.0.0.0:5683",
						"lwm2m.bind.udp.2":   "0.0.0.0:5684",
						"lwm2m.bind.dtls.1":  "0.0.0.0:5685",
						"lwm2m.bind.dtls.2":  "0.0.0.0:5686",
						"lwm2m.xml_dir":      "/opt/emqx/etc/lwm2m_xml",
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), lwm2m)).Should(Succeed())

			Eventually(func() bool {
				_ = k8sClient.Get(context.Background(), types.NamespacedName{Name: lwm2m.GetName(), Namespace: lwm2m.GetNamespace()}, lwm2m)
				return lwm2m.Status.Phase == v1beta3.EmqxPluginStatusLoaded
			}, timeout, interval).Should(BeTrue())

			emqxReady <- "ready"
		}(emqx)
	}

	// wait emqx custom resource ready
	_, _ = <-emqxReady, <-emqxReady
})

var _ = AfterSuite(func() {
	cleanAll()

	By("tearing down the test environment")
	Expect(testEnv.Stop()).NotTo(HaveOccurred())
})

func cleanAll() {
	Expect(removePluginsFinalizer(broker.GetNamespace())).Should(Succeed())
	Expect(removePluginsFinalizer(enterprise.GetNamespace())).Should(Succeed())

	Expect(k8sClient.Delete(context.Background(), broker)).Should(Succeed())
	Expect(k8sClient.Delete(context.Background(), enterprise)).Should(Succeed())

	Expect(k8sClient.Delete(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: broker.GetNamespace()}})).Should(Succeed())
	Expect(k8sClient.Delete(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: enterprise.GetNamespace()}})).Should(Succeed())
}

func removePluginsFinalizer(namespace string) error {
	finalizer := "apps.emqx.io/finalizer"

	plugins := &v1beta3.EmqxPluginList{}
	_ = k8sClient.List(
		context.Background(),
		plugins,
		client.InNamespace(namespace),
	)
	for _, plugin := range plugins.Items {
		controllerutil.RemoveFinalizer(&plugin, finalizer)
		err := k8sClient.Update(context.Background(), &plugin)
		if err != nil {
			return err
		}
	}
	return nil
}

func updateEmqx(emqx v1beta3.Emqx) {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(emqx.GetObjectKind().GroupVersionKind())
	switch emqx.(type) {
	case *v1beta3.EmqxBroker:
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "apps.emqx.io",
			Version: "v1beta3",
			Kind:    "EmqxBroker",
		})
	case *v1beta3.EmqxEnterprise:
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "apps.emqx.io",
			Version: "v1beta3",
			Kind:    "EmqxEnterprise",
		})
	}

	Eventually(func() error {
		err := k8sClient.Get(
			context.TODO(),
			types.NamespacedName{
				Name:      emqx.GetName(),
				Namespace: emqx.GetNamespace(),
			},
			u,
		)
		if err != nil {
			return err
		}
		emqx.SetResourceVersion(u.GetResourceVersion())
		emqx.SetCreationTimestamp(u.GetCreationTimestamp())
		emqx.SetManagedFields(u.GetManagedFields())

		return k8sClient.Update(context.Background(), emqx)
	}, timeout, interval).Should(Succeed())
}
