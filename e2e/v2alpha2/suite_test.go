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

package v2alpha2

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"go.uber.org/zap/zapcore"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	appscontrollersv2alpha2 "github.com/emqx/emqx-operator/controllers/apps/v2alpha2"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

// var cfg *rest.Config
var timeout, interval time.Duration
var k8sClient client.Client
var testEnv *envtest.Environment
var emqx *appsv2alpha2.EMQX

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	format.MaxLength = 0

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

	err = appsv2alpha2.AddToScheme(scheme.Scheme)
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

	err = appscontrollersv2alpha2.NewEMQXReconciler(k8sManager).SetupWithManager(k8sManager)
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

func genEMQX() *appsv2alpha2.EMQX {
	emqx := &appsv2alpha2.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "e2e-test-v2alpha2" + "-" + rand.String(5),
		},
		Spec: appsv2alpha2.EMQXSpec{
			Image:         "emqx/emqx-enterprise:5.1.0",
			ClusterDomain: "cluster.local",
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
