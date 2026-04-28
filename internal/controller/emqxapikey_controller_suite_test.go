package controller

import (
	"encoding/json"
	"net/http"
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	req "github.com/emqx/emqx-operator/internal/requester"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("EMQXAPIKeyReconciler", Ordered, func() {
	var ns *corev1.Namespace
	var apiKey *crd.EMQXAPIKey
	var reconciler *EMQXAPIKeyReconciler

	BeforeAll(func() {
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "controller-emqxapikey-test",
			},
		}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, ns)).To(Succeed())
	})

	BeforeEach(func() {
		reconciler = NewEMQXAPIKeyReconciler(k8sManager)
		apiKey = &crd.EMQXAPIKey{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "operator",
				Namespace: ns.Name,
			},
			Spec: crd.EMQXAPIKeySpec{
				EMQXRef:     crd.LocalEMQXReference{Name: "emqx"},
				SecretRef:   crd.LocalSecretReference{Name: "operator-credentials"},
				Description: "managed",
				Role:        crd.EMQXAPIKeyRoleAdministrator,
			},
		}
		Expect(k8sClient.Create(ctx, apiKey)).To(Succeed())
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(apiKey), apiKey)).To(Succeed())
	})

	AfterEach(func() {
		secret := &corev1.Secret{}
		err := k8sClient.Get(ctx, client.ObjectKey{
			Namespace: ns.Name,
			Name:      apiKey.Spec.SecretRef.Name,
		}, secret)
		if err == nil {
			Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
		} else {
			Expect(k8sErrors.IsNotFound(err)).To(BeTrue())
		}

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(apiKey), apiKey)
		if err == nil {
			Expect(k8sClient.Delete(ctx, apiKey)).To(Succeed())
		} else {
			Expect(k8sErrors.IsNotFound(err)).To(BeTrue())
		}
	})

	It("stores one-time EMQX API key credentials in the configured Secret", func() {
		var request map[string]interface{}
		requester := req.NewMockRequester(
			func(method string, u url.URL, body []byte, header http.Header) (*http.Response, []byte, error) {
				Expect(method).To(Equal(http.MethodPost))
				Expect(u.Path).To(Equal("api/v5/api_key"))
				Expect(json.Unmarshal(body, &request)).To(Succeed())
				return &http.Response{StatusCode: http.StatusOK}, []byte(`{
					"name": "operator",
					"api_key": "generated-key",
					"api_secret": "generated-secret",
					"desc": "managed",
					"enable": true,
					"expired_at": "infinity",
					"role": "administrator",
					"expired": false
				}`), nil
			},
		)

		round := &apiKeyReconcileRound{
			ctx: ctx,
			log: logger,
		}
		result, err := reconciler.reconcileCreate(round, requester, apiKey)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(ctrl.Result{}))

		Expect(request).To(HaveKeyWithValue("name", "operator"))
		Expect(request).To(HaveKeyWithValue("desc", "managed"))
		Expect(request).To(HaveKeyWithValue("role", "administrator"))

		secret := &corev1.Secret{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{
			Namespace: ns.Name,
			Name:      apiKey.Spec.SecretRef.Name,
		}, secret)).To(Succeed())
		Expect(secret.Data).To(HaveKeyWithValue(apiKeySecretKey, []byte("generated-key")))
		Expect(secret.Data).To(HaveKeyWithValue(apiSecretSecretKey, []byte("generated-secret")))
		Expect(secret.OwnerReferences).To(HaveLen(1))
		Expect(secret.OwnerReferences[0].Name).To(Equal(apiKey.Name))

		actual := &crd.EMQXAPIKey{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(apiKey), actual)).To(Succeed())
		Expect(actual.Status.GetCondition(crd.EMQXAPIKeyReady).Status).To(Equal(metav1.ConditionTrue))
		Expect(actual.Status.GetCondition(crd.EMQXAPIKeyIssuing).Status).To(Equal(metav1.ConditionFalse))
		Expect(actual.Status.GetCondition(crd.EMQXAPIKeyExpired).Status).To(Equal(metav1.ConditionFalse))
	})
})
