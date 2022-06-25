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

package controller_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	controllers "github.com/emqx/emqx-operator/controllers/apps"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
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

const (
	brokerName      = "emqx"
	brokerNameSpace = "broker"

	enterpriseName      = "emqx-ee"
	enterpriseNameSpace = "enterprise"
)

var timeout, interval time.Duration
var k8sClient client.Client
var testEnv *envtest.Environment

var emqxList = func() []v1beta3.Emqx {
	return []v1beta3.Emqx{
		generateEmqxBroker(brokerName, brokerNameSpace),
		generateEmqxEnterprise(enterpriseName, enterpriseNameSpace),
	}
}

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
	interval = time.Millisecond * 250
	timeout = time.Minute * 1

	if os.Getenv("CI") == "true" {
		Expect(os.Setenv("USE_EXISTING_CLUSTER", "true")).To(Succeed())
		timeout = time.Minute * 10
	}

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
	handler := controllers.Handler{
		Client:        k8sClient,
		Clientset:     *clientset,
		Config:        *cfg,
		EventRecorder: k8sManager.GetEventRecorderFor("emqx-operator"),
	}

	err = (&controllers.EmqxBrokerReconciler{
		Handler: handler,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&controllers.EmqxEnterpriseReconciler{
		Handler: handler,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&controllers.EmqxPluginReconciler{
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

	emqxReady := make(chan string)
	for _, emqx := range emqxList() {
		go func(emqx v1beta3.Emqx) {
			namespace := generateEmqxNamespace(emqx.GetNamespace())
			Expect(k8sClient.Create(context.Background(), namespace)).Should(Succeed())
			Expect(k8sClient.Create(context.Background(), emqx)).Should(Succeed())

			if os.Getenv("USE_EXISTING_CLUSTER") == "true" {
				var instance v1beta3.Emqx
				switch emqx.(type) {
				case *v1beta3.EmqxBroker:
					instance = &v1beta3.EmqxBroker{}
				case *v1beta3.EmqxEnterprise:
					instance = &v1beta3.EmqxEnterprise{}
				}
				Eventually(func() []v1beta3.ConditionType {
					_ = k8sClient.Get(
						context.TODO(),
						types.NamespacedName{
							Name:      emqx.GetName(),
							Namespace: emqx.GetNamespace(),
						},
						instance,
					)
					list := []v1beta3.ConditionType{}
					for _, c := range instance.GetStatus().Conditions {
						list = append(list, c.Type)
					}
					return list
				}, timeout, interval).Should(ContainElement(v1beta3.ClusterConditionRunning))
			}
			emqxReady <- "ready"
		}(emqx)
	}

	// // wait emqx custom resource ready
	_, _ = <-emqxReady, <-emqxReady
})

var _ = AfterSuite(func() {
	Expect(cleanAll()).Should(Succeed())

	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func cleanAll() error {
	if err := removePluginsFinalizer(brokerNameSpace); err != nil {
		return err
	}

	broker := &v1beta3.EmqxBroker{}
	if err := k8sClient.Get(
		context.Background(),
		types.NamespacedName{
			Name:      brokerName,
			Namespace: brokerNameSpace,
		},
		broker,
	); !errors.IsNotFound(err) {
		if err := k8sClient.Delete(
			context.Background(),
			&v1beta3.EmqxBroker{
				ObjectMeta: metav1.ObjectMeta{
					Name:      broker.GetName(),
					Namespace: broker.GetNamespace(),
				},
			},
		); err != nil {
			return err
		}
		// If PVC is set, then it should be retained
		if !reflect.ValueOf(broker.GetPersistent()).IsZero() {
			if err := k8sClient.List(
				context.Background(),
				&corev1.PersistentVolumeClaimList{},
				&client.ListOptions{
					Namespace:     broker.GetNamespace(),
					LabelSelector: labels.SelectorFromSet(broker.GetLabels()),
				},
			); err != nil {
				return err
			}
		}
		if err := k8sClient.Delete(
			context.Background(),
			generateEmqxNamespace(brokerNameSpace),
		); err != nil {
			return err
		}
	}

	if err := removePluginsFinalizer(enterpriseNameSpace); err != nil {
		return err
	}
	enterprise := &v1beta3.EmqxEnterprise{}
	if err := k8sClient.Get(
		context.Background(),
		types.NamespacedName{
			Name:      enterpriseName,
			Namespace: enterpriseNameSpace,
		},
		enterprise,
	); !errors.IsNotFound(err) {
		if err := k8sClient.Delete(
			context.Background(),
			&v1beta3.EmqxEnterprise{
				ObjectMeta: metav1.ObjectMeta{
					Name:      enterprise.GetName(),
					Namespace: enterprise.GetNamespace(),
				},
			},
		); err != nil {
			return err
		}
		// If PVC is set, then it should be retained
		if !reflect.ValueOf(enterprise.GetPersistent()).IsZero() {
			if err := k8sClient.List(
				context.Background(),
				&corev1.PersistentVolumeClaimList{},
				&client.ListOptions{
					Namespace:     enterprise.GetNamespace(),
					LabelSelector: labels.SelectorFromSet(enterprise.GetLabels()),
				},
			); err != nil {
				return err
			}
		}
		// if err := k8sClient.Delete(
		// 	context.Background(),
		// 	generateEmqxNamespace(enterpriseNameSpace),
		// ); err != nil {
		// 	return err
		// }
	}

	return nil
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

func generateEmqxNamespace(namespace string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
}

// Full
func generateEmqxBroker(name, namespace string) *v1beta3.EmqxBroker {
	defaultReplicas := int32(3)
	storageClassName := "standard"
	emqx := &v1beta3.EmqxBroker{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps.emqx.io/v1beta3",
			Kind:       "EmqxBroker",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"cluster": "emqx",
			},
		},
		Spec: v1beta3.EmqxBrokerSpec{
			Replicas: &defaultReplicas,
			Persistent: corev1.PersistentVolumeClaimSpec{
				StorageClassName: &storageClassName,
				AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("20Mi"),
					},
				},
			},
			Env: []corev1.EnvVar{
				{
					Name:  "EMQX_LOG__LEVEL",
					Value: "debug",
				},
			},
			InitContainers: []corev1.Container{
				{
					Name:  "busybox",
					Image: "busybox",
					Args: []string{
						"sh",
						"-c",
						"echo 'Hello World'",
					},
				},
			},
			EmqxTemplate: v1beta3.EmqxBrokerTemplate{
				Image: "emqx/emqx:4.4.3",
				Args: []string{
					"bash",
					"-c",
					"echo 'Hello World' && emqx foreground",
				},
				ACL: []string{
					`{allow, all}`,
				},
				Modules: []v1beta3.EmqxBrokerModule{
					{
						Name:   "emqx_mod_acl_internal",
						Enable: true,
					},
				},
			},
		},
	}
	emqx.Default()
	return emqx
}

// Slim
func generateEmqxEnterprise(name, namespace string) *v1beta3.EmqxEnterprise {
	defaultReplicas := int32(3)
	emqx := &v1beta3.EmqxEnterprise{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps.emqx.io/v1beta3",
			Kind:       "EmqxEnterprise",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"cluster": "emqx",
			},
		},
		Spec: v1beta3.EmqxEnterpriseSpec{
			Replicas: &defaultReplicas,
			EmqxTemplate: v1beta3.EmqxEnterpriseTemplate{
				Image: "emqx/emqx-ee:4.4.3",
				// Any []byte slices will be converted to a base64-encoded string when encoding them to JSON.
				// If we create it via kubectl, then the `.spec.emqxTemplate.license.data` needs to be base64
				// If we create it via code, then the `.spec.emqxTemplate.license.data` doesn't needs to be base64
				License: v1beta3.License{
					Data: []byte(`-----BEGIN CERTIFICATE-----
MIIENzCCAx+gAwIBAgIDdMvVMA0GCSqGSIb3DQEBBQUAMIGDMQswCQYDVQQGEwJD
TjERMA8GA1UECAwIWmhlamlhbmcxETAPBgNVBAcMCEhhbmd6aG91MQwwCgYDVQQK
DANFTVExDDAKBgNVBAsMA0VNUTESMBAGA1UEAwwJKi5lbXF4LmlvMR4wHAYJKoZI
hvcNAQkBFg96aGFuZ3doQGVtcXguaW8wHhcNMjAwNjIwMDMwMjUyWhcNNDkwMTAx
MDMwMjUyWjBjMQswCQYDVQQGEwJDTjEZMBcGA1UECgwQRU1RIFggRXZhbHVhdGlv
bjEZMBcGA1UEAwwQRU1RIFggRXZhbHVhdGlvbjEeMBwGCSqGSIb3DQEJARYPY29u
dGFjdEBlbXF4LmlvMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArw+3
2w9B7Rr3M7IOiMc7OD3Nzv2KUwtK6OSQ07Y7ikDJh0jynWcw6QamTiRWM2Ale8jr
0XAmKgwUSI42+f4w84nPpAH4k1L0zupaR10VYKIowZqXVEvSyV8G2N7091+6Jcon
DcaNBqZLRe1DiZXMJlhXnDgq14FPAxffKhCXiCgYtluLDDLKv+w9BaQGZVjxlFe5
cw32+z/xHU366npHBpafCbxBtWsNvchMVtLBqv9yPmrMqeBROyoJaI3nL78xDgpd
cRorqo+uQ1HWdcM6InEFET6pwkeuAF8/jJRlT12XGgZKKgFQTCkZi4hv7aywkGBE
JruPif/wlK0YuPJu6QIDAQABo4HSMIHPMBEGCSsGAQQBg5odAQQEDAIxMDCBlAYJ
KwYBBAGDmh0CBIGGDIGDZW1xeF9iYWNrZW5kX3JlZGlzLGVtcXhfYmFja2VuZF9t
eXNxbCxlbXF4X2JhY2tlbmRfcGdzcWwsZW1xeF9iYWNrZW5kX21vbmdvLGVtcXhf
YmFja2VuZF9jYXNzYSxlbXF4X2JyaWRnZV9rYWZrYSxlbXF4X2JyaWRnZV9yYWJi
aXQwEAYJKwYBBAGDmh0DBAMMATEwEQYJKwYBBAGDmh0EBAQMAjEwMA0GCSqGSIb3
DQEBBQUAA4IBAQDHUe6+P2U4jMD23u96vxCeQrhc/rXWvpmU5XB8Q/VGnJTmv3yU
EPyTFKtEZYVX29z16xoipUE6crlHhETOfezYsm9K0DxF3fNilOLRKkg9VEWcb5hj
iL3a2tdZ4sq+h/Z1elIXD71JJBAImjr6BljTIdUCfVtNvxlE8M0D/rKSn2jwzsjI
UrW88THMtlz9sb56kmM3JIOoIJoep6xNEajIBnoChSGjtBYFNFwzdwSTCodYkgPu
JifqxTKSuwAGSlqxJUwhjWG8ulzL3/pCAYEwlWmd2+nsfotQdiANdaPnez7o0z0s
EujOCZMbK8qNfSbyo50q5iIXhz2ZIGl+4hdp
-----END CERTIFICATE-----`),
				},
			},
		},
	}
	emqx.Default()
	return emqx
}

func updateEmqx(emqx v1beta3.Emqx) error {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(emqx.GetObjectKind().GroupVersionKind())

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
}
