package controller

import (
	"context"

	emperror "emperror.dev/errors"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	config "github.com/emqx/emqx-operator/internal/controller/config"
	resources "github.com/emqx/emqx-operator/internal/controller/resources"
	"github.com/sethvargo/go-password/password"
)

type addBootstrap struct {
	*EMQXReconciler
}

func (a *addBootstrap) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	bootstrapAPIKeys, err := a.getAPIKeyString(r.ctx, instance)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to get bootstrap api keys")}
	}

	for _, resource := range []client.Object{
		generateNodeCookieSecret(instance, r.conf),
		generateBootstrapAPIKeySecret(instance, bootstrapAPIKeys),
	} {
		if err := ctrl.SetControllerReference(instance, resource, a.Scheme); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to set controller reference")}
		}
		if err := a.Create(r.ctx, resource); err != nil {
			if !k8sErrors.IsAlreadyExists(err) {
				return subResult{err: emperror.Wrap(err, "failed to create bootstrap configMap")}
			}
		}
	}

	return subResult{}
}

func (a *addBootstrap) getAPIKeyString(ctx context.Context, instance *crd.EMQX) (string, error) {
	var bootstrapAPIKeys string

	for _, apiKey := range instance.Spec.BootstrapAPIKeys {
		if apiKey.SecretRef != nil {
			keyValue, err := a.readSecret(ctx, instance, apiKey.SecretRef.Key.SecretName, apiKey.SecretRef.Key.SecretKey)
			if err != nil {
				return "", err
			}
			secretValue, err := a.readSecret(ctx, instance, apiKey.SecretRef.Secret.SecretName, apiKey.SecretRef.Secret.SecretKey)
			if err != nil {
				return "", err
			}
			bootstrapAPIKeys += keyValue + ":" + secretValue + "\n"
		} else {
			bootstrapAPIKeys += apiKey.Key + ":" + apiKey.Secret + "\n"
		}
	}

	return bootstrapAPIKeys, nil
}

func (a *addBootstrap) readSecret(ctx context.Context, instance *crd.EMQX, name string, key string) (string, error) {
	secret := &corev1.Secret{}
	if err := a.Client.Get(ctx, instance.NamespacedName(name), secret); err != nil {
		return "", emperror.Wrap(err, "failed to get secret")
	}

	if _, ok := secret.Data[key]; !ok {
		return "", emperror.NewWithDetails("secret does not contain the key", "secret", secret.Name, "key", key)
	}

	return string(secret.Data[key]), nil
}

func generateBootstrapAPIKeySecret(instance *crd.EMQX, bootstrapAPIKeys string) *corev1.Secret {
	defPassword, _ := password.Generate(64, 10, 0, true, true)
	bootstrapAPIKeys += resources.DefaultBootstrapAPIKey + ":" + defPassword
	return resources.BootstrapAPIKey(instance).Secret(bootstrapAPIKeys)
}

func generateNodeCookieSecret(instance *crd.EMQX, conf *config.EMQX) *corev1.Secret {
	cookie := conf.GetNodeCookie()
	if cookie == "" {
		cookie, _ = password.Generate(64, 10, 0, true, true)
	}
	return resources.Cookie(instance).Secret(cookie)
}
