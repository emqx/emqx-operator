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

package v1beta4

import (
	"path/filepath"
	"strings"
	"testing"

	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

var instance = appsv1beta4.EmqxEnterprise{
	ObjectMeta: metav1.ObjectMeta{
		Name:        "emqx",
		Namespace:   "default",
		Labels:      map[string]string{"foo": "bar"},
		Annotations: map[string]string{"foo": "bar"},
	},
}

func TestGenerateInitPluginList(t *testing.T) {
	emqx := instance.DeepCopy()
	existPluginList := &appsv1beta4.EmqxPluginList{}
	existPluginList.Items = append(
		existPluginList.Items,
		appsv1beta4.EmqxPlugin{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake",
				Namespace: "default",
			},
		},
	)

	expectPlugin := &appsv1beta4.EmqxPlugin{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps.emqx.io/v1beta4",
			Kind:       "EmqxPlugin",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   "default",
			Labels:      map[string]string{"foo": "bar"},
			Annotations: map[string]string{"foo": "bar"},
		},
		Spec: appsv1beta4.EmqxPluginSpec{
			Selector: map[string]string{"foo": "bar"},
			Config:   map[string]string{},
		},
	}

	t.Run("default eviction agent", func(t *testing.T) {
		plugin := expectPlugin.DeepCopy()
		plugin.Name = "emqx-eviction-agent"
		plugin.Spec.PluginName = "emqx_eviction_agent"
		assert.Contains(t, generateInitPluginList(emqx, existPluginList), plugin)

		existPluginList.Items = append(existPluginList.Items, *plugin)
		assert.NotContains(t, generateInitPluginList(emqx, existPluginList), plugin)
	})

	t.Run("default node rebalance", func(t *testing.T) {
		plugin := expectPlugin.DeepCopy()
		plugin.Name = "emqx-node-rebalance"
		plugin.Spec.PluginName = "emqx_node_rebalance"
		assert.Contains(t, generateInitPluginList(emqx, existPluginList), plugin)

		existPluginList.Items = append(existPluginList.Items, *plugin)
		assert.NotContains(t, generateInitPluginList(emqx, existPluginList), plugin)
	})

	t.Run("default rule engine", func(t *testing.T) {
		plugin := expectPlugin.DeepCopy()
		plugin.Name = "emqx-rule-engine"
		plugin.Spec.PluginName = "emqx_rule_engine"
		assert.Contains(t, generateInitPluginList(emqx, existPluginList), plugin)

		existPluginList.Items = append(existPluginList.Items, *plugin)
		assert.NotContains(t, generateInitPluginList(emqx, existPluginList), plugin)
	})

	t.Run("default retainer", func(t *testing.T) {
		plugin := expectPlugin.DeepCopy()
		plugin.Name = "emqx-retainer"
		plugin.Spec.PluginName = "emqx_retainer"
		assert.Contains(t, generateInitPluginList(emqx, existPluginList), plugin)

		existPluginList.Items = append(existPluginList.Items, *plugin)
		assert.NotContains(t, generateInitPluginList(emqx, existPluginList), plugin)
	})

	t.Run("default modules", func(t *testing.T) {
		plugin := expectPlugin.DeepCopy()
		plugin.Name = "emqx-modules"
		plugin.Spec.PluginName = "emqx_modules"
		assert.Contains(t, generateInitPluginList(emqx, existPluginList), plugin)

		existPluginList.Items = append(existPluginList.Items, *plugin)
		assert.NotContains(t, generateInitPluginList(emqx, existPluginList), plugin)
	})
}

func TestGenerateDefaultPluginsConfig(t *testing.T) {
	got := generateDefaultPluginsConfig(&instance)
	assert.Equal(t, got.ObjectMeta, metav1.ObjectMeta{
		Name:        "emqx-plugins-config",
		Namespace:   "default",
		Labels:      map[string]string{"foo": "bar"},
		Annotations: map[string]string{"foo": "bar"},
	})
	assert.Len(t, got.Data, 47)
}

