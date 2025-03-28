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

package v2beta1

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	dedent "github.com/lithammer/dedent"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"

	appscontrollersv1beta4 "github.com/emqx/emqx-operator/controllers/apps/v1beta4"
	appscontrollersv2beta1 "github.com/emqx/emqx-operator/controllers/apps/v2beta1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

// var cfg *rest.Config
var timeout, interval time.Duration
var testEnv *envtest.Environment
var ctx context.Context
var k8sClient client.Client
var emqx emqxSpecs

type emqxSpecs struct {
	coresOnly       *appsv2beta1.EMQX
	coresReplicants *appsv2beta1.EMQX
}

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
	emqx.initSpecs()
	timeout = time.Minute * 3
	interval = time.Second * 1
	ctx = context.Background()

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

	err = appsv1beta4.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = appsv2beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&appscontrollersv1beta4.EmqxEnterpriseReconciler{
		EmqxReconciler: appscontrollersv1beta4.NewEmqxReconciler(k8sManager),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = appscontrollersv2beta1.NewEMQXReconciler(k8sManager).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = appscontrollersv2beta1.NewRebalanceReconciler(k8sManager).SetupWithManager(k8sManager)
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

func (e *emqxSpecs) initSpecs() {
	// Sample config.
	config := dedent.Dedent(`
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
	`)

	// Cores only cluster.
	e.coresOnly = &appsv2beta1.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "e2e-test-v2beta1" + "-" + rand.String(5),
		},
		Spec: appsv2beta1.EMQXSpec{
			Image:           "emqx/emqx-enterprise:latest",
			ImagePullPolicy: corev1.PullAlways,
			ClusterDomain:   "cluster.local",
			Config:          appsv2beta1.Config{Data: config},
		},
	}
	e.coresOnly.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(2))

	// Cores and replicants cluster.
	e.coresReplicants = e.coresOnly.DeepCopy()
	e.coresReplicants.Spec.ReplicantTemplate = &appsv2beta1.EMQXReplicantTemplate{}
	e.coresReplicants.Spec.ReplicantTemplate.Spec.Replicas = ptr.To(int32(2))
}
