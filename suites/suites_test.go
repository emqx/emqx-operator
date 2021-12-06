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
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/emqx/emqx-operator/api/v1beta1"
	"github.com/emqx/emqx-operator/controllers"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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
	BrokerName      = "emqx"
	BrokerNameSpace = "broker"

	EnterpriseName      = "emqx-ee"
	EnterpriseNameSpace = "enterprise"

	Timeout  = time.Second * 10
	Duration = time.Second * 10
	Interval = time.Millisecond * 250
)

var k8sClient client.Client
var testEnv *envtest.Environment

var emqxList = func() []v1beta1.Emqx {
	return []v1beta1.Emqx{
		GenerateEmqxBroker(BrokerName, BrokerNameSpace),
		GenerateEmqxEnterprise(EnterpriseName, EnterpriseNameSpace),
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
		namespace := GenerateEmqxNamespace(emqx.GetNamespace())
		Expect(k8sClient.Create(context.Background(), namespace)).Should(Succeed())

		sa, role, roleBinding := GenerateRBAC(emqx.GetName(), emqx.GetNamespace())
		Expect(k8sClient.Create(context.Background(), sa)).Should(Succeed())
		Expect(k8sClient.Create(context.Background(), role)).Should(Succeed())
		Expect(k8sClient.Create(context.Background(), roleBinding)).Should(Succeed())
	}
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func GenerateEmqxNamespace(namespace string) *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
}

func GenerateRBAC(name, namespace string) (*corev1.ServiceAccount, *rbacv1.Role, *rbacv1.RoleBinding) {
	meta := metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}

	sa := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: meta,
	}

	role := &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: meta,
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "watch", "list"},
				APIGroups: []string{""},
				Resources: []string{"endpoints"},
			},
		},
	}

	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: meta,
		Subjects: []rbacv1.Subject{
			{
				Kind:      sa.Kind,
				Name:      sa.Name,
				Namespace: sa.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     role.Kind,
			Name:     role.Name,
		},
	}

	return sa, role, roleBinding
}

func GenerateEmqxBroker(name, namespace string) *v1beta1.EmqxBroker {
	replicas := int32(3)
	return &v1beta1.EmqxBroker{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps.emqx.io/v1beta1",
			Kind:       "EmqxBroker",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1beta1.EmqxBrokerSpec{
			Image:              "emqx/emqx:4.3.10",
			ServiceAccountName: "emqx",
			Replicas:           &replicas,
		},
	}
}

func GenerateEmqxEnterprise(name, namespace string) *v1beta1.EmqxEnterprise {
	replicas := int32(3)
	return &v1beta1.EmqxEnterprise{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps.emqx.io/v1beta1",
			Kind:       "EmqxBroker",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1beta1.EmqxEnterpriseSpec{
			Image:              "emqx/emqx-ee:4.3.5",
			ServiceAccountName: "emqx",
			Replicas:           &replicas,
		},
	}
}

func DeleteAll(emqx v1beta1.Emqx) error {
	ctx := context.Background()

	for _, resource := range listResource(emqx) {
		err := k8sClient.Delete(ctx, resource)
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func EnsureDeleteAll(emqx v1beta1.Emqx) bool {
	ctx := context.Background()

	if err := k8sClient.Get(
		ctx,
		types.NamespacedName{Name: emqx.GetName(), Namespace: emqx.GetNamespace()},
		&corev1.Service{},
	); !errors.IsNotFound(err) {
		return false
	}
	if err := k8sClient.Get(
		ctx,
		types.NamespacedName{Name: emqx.GetHeadlessServiceName(), Namespace: emqx.GetNamespace()},
		&corev1.Service{},
	); !errors.IsNotFound(err) {
		return false
	}
	if err := k8sClient.Get(
		ctx,
		types.NamespacedName{
			Name:      emqx.GetACL()["name"],
			Namespace: emqx.GetNamespace(),
		},
		&corev1.ConfigMap{},
	); !errors.IsNotFound(err) {
		return false
	}
	if err := k8sClient.Get(
		ctx,
		types.NamespacedName{
			Name:      emqx.GetLoadedModules()["name"],
			Namespace: emqx.GetNamespace(),
		},
		&corev1.ConfigMap{},
	); !errors.IsNotFound(err) {
		return false
	}
	if err := k8sClient.Get(
		ctx,
		types.NamespacedName{
			Name:      emqx.GetLoadedPlugins()["name"],
			Namespace: emqx.GetNamespace(),
		},
		&corev1.ConfigMap{},
	); !errors.IsNotFound(err) {
		return false
	}
	if err := k8sClient.Get(
		ctx,
		types.NamespacedName{
			Name:      emqx.GetName(),
			Namespace: emqx.GetNamespace(),
		},
		&appsv1.StatefulSet{},
	); !errors.IsNotFound(err) {
		return false
	}
	return true
}

func listResource(emqx v1beta1.Emqx) []client.Object {
	meta := metav1.ObjectMeta{
		Name:      emqx.GetName(),
		Namespace: emqx.GetNamespace(),
	}

	list := []client.Object{
		emqx,
		// &corev1.ServiceAccount{ObjectMeta: meta},
		// &rbacv1.Role{ObjectMeta: meta},
		// &rbacv1.RoleBinding{ObjectMeta: meta},
		&corev1.Service{ObjectMeta: meta},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      emqx.GetHeadlessServiceName(),
				Namespace: emqx.GetNamespace(),
			},
		},
		&corev1.ConfigMap{ObjectMeta: meta},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      emqx.GetACL()["name"],
				Namespace: emqx.GetNamespace(),
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      emqx.GetLoadedPlugins()["name"],
				Namespace: emqx.GetNamespace(),
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      emqx.GetLoadedModules()["name"],
				Namespace: emqx.GetNamespace(),
			},
		},
		&appsv1.StatefulSet{ObjectMeta: meta},
	}
	return list
}
