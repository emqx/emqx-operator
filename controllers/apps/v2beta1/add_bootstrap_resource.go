package v2beta1

import (
	"context"

	emperror "emperror.dev/errors"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	"github.com/rory-z/go-hocon"
	"github.com/sethvargo/go-password/password"
)

type addBootstrap struct {
	*EMQXReconciler
}

func (a *addBootstrap) reconcile(ctx context.Context, instance *appsv2beta1.EMQX, _ innerReq.RequesterInterface) subResult {
	for _, resource := range []client.Object{
		generateNodeCookieSecret(instance),
		generateBootstrapAPIKeySecret(a.Client, ctx, instance),
	} {
		if err := ctrl.SetControllerReference(instance, resource, a.Scheme); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to set controller reference")}
		}
		if err := a.Create(resource); err != nil {
			if !k8sErrors.IsAlreadyExists(err) {
				return subResult{err: emperror.Wrap(err, "failed to create bootstrap configMap")}
			}
		}
	}

	return subResult{}
}

func generateNodeCookieSecret(instance *appsv2beta1.EMQX) *corev1.Secret {
	var cookie string

	config, _ := hocon.ParseString(instance.Spec.Config.Data)
	cookie = config.GetString("node.cookie")
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

// ReadSecret reads a secret from the Kubernetes cluster.
func ReadSecret(k8sClient client.Client, ctx context.Context, namespace string, name string, key string) (string, error) {
	// Define a new Secret object
	secret := &corev1.Secret{}

	// Define the Secret Name and Namespace
	secretName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	// Use the client to fetch the Secret
	if err := k8sClient.Get(ctx, secretName, secret); err != nil {
		return "", err
	}

	// secret.Data is a map[string][]byte
	secretValue := string(secret.Data[key])

	return secretValue, nil
}

func generateBootstrapAPIKeySecret(k8sClient client.Client, ctx context.Context, instance *appsv2beta1.EMQX) *corev1.Secret {
	logger := log.FromContext(ctx)
	bootstrapAPIKeys := ""

	for _, apiKey := range instance.Spec.BootstrapAPIKeys {
		if apiKey.SecretRef != nil {
			logger.V(1).Info("Read SecretRef")

			// Read key and secret values from the refenced secrets
			keyValue, err := ReadSecret(k8sClient, ctx, instance.Namespace, apiKey.SecretRef.Key.SecretName, apiKey.SecretRef.Key.SecretKey)
			if err != nil {
				logger.V(1).Error(err, "read secretRef", "key")
				continue
			}
			secretValue, err := ReadSecret(k8sClient, ctx, instance.Namespace, apiKey.SecretRef.Secret.SecretName, apiKey.SecretRef.Secret.SecretKey)
			if err != nil {
				logger.V(1).Error(err, "read secretRef", "secret")
				continue
			}
			bootstrapAPIKeys += keyValue + ":" + secretValue + "\n"
		} else {
			bootstrapAPIKeys += apiKey.Key + ":" + apiKey.Secret + "\n"
		}
	}
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
