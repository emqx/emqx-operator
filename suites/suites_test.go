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

package suites_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/emqx/emqx-operator/api/v1beta1"
	"github.com/emqx/emqx-operator/controllers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	brokerName      = "emqx"
	brokerNameSpace = "broker"

	enterpriseName      = "emqx-ee"
	enterpriseNameSpace = "enterprise"

	tuneout  = time.Minute * 5
	interval = time.Millisecond * 250
)

var k8sClient client.Client
var testEnv *envtest.Environment

var emqxList = func() []v1beta1.Emqx {
	return []v1beta1.Emqx{
		generateEmqxBroker(brokerName, brokerNameSpace),
		generateEmqxEnterprise(enterpriseName, enterpriseNameSpace),
	}
}

func TestSuites(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	Expect(os.Setenv("USE_EXISTING_CLUSTER", "true")).To(Succeed())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = v1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	newEmqxBroker := controllers.NewEmqxBrokerReconciler(k8sManager)
	err = newEmqxBroker.SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	newEmqxEnterpriseReconciler := controllers.NewEmqxEnterpriseReconciler(k8sManager)
	err = newEmqxEnterpriseReconciler.SetupWithManager(k8sManager)
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

	for _, emqx := range emqxList() {
		namespace := generateEmqxNamespace(emqx.GetNamespace())
		Expect(k8sClient.Create(context.Background(), namespace)).Should(Succeed())

		Expect(k8sClient.Create(context.Background(), emqx)).Should(Succeed())
	}
}, 60)

var _ = AfterSuite(func() {
	Expect(cleanAll()).Should(Succeed())

	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func cleanAll() error {
	broker := &v1beta1.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      brokerName,
			Namespace: brokerNameSpace,
		},
	}
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
			&v1beta1.EmqxBroker{
				ObjectMeta: metav1.ObjectMeta{
					Name:      brokerName,
					Namespace: brokerNameSpace,
				},
			},
		); err != nil {
			return err
		}
		if err := k8sClient.Delete(
			context.Background(),
			generateEmqxNamespace(brokerNameSpace),
		); err != nil {
			return err
		}
	}

	enterprise := &v1beta1.EmqxEnterprise{
		ObjectMeta: metav1.ObjectMeta{
			Name:      enterpriseName,
			Namespace: enterpriseNameSpace,
		},
	}

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
			&v1beta1.EmqxEnterprise{
				ObjectMeta: metav1.ObjectMeta{
					Name:      enterpriseName,
					Namespace: enterpriseNameSpace,
				},
			},
		); err != nil {
			return err
		}
		if err := k8sClient.Delete(
			context.Background(),
			generateEmqxNamespace(enterpriseNameSpace),
		); err != nil {
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

func generateEmqxBroker(name, namespace string) *v1beta1.EmqxBroker {
	return &v1beta1.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1beta1.EmqxBrokerSpec{
			Image: "emqx/emqx:4.3.10",
		},
	}
}

func generateEmqxEnterprise(name, namespace string) *v1beta1.EmqxEnterprise {
	return &v1beta1.EmqxEnterprise{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1beta1.EmqxEnterpriseSpec{
			Image: "emqx/emqx-ee:4.3.5",
		},
	}
}
