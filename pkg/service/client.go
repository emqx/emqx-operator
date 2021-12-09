package service

import (
	"reflect"

	"github.com/emqx/emqx-operator/api/v1beta1"
	"github.com/emqx/emqx-operator/pkg/client/k8s"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EmqxClient interface {
	EnsureEmqxNamespace(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxSecret(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxRBAC(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxHeadlessService(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxListenerService(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxConfigMapForAcl(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxConfigMapForLoadedModules(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxConfigMapForLoadedPlugins(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxStatefulSet(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
}

type Client struct {
	k8s.Manager
}

func NewClient(manager k8s.Manager) *Client {
	return &Client{manager}
}

func (client *Client) EnsureEmqxSecret(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	emqxEnterprise, ok := emqx.(*v1beta1.EmqxEnterprise)
	if ok && emqxEnterprise.GetLicense() != "" {
		new := NewSecretForCR(*emqxEnterprise, labels, ownerRefs)
		old, err := client.GetSecret(emqx.GetNamespace(), emqx.GetSecretName())
		if err != nil {
			if errors.IsNotFound(err) {
				return client.CreateSecret(new)
			}
			return err
		}

		if new.StringData["emqx.lic"] != old.StringData["emqx.lic"] {
			new.ResourceVersion = old.ResourceVersion
			if err := client.UpdateSecret(new); err != nil {
				return err
			}
			// TODO reload emqx.lic
			return nil
		}
	}
	return nil
}

func (client *Client) EnsureEmqxRBAC(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	sa, role, roleBinding := NewRBAC(emqx, labels, ownerRefs)

	if _, err := client.GetServiceAccount(
		emqx.GetNamespace(),
		emqx.GetName(),
	); err != nil {
		if errors.IsNotFound(err) {
			return client.CreateServiceAccount(sa)
		}
		return err
	}

	if _, err := client.GetRole(
		emqx.GetNamespace(),
		emqx.GetName(),
	); err != nil {
		if errors.IsNotFound(err) {
			return client.CreateRole(role)
		}
		return err
	}

	if _, err := client.GetRoleBinding(
		emqx.GetNamespace(),
		emqx.GetName(),
	); err != nil {
		if errors.IsNotFound(err) {
			return client.CreateRoleBinding(roleBinding)
		}
		return err
	}

	return nil
}

func (client *Client) EnsureEmqxHeadlessService(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	_, err := client.GetService(emqx.GetNamespace(), emqx.GetHeadlessServiceName())
	if err != nil {
		if errors.IsNotFound(err) {
			svc := NewHeadLessSvcForCR(emqx, labels, ownerRefs)
			return client.CreateService(svc)
		}
		return err
	}
	return nil
}

func (client *Client) EnsureEmqxListenerService(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	_, err := client.GetService(emqx.GetNamespace(), emqx.GetName())
	if err != nil {
		if errors.IsNotFound(err) {
			listenerSvc := NewListenerSvcForCR(emqx, labels, ownerRefs)
			return client.CreateService(listenerSvc)
		}
		return err
	}
	return nil
}

func (client *Client) EnsureEmqxConfigMapForAcl(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	new := NewConfigMapForAcl(emqx, labels, ownerRefs)
	old, err := client.GetConfigMap(new.Namespace, new.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return client.CreateConfigMap(new)
		} else {
			return err
		}
	}

	if new.Data["acl.conf"] != old.Data["acl.conf"] {
		new.ResourceVersion = old.ResourceVersion
		if err := client.UpdateConfigMap(new); err != nil {
			return err
		}
		//TODO restart emqx pods
		return nil
	}
	return nil
}

func (client *Client) EnsureEmqxConfigMapForLoadedModules(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	new := NewConfigMapForLoadedModules(emqx, labels, ownerRefs)
	old, err := client.GetConfigMap(new.Namespace, new.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return client.CreateConfigMap(new)
		} else {
			return err
		}
	}

	if new.Data["loaded_modules"] != old.Data["loaded_modules"] {
		new.ResourceVersion = old.ResourceVersion
		if err := client.UpdateConfigMap(new); err != nil {
			return err
		}
		//TODO restart emqx pods
		return err
	}
	return nil
}

func (client *Client) EnsureEmqxConfigMapForLoadedPlugins(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	new := NewConfigMapForLoadedPlugins(emqx, labels, ownerRefs)
	old, err := client.GetConfigMap(new.Namespace, new.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return client.CreateConfigMap(new)
		} else {
			return err
		}
	}

	if new.Data["loaded_plugins"] != old.Data["loaded_plugins"] {
		new.ResourceVersion = old.ResourceVersion
		if err := client.UpdateConfigMap(new); err != nil {
			//TODO restart emqx pods
			return nil
		}
		return err
	}
	return nil
}

func (client *Client) EnsureEmqxStatefulSet(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	new := NewEmqxStatefulSet(emqx, labels, ownerRefs)
	old, err := client.GetStatefulSet(emqx.GetNamespace(), emqx.GetName())
	if err != nil {
		if errors.IsNotFound(err) {
			return client.CreateStatefulSet(new)
		}
		return err
	}

	if broker, ok := emqx.(*v1beta1.EmqxBroker); ok {
		if oldBroker, err := client.GetEmqxBroker(
			emqx.GetNamespace(),
			emqx.GetName(),
		); err != nil {
			if reflect.DeepEqual(oldBroker.Spec, broker.Spec) {
				new.ResourceVersion = old.ResourceVersion
				return client.UpdateStatefulSet(new)
			}
		}
	}

	if enterprise, ok := emqx.(*v1beta1.EmqxEnterprise); ok {
		if oldEnterprise, err := client.GetEmqxEnterprise(
			emqx.GetNamespace(),
			emqx.GetName(),
		); err != nil {
			if reflect.DeepEqual(oldEnterprise.Spec, enterprise.Spec) {
				new.ResourceVersion = old.ResourceVersion
				return client.UpdateStatefulSet(new)
			}
		}
	}

	return nil
}