func TestGenerateLicense(t *testing.T) {
	emqx := instance.DeepCopy()
	assert.Nil(t, generateLicense(emqx))

	emqx.Spec.License = appsv1beta4.EmqxLicense{
		Data:       []byte("fake data"),
		StringData: string("fake string data"),
	}

	expect := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "emqx-license",
			Namespace:   "default",
			Labels:      map[string]string{"foo": "bar"},
			Annotations: map[string]string{"foo": "bar"},
		},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{"emqx.lic": []byte("fake data")},
		StringData: map[string]string{"emqx.lic": "fake string data"},
	}

	assert.Equal(t, generateLicense(emqx), expect)
}

func TestGenerateEmqxACL(t *testing.T) {
	emqx := instance.DeepCopy()
	emqx.Spec.Template.Spec.EmqxContainer.EmqxACL = []string{
		`{allow, {user, "dashboard"}, subscribe, ["$SYS/#"]}.`,
		`{allow, {ipaddr, "127.0.0.1"}, pubsub, ["$SYS/#", "#"]}.`,
		`{deny, all, subscribe, ["$SYS/#", {eq, "#"}]}.`,
		`{allow, all}.`,
	}
	expect := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "emqx-acl",
			Namespace:   "default",
			Labels:      map[string]string{"foo": "bar"},
			Annotations: map[string]string{"foo": "bar"},
		},
		Data: map[string]string{"acl.conf": "{allow, {user, \"dashboard\"}, subscribe, [\"$SYS/#\"]}.\n{allow, {ipaddr, \"127.0.0.1\"}, pubsub, [\"$SYS/#\", \"#\"]}.\n{deny, all, subscribe, [\"$SYS/#\", {eq, \"#\"}]}.\n{allow, all}.\n"},
	}
	assert.Equal(t, generateEmqxACL(emqx), expect)
}

