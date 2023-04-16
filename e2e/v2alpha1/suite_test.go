/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v2alpha1

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zapcore"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	appscontrollersv2alpha1 "github.com/emqx/emqx-operator/controllers/apps/v2alpha1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

// var cfg *rest.Config
var timeout, interval time.Duration
var k8sClient client.Client
var testEnv *envtest.Environment
var emqx *appsv2alpha1.EMQX

func TestAPIs(t *testing.T) {
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
	emqx = genEMQX()
	timeout = time.Minute * 5
	interval = time.Second * 1

	Expect(os.Setenv("USE_EXISTING_CLUSTER", "true")).To(Succeed())

	opts := zap.Options{
		Development: true,
		Level:       zapcore.DebugLevel,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
		DestWriter:  GinkgoWriter,
	}

	logf.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = appsv2alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		MetricsBindAddress: "0",
	})
	Expect(err).ToNot(HaveOccurred())

	err = appscontrollersv2alpha1.NewEMQXReconciler(k8sManager).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func genEMQX() *appsv2alpha1.EMQX {
	emqx := &appsv2alpha1.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "e2e-test-v2alpha1" + "-" + rand.String(5),
		},
		Spec: appsv2alpha1.EMQXSpec{
			ReplicantTemplate: appsv2alpha1.EMQXReplicantTemplate{
				Spec: appsv2alpha1.EMQXReplicantTemplateSpec{
					Replicas: pointer.Int32(2),
				},
			},
			// TODO: emqx 5.0.22 have bug, can not use and gateway config in emqx.conf
			// Wait emqx fix it, and restore this change
			// Image: "emqx:5.0",
			Image: "emqx:5.0.21",
			BootstrapConfig: `
			gateway.lwm2m {
			  auto_observe = true
			  enable_stats = true
			  idle_timeout = "30s"
			  lifetime_max = "86400s"
			  lifetime_min = "1s"
			  listeners {
			    udp {
			      default {
			        bind = "5783"
			        max_conn_rate = 1000
			        max_connections = 1024000
			      }
			    }
			  }
			  mountpoint = ""
			  qmode_time_window = "22s"
			  translators {
			    command {qos = 0, topic = "dn/#"}
			    notify {qos = 0, topic = "up/notify"}
			    register {qos = 0, topic = "up/resp"}
			    response {qos = 0, topic = "up/resp"}
			    update {qos = 0, topic = "up/update"}
			  }
			  update_msg_publish_condition = "contains_object_list"
			  xml_dir = "etc/lwm2m_xml/"
			}
			`,
		},
	}
	emqx.Default()
	return emqx
}
