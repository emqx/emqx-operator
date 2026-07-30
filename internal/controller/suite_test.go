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
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	"go.uber.org/zap/zapcore"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	controllerlog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	config "github.com/emqx/emqx-operator/internal/controller/config"
	"github.com/emqx/emqx-operator/internal/handler"
	req "github.com/emqx/emqx-operator/internal/requester"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var k8sClient client.WithWatch
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc
var logger logr.Logger

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

const (
	clientFaultModeNone   = "none"
	clientFaultModeFixed  = "fixed"
	clientFaultModeRandom = "random"
)

var (
	clientFaultMode         string
	clientFaultRandomEvents = flag.Int(
		"client-fault-events",
		10,
		"number of fault events evaluated when random client faults are simulated",
	)
	clientFaultRandomProbability = flag.Float64(
		"client-fault-probability",
		0.25,
		"probability that each random event emits a client fault",
	)
)

var baseReconciler *EMQXReconciler

func emqxReconciler() *EMQXReconciler {
	switch clientFaultMode {
	case clientFaultModeFixed:
		r := *baseReconciler
		r.Handler = handler.NewHandler(newFaultyClient(k8sClient, 1, 1))
		return &r
	case clientFaultModeRandom:
		r := *baseReconciler
		r.Handler = handler.NewHandler(newFaultyClient(
			k8sClient,
			*clientFaultRandomEvents,
			*clientFaultRandomProbability,
		))
		return &r
	default:
		r := *baseReconciler
		return &r
	}
}

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(time.Second * 10)
	SetDefaultEventuallyPollingInterval(time.Second)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logger = zap.New(
		zap.WriteTo(GinkgoWriter),
		zap.UseDevMode(true),
		zap.Level(zapcore.DebugLevel),
	)
	controllerlog.SetLogger(logger)

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
			fmt.Sprintf("1.32.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	restConfig, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(restConfig).NotTo(BeNil())

	err = crd.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.NewWithWatch(restConfig, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	emqxConf, err = config.EMQXConfigWithDefaults(emqx.Spec.Config.Data)
	Expect(err).ToNot(HaveOccurred())
	Expect(emqxConf).ToNot(BeNil())

	baseReconciler = &EMQXReconciler{
		Handler:       handler.NewHandler(k8sClient),
		RESTConfig:    restConfig,
		Scheme:        scheme.Scheme,
		EventRecorder: &eventLogger{logger},
	}
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
			HaveValue(BeEquivalentTo(1)),
		)))
		Expect(actual.Spec.CoreTemplate.Spec).To(HaveField("PodSecurityContext", And(
			Not(BeNil()),
			HaveValue(HaveField("RunAsUser", HaveValue(BeEquivalentTo(1000)))),
		)))
	})

	It("defaults updateStrategy.evacuationStrategy for minimal instances", func() {
		instance := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": crd.GroupVersion.String(),
				"kind":       "EMQX",
				"metadata": map[string]interface{}{
					"name":      "emqx-update-strategy-defaults",
					"namespace": ns.Name,
				},
				"spec": map[string]interface{}{
					"image": "emqx",
				},
			},
		}
		Expect(k8sClient.Create(ctx, instance)).To(Succeed())

		actual := &crd.EMQX{}
		key := client.ObjectKey{Namespace: ns.Name, Name: "emqx-update-strategy-defaults"}
		Expect(k8sClient.Get(ctx, key, actual)).To(Succeed())
		Expect(actual.Spec.UpdateStrategy.Type).To(Equal("RollingUpdate"))
		Expect(actual.Spec.UpdateStrategy.EvacuationStrategy).To(Equal(crd.EvacuationStrategy{
			Type:            crd.NodeEvacuationStrategy,
			ConnEvictRate:   1000,
			SessEvictRate:   1000,
			WaitTakeover:    10,
			WaitHealthCheck: 60,
		}))

		actual.Annotations = map[string]string{"apps.emqx.io/test": "updated"}
		Expect(k8sClient.Update(ctx, actual)).To(Succeed())
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

// apiRequesterOverride always provides the given fixed API requester.
type apiRequesterOverride struct {
	requester req.RequesterInterface
}

func (b *apiRequesterOverride) forOldestCore(
	_ *reconcileState,
	_ ...reconcileStatePodFilter,
) req.RequesterInterface {
	return b.requester
}

func (b *apiRequesterOverride) forPod(_ *corev1.Pod) req.RequesterInterface {
	return b.requester
}

// apiRequesterUnavailable simulates apiRequester for completely unavailable cluser.
type apiRequesterUnavailable struct{}

func (b *apiRequesterUnavailable) forOldestCore(
	_ *reconcileState,
	_ ...reconcileStatePodFilter,
) req.RequesterInterface {
	return nil
}

func (b *apiRequesterUnavailable) forPod(_ *corev1.Pod) req.RequesterInterface {
	return nil
}

// eventLogger is an EventRecorder that simply redirects events into the specified logger instance.
type eventLogger struct {
	logger logr.Logger
}

func (el *eventLogger) writeEvent(
	object apiruntime.Object,
	annotations map[string]string,
	eventtype,
	reason,
	message string,
) {
	kvs := []any{
		"type", eventtype,
		"reason", reason,
		"message", message,
	}
	gvk, err := apiutil.GVKForObject(object, scheme.Scheme)
	if err == nil {
		kvs = append(kvs, "gv", gvk.GroupVersion())
	}
	if annotations != nil {
		kvs = append(kvs, "annotations", annotations)
	}
	el.logger.Info("controller-event", kvs...)
}

func (el *eventLogger) Event(object apiruntime.Object, eventtype, reason, message string) {
	el.writeEvent(object, nil, eventtype, reason, message)
}

func (el *eventLogger) Eventf(object apiruntime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	el.writeEvent(object, nil, eventtype, reason, fmt.Sprintf(messageFmt, args...))
}

func (el *eventLogger) AnnotatedEventf(
	object apiruntime.Object,
	annotations map[string]string,
	eventtype,
	reason,
	messageFmt string,
	args ...interface{},
) {
	el.writeEvent(object, annotations, eventtype, reason, fmt.Sprintf(messageFmt, args...))
}

// Ginkgo helpers

func DescribeClientFaultMatrix(text string, args ...interface{}) bool {
	var testf func() = nil
	var nodeArgs []interface{}

	nodeArgs = append(nodeArgs, Offset(1))
	for _, arg := range args {
		if reflect.TypeOf(arg).Kind() == reflect.Func {
			testf = arg.(func())
		} else {
			nodeArgs = append(nodeArgs, arg)
		}
	}

	return Describe(text, Ordered, func() {
		var contextArgs []interface{}

		contextArgs = append(nodeArgs, Label("smoke"), func() {
			BeforeAll(func() { clientFaultMode = clientFaultModeNone })
			testf()
		})
		Context("client", contextArgs...)

		contextArgs = append(nodeArgs, func() {
			BeforeAll(func() { clientFaultMode = clientFaultModeRandom })
			testf()
		})
		Context("faulty client", contextArgs...)
	})
}

// Gomega helpers

func BeSuccessfulReconcile() gomegatypes.GomegaMatcher {
	return WithTransform(
		func(in subResult) error {
			return in.err
		},
		Succeed(),
	)
}
