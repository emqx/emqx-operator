/*
Copyright 2025-2026.

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

package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	ginkgotypes "github.com/onsi/ginkgo/v2/types"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zapcore"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	config "github.com/emqx/emqx-operator/internal/controller/config"
	req "github.com/emqx/emqx-operator/internal/requester"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var k8sManager ctrl.Manager
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc
var logger logr.Logger
var timeout, interval time.Duration

var emqxReconciler *EMQXReconciler
var emqxConf *config.EMQX
var emqx *crd.EMQX = &crd.EMQX{
	ObjectMeta: metav1.ObjectMeta{
		UID:  "fake-1234567890",
		Name: "emqx",
		Labels: map[string]string{
			crd.LabelManagedBy: "emqx-operator",
			crd.LabelInstance:  "emqx",
		},
	},
	Spec: crd.EMQXSpec{
		Image: "emqx",
		UpdateStrategy: crd.UpdateStrategy{
			Type: "RollingUpdate",
			EvacuationStrategy: crd.EvacuationStrategy{
				Type: "NodeEvacuation",
			},
		},
	},
}

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite", ginkgotypes.ReporterConfig{
		Verbose: true,
	})
}

var _ = BeforeSuite(func() {
	timeout = time.Second * 10
	interval = time.Second

	gomega.SetDefaultEventuallyTimeout(timeout)
	gomega.SetDefaultEventuallyPollingInterval(interval)

	logger = zap.New(
		zap.WriteTo(GinkgoWriter),
		zap.UseDevMode(true),
		zap.Level(zapcore.DebugLevel),
	)
	logf.SetLogger(logger)

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,

		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.31.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = crd.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
	})
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

	emqxReconciler = NewEMQXReconciler(k8sManager)
	emqxConf, err = config.EMQXConfigWithDefaults(emqx.Spec.Config.Data)
	Expect(err).ToNot(HaveOccurred())
	Expect(emqxConf).ToNot(BeNil())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	_ = testEnv.Stop()
	// err := testEnv.Stop()
	// Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("CRD Defaults", Ordered, func() {
	var ns *corev1.Namespace

	BeforeAll(func() {
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "api-defaults-test-ns",
			},
		}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, ns)).To(Succeed())
	})

	It("defaults coreTemplate.spec when spec is missing", func() {
		instance := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": crd.GroupVersion.String(),
				"kind":       "EMQX",
				"metadata": map[string]interface{}{
					"name":      "emqx",
					"namespace": ns.Name,
				},
				"spec": map[string]interface{}{
					"image": "emqx",
					"coreTemplate": map[string]interface{}{
						"metadata": map[string]interface{}{},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, instance)).To(Succeed())

		actual := &crd.EMQX{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: "emqx"}, actual)).To(Succeed())
		Expect(actual.Spec.CoreTemplate.Spec).To(HaveField("Replicas", And(
			Not(BeNil()),
			HaveValue(BeEquivalentTo(2)),
		)))
		Expect(actual.Spec.CoreTemplate.Spec).To(HaveField("PodSecurityContext", And(
			Not(BeNil()),
			HaveValue(HaveField("RunAsUser", HaveValue(BeEquivalentTo(1000)))),
		)))
	})
})

func actualObject[Object client.Object](o Object) (Object, error) {
	err := k8sClient.Get(ctx, client.ObjectKeyFromObject(o), o)
	return o, err
}

func ownerReferences(owner client.Object) []metav1.OwnerReference {
	var apiVersion, kind string
	switch owner.(type) {
	case *appsv1.StatefulSet:
		apiVersion = "apps/v1"
		kind = "StatefulSet"
	case *appsv1.ReplicaSet:
		apiVersion = "apps/v1"
		kind = "ReplicaSet"
	}
	return []metav1.OwnerReference{
		{
			APIVersion:         apiVersion,
			Kind:               kind,
			Name:               owner.GetName(),
			UID:                owner.GetUID(),
			BlockOwnerDeletion: ptr.To(true),
			Controller:         ptr.To(true),
		},
	}
}

func newReconcileRound() *reconcileRound {
	req := req.NewMockRequester(
		func(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error) {
			return &http.Response{StatusCode: 501}, []byte{}, nil
		},
	)
	return newReconcileRoundWithRequester(req)
}

func newReconcileRoundWithRequester(requester req.RequesterInterface) *reconcileRound {
	return &reconcileRound{
		ctx:       ctx,
		log:       logger,
		conf:      emqxConf,
		requester: &apiRequesterOverride{requester},
		state:     &reconcileState{},
	}
}

type apiRequesterOverride struct {
	requester req.RequesterInterface
}

func (b *apiRequesterOverride) forOldestCore(_ *reconcileState, _ ...reconcileStatePodFilter) req.RequesterInterface {
	return b.requester
}

func (b *apiRequesterOverride) forPod(_ *corev1.Pod) req.RequesterInterface {
	return b.requester
}

type apiRequesterUnavailable struct {
}

func (b *apiRequesterUnavailable) forOldestCore(_ *reconcileState, _ ...reconcileStatePodFilter) req.RequesterInterface {
	return nil
}

func (b *apiRequesterUnavailable) forPod(_ *corev1.Pod) req.RequesterInterface {
	return nil
}
