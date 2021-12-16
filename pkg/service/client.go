package service

import (
	"reflect"

	"github.com/emqx/emqx-operator/api/v1beta1"
	"github.com/emqx/emqx-operator/pkg/client/k8s"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EmqxClient interface {
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
		old, err := client.Secret.Get(emqx.GetNamespace(), emqx.GetSecretName())
		if err != nil {
			if errors.IsNotFound(err) {
				return client.Secret.Create(new)
			}
			return err
		}

		if new.StringData["emqx.lic"] != old.StringData["emqx.lic"] {
			new.ResourceVersion = old.ResourceVersion
			if err := client.Secret.Update(new); err != nil {
				return err
			}
			// TODO Use the emqx api to reload the license instead of restarting the pod
			return nil
		}
	}
	return nil
}

func (client *Client) EnsureEmqxRBAC(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	sa, role, roleBinding := NewRBAC(emqx, labels, ownerRefs)

	if _, err := client.ServiceAccount.Get(
		sa.Namespace,
		sa.Name,
	); err != nil {
		if errors.IsNotFound(err) {
			if err := client.ServiceAccount.Create(sa); err != nil {
				return err
			}
		}
		return err
	}

	existRoleBinding := false
	roleBindingList, err := client.RoleBinding.List(emqx.GetNamespace())
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if len(roleBindingList.Items) != 0 {
		for _, roleBinding := range roleBindingList.Items {
			for _, subject := range roleBinding.Subjects {
				if subject.Kind == "ServiceAccount" && subject.Name == emqx.GetServiceAccountName() {
					existRoleBinding = true
					role.Name = roleBinding.RoleRef.Name
				}
			}
		}
	}

	if _, err := client.Role.Get(
		role.Namespace,
		role.Name,
	); err != nil {
		if errors.IsNotFound(err) {
			if err := client.Role.Create(role); err != nil {
				return err
			}
		}
		return err
	}

	if !existRoleBinding {
		if err := client.RoleBinding.Create(roleBinding); err != nil {
			return err
		}
	}

	return nil
}

func (client *Client) EnsureEmqxHeadlessService(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	_, err := client.Service.Get(emqx.GetNamespace(), emqx.GetHeadlessServiceName())
	if err != nil {
		if errors.IsNotFound(err) {
			svc := NewHeadLessSvcForCR(emqx, labels, ownerRefs)
			return client.Service.Create(svc)
		}
		return err
	}
	return nil
}

func (client *Client) EnsureEmqxListenerService(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	_, err := client.Service.Get(emqx.GetNamespace(), emqx.GetName())
	if err != nil {
		if errors.IsNotFound(err) {
			listenerSvc := NewListenerSvcForCR(emqx, labels, ownerRefs)
			return client.Service.Create(listenerSvc)
		}
		return err
	}
	return nil
}

func (client *Client) EnsureEmqxConfigMapForAcl(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	new := NewConfigMapForAcl(emqx, labels, ownerRefs)
	old, err := client.ConfigMap.Get(new.Namespace, new.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return client.ConfigMap.Create(new)
		} else {
			return err
		}
	}

	if new.Data["acl.conf"] != old.Data["acl.conf"] {
		new.ResourceVersion = old.ResourceVersion
		if err := client.ConfigMap.Update(new); err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (client *Client) EnsureEmqxConfigMapForLoadedModules(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	new := NewConfigMapForLoadedModules(emqx, labels, ownerRefs)
	old, err := client.ConfigMap.Get(new.Namespace, new.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return client.ConfigMap.Create(new)
		} else {
			return err
		}
	}

	if new.Data["loaded_modules"] != old.Data["loaded_modules"] {
		new.ResourceVersion = old.ResourceVersion
		if err := client.ConfigMap.Update(new); err != nil {
			return err
		}
		return err
	}
	return nil
}

func (client *Client) EnsureEmqxConfigMapForLoadedPlugins(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	new := NewConfigMapForLoadedPlugins(emqx, labels, ownerRefs)
	old, err := client.ConfigMap.Get(new.Namespace, new.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return client.ConfigMap.Create(new)
		} else {
			return err
		}
	}

	if new.Data["loaded_plugins"] != old.Data["loaded_plugins"] {
		new.ResourceVersion = old.ResourceVersion
		if err := client.ConfigMap.Update(new); err != nil {
			//TODO Use the emqx api to reload the license instead of restarting the pod
			return nil
		}
		return err
	}
	return nil
}

func (client *Client) EnsureEmqxStatefulSet(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	new := NewEmqxStatefulSet(emqx, labels, ownerRefs)
	old, err := client.StatefulSet.Get(emqx.GetNamespace(), emqx.GetName())
	if err != nil {
		if errors.IsNotFound(err) {
			return client.StatefulSet.Create(new)
		}
		return err
	}

	if broker, ok := emqx.(*v1beta1.EmqxBroker); ok {
		if oldBroker, err := client.EmqxBroker.Get(
			emqx.GetNamespace(),
			emqx.GetName(),
		); err == nil {
			if !reflect.DeepEqual(oldBroker.Spec, broker.Spec) {
				new.ResourceVersion = old.ResourceVersion
				return client.StatefulSet.Update(new)
			}
		}
	}

	if enterprise, ok := emqx.(*v1beta1.EmqxEnterprise); ok {
		if oldEnterprise, err := client.EmqxEnterprise.Get(
			emqx.GetNamespace(),
			emqx.GetName(),
		); err == nil {
			if !reflect.DeepEqual(oldEnterprise.Spec, enterprise.Spec) {
				new.ResourceVersion = old.ResourceVersion
				return client.StatefulSet.Update(new)
			}
		}
	}

	return nil
}