func TestGenerateStatefulSet(t *testing.T) {
	emqx := instance.DeepCopy()
	emqx.Default()

	emqx.Spec.Replicas = pointer.Int32(3)
	emqx.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
		{
			Name: "fake",
		},
	}
	emqx.Spec.Template.Spec.EmqxContainer.Image.Repository = "emqx/emqx-ee"
	emqx.Spec.Template.Spec.EmqxContainer.Image.Version = "latest"
	emqx.Spec.Template.Spec.EmqxContainer.Image.PullPolicy = corev1.PullAlways
	emqx.Spec.Template.Spec.EmqxContainer.EnvFrom = []corev1.EnvFromSource{
		{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "fake",
				},
			},
		},
	}
	emqx.Spec.Template.Spec.EmqxContainer.Env = []corev1.EnvVar{
		{
			Name:  "Foo",
			Value: "Bar",
		},
	}
	emqx.Spec.Template.Spec.EmqxContainer.VolumeMounts = []corev1.VolumeMount{
		{
			Name:      "fake",
			MountPath: "/fake",
		},
	}
	emqx.Spec.Template.Spec.ExtraContainers = []corev1.Container{
		{
			Name: "fake",
		},
	}
	emqx.Spec.Template.Spec.InitContainers = []corev1.Container{
		{
			Name: "fake",
		},
	}
	emqx.Spec.Template.Spec.Volumes = []corev1.Volume{
		{
			Name: "fake",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}

	got := generateStatefulSet(emqx)
	assert.Equal(t, []corev1.Volume{
		{
			Name: "fake",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "emqx-log",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "emqx-data",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}, got.Spec.Template.Spec.Volumes)

	emqx.Spec.Persistent = &corev1.PersistentVolumeClaimTemplate{
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: pointer.String("fake"),
		},
	}
	emqx.Default()

	got = generateStatefulSet(emqx)
	assert.Equal(t, emqx.Spec.Persistent.ObjectMeta, got.Spec.VolumeClaimTemplates[0].ObjectMeta)
	assert.Equal(t, emqx.Spec.Persistent.Spec, got.Spec.VolumeClaimTemplates[0].Spec)

	assert.Equal(t, "emqx", got.Name)
	assert.Equal(t, "default", got.Namespace)
	assert.Equal(t, map[string]string{
		"apps.emqx.io/instance":   "emqx",
		"apps.emqx.io/managed-by": "emqx-operator",
		"foo":                     "bar",
	}, got.Labels)
	assert.Equal(t, map[string]string{
		"foo": "bar",
	}, got.Annotations)

	assert.Equal(t, emqx.Spec.Replicas, got.Spec.Replicas)
	assert.Equal(t, "emqx-headless", got.Spec.ServiceName)
	assert.Equal(t, map[string]string{
		"apps.emqx.io/instance":   "emqx",
		"apps.emqx.io/managed-by": "emqx-operator",
		"foo":                     "bar",
	}, got.Spec.Selector.MatchLabels)
	assert.Equal(t, appsv1.ParallelPodManagement, got.Spec.PodManagementPolicy)

	assert.Equal(t, map[string]string{
		"apps.emqx.io/instance":   "emqx",
		"apps.emqx.io/managed-by": "emqx-operator",
		"foo":                     "bar",
	}, got.Spec.Template.Labels)
	assert.Equal(t, map[string]string{
		"apps.emqx.io/manage-containers": "emqx,reloader,fake",
		"foo":                            "bar",
	}, got.Spec.Template.Annotations)
	assert.Equal(t, emqx.Spec.Template.Spec.Affinity, got.Spec.Template.Spec.Affinity)
	assert.Equal(t, emqx.Spec.Template.Spec.Tolerations, got.Spec.Template.Spec.Tolerations)
	assert.Equal(t, emqx.Spec.Template.Spec.NodeName, got.Spec.Template.Spec.NodeName)
	assert.Equal(t, emqx.Spec.Template.Spec.NodeSelector, got.Spec.Template.Spec.NodeSelector)
	assert.Equal(t, "fake", got.Spec.Template.Spec.ImagePullSecrets[0].Name)
	assert.Equal(t, "fake", got.Spec.Template.Spec.InitContainers[0].Name)
	assert.Equal(t, emqx.Spec.Template.Spec.PodSecurityContext, got.Spec.Template.Spec.SecurityContext)

	assert.Equal(t, "emqx", got.Spec.Template.Spec.Containers[0].Name)
	assert.Equal(t, "emqx/emqx-ee:latest", got.Spec.Template.Spec.Containers[0].Image)
	assert.Equal(t, corev1.PullAlways, got.Spec.Template.Spec.Containers[0].ImagePullPolicy)
	assert.Equal(t, []corev1.EnvFromSource{
		{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "fake",
				},
			},
		},
	}, got.Spec.Template.Spec.Containers[0].EnvFrom)
	assert.Equal(t, []corev1.EnvVar{
		{
			Name:  "EMQX_CLUSTER__DISCOVERY",
			Value: "dns",
		},
		{
			Name:  "EMQX_CLUSTER__DNS__APP",
			Value: "emqx",
		},
		{
			Name:  "EMQX_CLUSTER__DNS__NAME",
			Value: "emqx-headless.default.svc.cluster.local",
		},
		{
			Name:  "EMQX_CLUSTER__DNS__TYPE",
			Value: "srv",
		},
		{
			Name:  "EMQX_LISTENER__TCP__INTERNAL",
			Value: "",
		},
		{
			Name:  "EMQX_LOG__TO",
			Value: "console",
		},
		{
			Name:  "EMQX_NAME",
			Value: "emqx",
		},
		{
			Name:  "Foo",
			Value: "Bar",
		},
	}, got.Spec.Template.Spec.Containers[0].Env)

	assert.Equal(t, []corev1.VolumeMount{
		{
			Name:      "fake",
			MountPath: "/fake",
		},
		{
			Name:      "emqx-log",
			MountPath: "/opt/emqx/log",
		},
		{
			Name:      "emqx-data",
			MountPath: "/opt/emqx/data",
		},
	}, got.Spec.Template.Spec.Containers[0].VolumeMounts)
	assert.Equal(t, emqx.Spec.Template.Spec.EmqxContainer.Resources, got.Spec.Template.Spec.Containers[0].Resources)
	assert.Equal(t, emqx.Spec.Template.Spec.EmqxContainer.SecurityContext, got.Spec.Template.Spec.Containers[0].SecurityContext)
	assert.Equal(t, emqx.Spec.Template.Spec.EmqxContainer.LivenessProbe, got.Spec.Template.Spec.Containers[0].LivenessProbe)
	assert.Equal(t, emqx.Spec.Template.Spec.EmqxContainer.ReadinessProbe, got.Spec.Template.Spec.Containers[0].ReadinessProbe)
	assert.Equal(t, emqx.Spec.Template.Spec.EmqxContainer.StartupProbe, got.Spec.Template.Spec.Containers[0].StartupProbe)
	assert.Equal(t, emqx.Spec.Template.Spec.EmqxContainer.Lifecycle, got.Spec.Template.Spec.Containers[0].Lifecycle)
	assert.Equal(t, emqx.Spec.Template.Spec.EmqxContainer.TerminationMessagePath, got.Spec.Template.Spec.Containers[0].TerminationMessagePath)
	assert.Equal(t, emqx.Spec.Template.Spec.EmqxContainer.TerminationMessagePolicy, got.Spec.Template.Spec.Containers[0].TerminationMessagePolicy)
	assert.Equal(t, emqx.Spec.Template.Spec.EmqxContainer.Image.PullPolicy, got.Spec.Template.Spec.Containers[0].ImagePullPolicy)
	assert.Equal(t, emqx.Spec.Template.Spec.EmqxContainer.Stdin, got.Spec.Template.Spec.Containers[0].Stdin)
	assert.Equal(t, emqx.Spec.Template.Spec.EmqxContainer.StdinOnce, got.Spec.Template.Spec.Containers[0].StdinOnce)
	assert.Equal(t, emqx.Spec.Template.Spec.EmqxContainer.TTY, got.Spec.Template.Spec.Containers[0].TTY)
	assert.Equal(t, emqx.Spec.Template.Spec.EmqxContainer.SecurityContext, got.Spec.Template.Spec.Containers[0].SecurityContext)
	assert.Equal(t, emqx.Spec.Template.Spec.EmqxContainer.Command, got.Spec.Template.Spec.Containers[0].Command)
	assert.Equal(t, emqx.Spec.Template.Spec.EmqxContainer.Args, got.Spec.Template.Spec.Containers[0].Args)

	assert.Equal(t, "reloader", got.Spec.Template.Spec.Containers[1].Name)
	assert.Equal(t, "emqx/emqx-operator-reloader:0.0.2", got.Spec.Template.Spec.Containers[1].Image)
	assert.Equal(t, corev1.PullAlways, got.Spec.Template.Spec.Containers[1].ImagePullPolicy)
	assert.Equal(t, []corev1.EnvVar{
		{
			Name:  "EMQX_CLUSTER__DISCOVERY",
			Value: "dns",
		},
		{
			Name:  "EMQX_CLUSTER__DNS__APP",
			Value: "emqx",
		},
		{
			Name:  "EMQX_CLUSTER__DNS__NAME",
			Value: "emqx-headless.default.svc.cluster.local",
		},
		{
			Name:  "EMQX_CLUSTER__DNS__TYPE",
			Value: "srv",
		},
		{
			Name:  "EMQX_LISTENER__TCP__INTERNAL",
			Value: "",
		},
		{
			Name:  "EMQX_LOG__TO",
			Value: "console",
		},
		{
			Name:  "EMQX_NAME",
			Value: "emqx",
		},
		{
			Name:  "Foo",
			Value: "Bar",
		},
	}, got.Spec.Template.Spec.Containers[1].Env)
	assert.Equal(t, []corev1.EnvFromSource{
		{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "fake",
				},
			},
		},
	}, got.Spec.Template.Spec.Containers[1].EnvFrom)
	assert.Equal(t, []corev1.VolumeMount{
		{
			Name:      "fake",
			MountPath: "/fake",
		},
	}, got.Spec.Template.Spec.Containers[1].VolumeMounts)
	assert.Equal(t, []string{
		"-u", "admin",
		"-p", "public",
		"-P", "8081",
	}, got.Spec.Template.Spec.Containers[1].Args)

	assert.Equal(t, []corev1.PodReadinessGate{
		{
			ConditionType: appsv1beta4.PodOnServing,
		},
	}, got.Spec.Template.Spec.ReadinessGates)
}

