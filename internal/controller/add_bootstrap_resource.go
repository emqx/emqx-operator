package controller

import (
	"context"

	emperror "emperror.dev/errors"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv2beta1 "github.com/emqx/emqx-operator/api/v2beta1"
	config "github.com/emqx/emqx-operator/internal/controller/config"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	"github.com/go-logr/logr"
	"github.com/sethvargo/go-password/password"
)

type addBootstrap struct {
	*EMQXReconciler
}

func (a *addBootstrap) reconcile(ctx context.Context, logger logr.Logger, instance *appsv2beta1.EMQX, _ innerReq.RequesterInterface) subResult {
	bootstrapAPIKeys, err := a.getAPIKeyString(ctx, instance)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to get bootstrap api keys")}
	}

	for _, resource := range []client.Object{
		generateNodeCookieSecret(instance, a.conf),
		generateBootstrapAPIKeySecret(instance, bootstrapAPIKeys),
	} {
		if err := ctrl.SetControllerReference(instance, resource, a.Scheme); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to set controller reference")}
		}
		if err := a.Create(ctx, resource); err != nil {
			if !k8sErrors.IsAlreadyExists(err) {
				return subResult{err: emperror.Wrap(err, "failed to create bootstrap configMap")}
			}
		}
	}

	return subResult{}
}

func (a *addBootstrap) getAPIKeyString(ctx context.Context, instance *appsv2beta1.EMQX) (string, error) {
	var bootstrapAPIKeys string

	for _, apiKey := range instance.Spec.BootstrapAPIKeys {
		if apiKey.SecretRef != nil {
			keyValue, err := a.readSecret(ctx, instance.Namespace, apiKey.SecretRef.Key.SecretName, apiKey.SecretRef.Key.SecretKey)
			if err != nil {
				a.EventRecorder.Event(instance, corev1.EventTypeWarning, "GetBootStrapSecretRef", err.Error())
				return "", err
			}
			secretValue, err := a.readSecret(ctx, instance.Namespace, apiKey.SecretRef.Secret.SecretName, apiKey.SecretRef.Secret.SecretKey)
			if err != nil {
				a.EventRecorder.Event(instance, corev1.EventTypeWarning, "GetBootStrapSecretRef", err.Error())
				return "", err
			}
			bootstrapAPIKeys += keyValue + ":" + secretValue + "\n"
		} else {
			bootstrapAPIKeys += apiKey.Key + ":" + apiKey.Secret + "\n"
		}
	}

	return bootstrapAPIKeys, nil
}

func (a *addBootstrap) readSecret(ctx context.Context, namespace string, name string, key string) (string, error) {
	secret := &corev1.Secret{}
	if err := a.Client.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, secret); err != nil {
		return "", emperror.Wrap(err, "failed to get secret")
	}

	if _, ok := secret.Data[key]; !ok {
		return "", emperror.NewWithDetails("secret does not contain the key", "secret", secret.Name, "key", key)
	}

	return string(secret.Data[key]), nil
}

func generateBootstrapAPIKeySecret(instance *appsv2beta1.EMQX, bootstrapAPIKeys string) *corev1.Secret {
	defPassword, _ := password.Generate(64, 10, 0, true, true)
	bootstrapAPIKeys += appsv2beta1.DefaultBootstrapAPIKey + ":" + defPassword
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: instance.Namespace,
			Name:      instance.BootstrapAPIKeyNamespacedName().Name,
			Labels:    appsv2beta1.CloneAndMergeMap(appsv2beta1.DefaultLabels(instance), instance.Labels),
		},
		StringData: map[string]string{
			"bootstrap_api_key": bootstrapAPIKeys,
		},
	}
}

func generateNodeCookieSecret(instance *appsv2beta1.EMQX, conf *config.Conf) *corev1.Secret {
	cookie := conf.GetNodeCookie()
	if cookie == "" {
		cookie, _ = password.Generate(64, 10, 0, true, true)
	}
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: instance.Namespace,
			Name:      instance.NodeCookieNamespacedName().Name,
			Labels:    appsv2beta1.CloneAndMergeMap(appsv2beta1.DefaultLabels(instance), instance.Labels),
		},
		StringData: map[string]string{
			"node_cookie": cookie,
		},
	}
}
