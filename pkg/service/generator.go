package service

import (
	"github.com/emqx/emqx-operator/api/v1alpha1"
	"github.com/emqx/emqx-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// func generateEmqxConfigMap(emqx *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.ConfigMap {
// 	name := util.GetEmqxName(emqx)
// 	namespace := emqx.Namespace

// }

func newSecretForCR(emqx *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.Secret {
	stringData := map[string]string{"emqx.lic": emqx.Spec.License}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            util.GetEmqxSecret(emqx),
			Namespace:       emqx.Namespace,
			OwnerReferences: ownerRefs,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: stringData,
	}

	return secret

}

func newHeadLessSvcForCR(emqx *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.Service {
	emqxPorts := []corev1.ServicePort{
		{
			Name:     SERVICE_TCP_NAME,
			Port:     SERVICE_TCP_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: SERVICE_TCP_PORT,
			},
		},
		{
			Name:     SERVICE_TCPS_NAME,
			Port:     SERVICE_TCPS_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: SERVICE_TCPS_PORT,
			},
		},
		{
			Name:     SERVICE_WS_NAME,
			Port:     SERVICE_WS_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: SERVICE_WS_PORT,
			},
		},
		{
			Name:     SERVICE_WSS_NAME,
			Port:     SERVICE_WSS_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: SERVICE_WSS_PORT,
			},
		},
		{
			Name:     SERVICE_DASHBOARD_NAME,
			Port:     SERVICE_DASHBOARD_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: SERVICE_DASHBOARD_PORT,
			},
		},
	}
	// labels = util.MergeLabels(labels, generateSelectorLabels(util.SentinelRoleName, cluster.Name))
	labels = map[string]string{}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            util.GetEmqxHeadlessSvc(emqx),
			Namespace:       emqx.Namespace,
			OwnerReferences: ownerRefs,
		},
		Spec: corev1.ServiceSpec{
			Ports:     emqxPorts,
			Selector:  labels,
			ClusterIP: corev1.ClusterIPNone,
		},
	}

	return svc
}

func newConfigMapForLoadedMoudles(emqx *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.ConfigMap {
	data := map[string]string{"loaded_modules": emqx.Spec.LoadedModulesConf}
	cmForPM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            util.GetEmqxConfigMapForLM(emqx),
			Namespace:       emqx.Namespace,
			OwnerReferences: ownerRefs,
		},
		Data: data,
	}

	return cmForPM
}

func newConfigMapForLoadedPlugins(emqx *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.ConfigMap {
	data := map[string]string{"loaded_plugins": emqx.Spec.LoadedPluginConf}
	cmForPG := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            util.GetEmqxConfigMapForPG(emqx),
			Namespace:       emqx.Namespace,
			OwnerReferences: ownerRefs,
		},
		Data: data,
	}

	return cmForPG
}