func TestUpdateStatefulSetForPluginsConfig(t *testing.T) {
	sts := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "emqx",
						},
						{
							Name: "reloader",
						},
					},
				},
			},
		},
	}

	pluginsConfig := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "emqx-plugins-config",
		},
	}

	updateStatefulSetForPluginsConfig(sts, pluginsConfig)

	assert.Equal(t, []corev1.Volume{
		{
			Name: pluginsConfig.Name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: pluginsConfig.Name,
					},
				},
			},
		},
	}, sts.Spec.Template.Spec.Volumes)

	assert.Equal(t, []corev1.VolumeMount{
		{
			Name:      pluginsConfig.Name,
			MountPath: "/mounted/plugins/etc",
		},
	}, sts.Spec.Template.Spec.Containers[0].VolumeMounts)

	assert.Equal(t, []corev1.VolumeMount{
		{
			Name:      pluginsConfig.Name,
			MountPath: "/mounted/plugins/etc",
		},
	}, sts.Spec.Template.Spec.Containers[1].VolumeMounts)

	assert.Equal(t, []corev1.EnvVar{
		{
			Name:  "EMQX_PLUGINS__ETC_DIR",
			Value: "/mounted/plugins/etc",
		},
	}, sts.Spec.Template.Spec.Containers[0].Env)

	assert.Equal(t, []corev1.EnvVar{
		{
			Name:  "EMQX_PLUGINS__ETC_DIR",
			Value: "/mounted/plugins/etc",
		},
	}, sts.Spec.Template.Spec.Containers[1].Env)
}

func TestUpdateStatefulSetForLicense(t *testing.T) {
	sts := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "emqx",
						},
						{
							Name: "reloader",
						},
					},
				},
			},
		},
	}

	license := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "emqx-license",
		},
		Data: map[string][]byte{
			"license": []byte("fake"),
		},
	}

	updateStatefulSetForLicense(sts, license)
	assert.Equal(t, []corev1.Volume{
		{
			Name: license.Name,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: license.Name,
				},
			},
		},
	}, sts.Spec.Template.Spec.Volumes)

	assert.Equal(t, []corev1.VolumeMount{
		{
			Name:      license.Name,
			MountPath: "/mounted/license",
			ReadOnly:  true,
		},
	}, sts.Spec.Template.Spec.Containers[0].VolumeMounts)

	assert.Equal(t, []corev1.VolumeMount{
		{
			Name:      license.Name,
			MountPath: "/mounted/license",
			ReadOnly:  true,
		},
	}, sts.Spec.Template.Spec.Containers[1].VolumeMounts)

	assert.Equal(t, []corev1.EnvVar{
		{
			Name:  "EMQX_LICENSE__FILE",
			Value: filepath.Join("/mounted/license", "license"),
		},
	}, sts.Spec.Template.Spec.Containers[0].Env)

	assert.Equal(t, []corev1.EnvVar{
		{
			Name:  "EMQX_LICENSE__FILE",
			Value: filepath.Join("/mounted/license", "license"),
		},
	}, sts.Spec.Template.Spec.Containers[1].Env)
}

func TestUpdateStatefulSetForACL(t *testing.T) {
	sts := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "emqx",
						},
						{
							Name: "reloader",
						},
					},
				},
			},
		},
	}

	acl := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "emqx-acl",
		},
	}

	updateStatefulSetForACL(sts, acl)

	assert.Equal(t, []corev1.Volume{
		{
			Name: acl.Name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: acl.Name,
					},
				},
			},
		},
	}, sts.Spec.Template.Spec.Volumes)

	assert.Equal(t, []corev1.VolumeMount{
		{
			Name:      acl.Name,
			MountPath: "/mounted/acl",
		},
	}, sts.Spec.Template.Spec.Containers[0].VolumeMounts)

	assert.Equal(t, []corev1.VolumeMount{
		{
			Name:      acl.Name,
			MountPath: "/mounted/acl",
		},
	}, sts.Spec.Template.Spec.Containers[1].VolumeMounts)

	assert.Equal(t, []corev1.EnvVar{
		{
			Name:  "EMQX_ACL_FILE",
			Value: "/mounted/acl/acl.conf",
		},
	}, sts.Spec.Template.Spec.Containers[0].Env)

	assert.Equal(t, []corev1.EnvVar{
		{
			Name:  "EMQX_ACL_FILE",
			Value: "/mounted/acl/acl.conf",
		},
	}, sts.Spec.Template.Spec.Containers[1].Env)
}

func TestGenerateBootstrapUserSecret(t *testing.T) {
	instance := &appsv1beta4.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: appsv1beta4.EmqxBrokerSpec{
			Template: appsv1beta4.EmqxTemplate{
				Spec: appsv1beta4.EmqxTemplateSpec{
					EmqxContainer: appsv1beta4.EmqxContainer{
						BootstrapAPIKeys: []appsv1beta4.BootstrapAPIKey{
							{
								Key:    "test_key",
								Secret: "secret",
							},
						},
					},
				},
			},
		},
	}

	got := generateBootstrapUserSecret(instance)
	assert.Equal(t, "emqx-bootstrap-user", got.Name)
	data, ok := got.StringData["bootstrap_user"]
	assert.True(t, ok)

	users := strings.Split(data, "\n")
	var usernames []string
	for _, user := range users {
		usernames = append(usernames, user[:strings.Index(user, ":")])
	}
	assert.ElementsMatch(t, usernames, []string{defUsername, "test_key"})
}

func TestUpdateStatefulSetForBootstrapUser(t *testing.T) {
	bootstrapUser := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "emqx-bootstrap-user",
		},
	}

	sts := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "emqx"},
						{Name: "reloader"},
					},
				},
			},
		},
	}

	got := updateStatefulSetForBootstrapUser(sts, bootstrapUser)

	assert.Equal(t, []corev1.Volume{{
		Name: "bootstrap-user",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: "emqx-bootstrap-user",
			},
		},
	}}, got.Spec.Template.Spec.Volumes)

	assert.Equal(t, []corev1.VolumeMount{{
		Name:      "bootstrap-user",
		MountPath: "/opt/emqx/data/bootstrap_user",
		SubPath:   "bootstrap_user",
		ReadOnly:  true,
	}}, got.Spec.Template.Spec.Containers[0].VolumeMounts)

	assert.Equal(t, []corev1.VolumeMount{{
		Name:      "bootstrap-user",
		MountPath: "/opt/emqx/data/bootstrap_user",
		SubPath:   "bootstrap_user",
		ReadOnly:  true,
	}}, got.Spec.Template.Spec.Containers[1].VolumeMounts)

	assert.Equal(t, []corev1.EnvVar{{
		Name:  "EMQX_MANAGEMENT__BOOTSTRAP_APPS_FILE",
		Value: "/opt/emqx/data/bootstrap_user",
	}}, got.Spec.Template.Spec.Containers[0].Env)

	assert.Equal(t, []corev1.EnvVar{{
		Name:  "EMQX_MANAGEMENT__BOOTSTRAP_APPS_FILE",
		Value: "/opt/emqx/data/bootstrap_user",
	}}, got.Spec.Template.Spec.Containers[1].Env)

}

func TestGenerateHeadlessService(t *testing.T) {
	emqx := instance.DeepCopy()

	t.Run("headless service", func(t *testing.T) {
		got := generateHeadlessService(emqx)
		assert.Equal(t, "emqx-headless", got.Name)
		assert.Equal(t, emqx.GetNamespace(), got.Namespace)
		assert.Equal(t, emqx.GetLabels(), got.Labels)
		assert.Equal(t, emqx.GetAnnotations(), got.Annotations)

		assert.Equal(t, emqx.GetLabels(), got.Spec.Selector)
		assert.Equal(t, corev1.ServiceTypeClusterIP, got.Spec.Type)
		assert.Equal(t, corev1.ClusterIPNone, got.Spec.ClusterIP)
		assert.Equal(t, true, got.Spec.PublishNotReadyAddresses)
		assert.Equal(t, []corev1.ServicePort{
			{
				Name:       "http-management-8081",
				Port:       8081,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt(8081),
			},
		}, got.Spec.Ports)
	})
}
