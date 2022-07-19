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

package apps

import (
	"context"
	"encoding/base64"
	"fmt"
	"reflect"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	"github.com/emqx/emqx-operator/pkg/handler"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Check EMQX Custom Resource", func() {
	var aclString, brokerModulesString, enterpriseModulesString string
	var headlessPort corev1.ServicePort
	var ports, pluginPorts []corev1.ServicePort
	var pluginList []string
	var license []byte

	Context("check resource", func() {
		BeforeEach(func() {
			aclString = "{allow, {user, \"dashboard\"}, subscribe, [\"$SYS/#\"]}.\n{allow, {ipaddr, \"127.0.0.1\"}, pubsub, [\"$SYS/#\", \"#\"]}.\n{deny, all, subscribe, [\"$SYS/#\", {eq, \"#\"}]}.\n{allow, all}.\n"
			brokerModulesString = "{emqx_mod_acl_internal, true}.\n"
			enterpriseModulesString = `[{"name":"internal_acl","enable":true,"configs":{"acl_rule_file":"/mounted/acl/acl.conf"}},{"name":"retainer","enable":true,"configs":{"expiry_interval":0,"max_payload_size":"1MB","max_retained_messages":0,"storage_type":"ram"}}]`
			headlessPort = corev1.ServicePort{
				Name:       "http-management-8081",
				Port:       8081,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt(8081),
			}
			ports = []corev1.ServicePort{
				{
					Name:       "mqtt-tcp-1883",
					Port:       1883,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(1883),
				},
				{
					Name:       "mqtt-ssl-8883",
					Port:       8883,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(8883),
				},
				{
					Name:       "mqtt-ws-8083",
					Port:       8083,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(8083),
				},
				{
					Name:       "mqtt-wss-8084",
					Port:       8084,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(8084),
				},
			}
			pluginPorts = []corev1.ServicePort{
				{
					Name:       "lwm2m-udp-5683",
					Protocol:   corev1.ProtocolUDP,
					Port:       5683,
					TargetPort: intstr.FromInt(5683),
				},
				{
					Name:       "lwm2m-udp-5684",
					Protocol:   corev1.ProtocolUDP,
					Port:       5684,
					TargetPort: intstr.FromInt(5684),
				},
				{
					Name:       "lwm2m-dtls-5685",
					Protocol:   corev1.ProtocolUDP,
					Port:       5685,
					TargetPort: intstr.FromInt(5685),
				},
				{
					Name:       "lwm2m-dtls-5686",
					Protocol:   corev1.ProtocolUDP,
					Port:       5686,
					TargetPort: intstr.FromInt(5686),
				},
			}
			pluginList = []string{"emqx_management", "emqx_dashboard", "emqx_rule_engine", "emqx_retainer", "emqx_lwm2m"}
		})

		It("check default resource", func() {
			By("should create a configMap with ACL")
			check_acl(broker, aclString)
			check_acl(enterprise, aclString)

			By("should create a configMap with loaded modules")
			check_modules(broker, brokerModulesString)
			check_modules(enterprise, enterpriseModulesString)

			By("should create a service")
			check_service_ports(broker, append(ports, pluginPorts...), headlessPort)
			check_service_ports(enterprise, append(ports, pluginPorts...), headlessPort)

			By("should create a statefulSet", func() {
				check_statefulset(broker)
				check_statefulset(enterprise)
			})

			By("should mount volume correctly")
			check_sts_volume(broker)
			check_sts_volume(enterprise)

			By("should create EMQX Plugins")
			check_plugin(broker, pluginList)
			check_plugin(enterprise, append(pluginList, "emqx_modules"))

			By("should not create secret with license")
			Eventually(func() bool {
				secret := &corev1.Secret{}
				err := k8sClient.Get(
					context.Background(),
					types.NamespacedName{
						Namespace: enterprise.GetNamespace(),
						Name:      fmt.Sprintf("%s-license", enterprise.GetName()),
					},
					secret,
				)
				return k8sErrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("check resource when update some filed", func() {
		BeforeEach(func() {
			aclString = "{deny, all}.\n"
			brokerModulesString = "{emqx_mod_presence, false}.\n"
			enterpriseModulesString = `[{"name":"internal_acl","enable":true,"configs":{"acl_rule_file":"/mounted/acl/acl.conf"}}]`
			ports = []corev1.ServicePort{
				{
					Name:       "mqtt-tcp-11883",
					Port:       11883,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(11883),
				},
				{
					Name:       "mqtt-tcp-21883",
					Port:       21883,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(21883),
				},
				{
					Name:       "mqtt-ssl-8883",
					Port:       8883,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(8883),
				},
				{
					Name:       "mqtt-ws-8083",
					Port:       8083,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(8083),
				},
				{
					Name:       "mqtt-wss-8084",
					Port:       8084,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(8084),
				},
			}
			pluginPorts = []corev1.ServicePort{
				{
					Name:       "lwm2m-dtls-5685",
					Protocol:   corev1.ProtocolUDP,
					Port:       5685,
					TargetPort: intstr.FromInt(5685),
				},
				{
					Name:       "lwm2m-dtls-5686",
					Protocol:   corev1.ProtocolUDP,
					Port:       5686,
					TargetPort: intstr.FromInt(5686),
				},
				{
					Name:       "lwm2m-udp-5687",
					Protocol:   corev1.ProtocolUDP,
					Port:       5687,
					TargetPort: intstr.FromInt(5687),
				},
				{
					Name:       "lwm2m-udp-5688",
					Protocol:   corev1.ProtocolUDP,
					Port:       5688,
					TargetPort: intstr.FromInt(5688),
				},
			}
			license = []byte(`-----BEGIN CERTIFICATE-----
MIIENzCCAx+gAwIBAgIDdMvVMA0GCSqGSIb3DQEBBQUAMIGDMQswCQYDVQQGEwJD
TjERMA8GA1UECAwIWmhlamlhbmcxETAPBgNVBAcMCEhhbmd6aG91MQwwCgYDVQQK
DANFTVExDDAKBgNVBAsMA0VNUTESMBAGA1UEAwwJKi5lbXF4LmlvMR4wHAYJKoZI
hvcNAQkBFg96aGFuZ3doQGVtcXguaW8wHhcNMjAwNjIwMDMwMjUyWhcNNDkwMTAx
MDMwMjUyWjBjMQswCQYDVQQGEwJDTjEZMBcGA1UECgwQRU1RIFggRXZhbHVhdGlv
bjEZMBcGA1UEAwwQRU1RIFggRXZhbHVhdGlvbjEeMBwGCSqGSIb3DQEJARYPY29u
dGFjdEBlbXF4LmlvMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArw+3
2w9B7Rr3M7IOiMc7OD3Nzv2KUwtK6OSQ07Y7ikDJh0jynWcw6QamTiRWM2Ale8jr
0XAmKgwUSI42+f4w84nPpAH4k1L0zupaR10VYKIowZqXVEvSyV8G2N7091+6Jcon
DcaNBqZLRe1DiZXMJlhXnDgq14FPAxffKhCXiCgYtluLDDLKv+w9BaQGZVjxlFe5
cw32+z/xHU366npHBpafCbxBtWsNvchMVtLBqv9yPmrMqeBROyoJaI3nL78xDgpd
cRorqo+uQ1HWdcM6InEFET6pwkeuAF8/jJRlT12XGgZKKgFQTCkZi4hv7aywkGBE
JruPif/wlK0YuPJu6QIDAQABo4HSMIHPMBEGCSsGAQQBg5odAQQEDAIxMDCBlAYJ
KwYBBAGDmh0CBIGGDIGDZW1xeF9iYWNrZW5kX3JlZGlzLGVtcXhfYmFja2VuZF9t
eXNxbCxlbXF4X2JhY2tlbmRfcGdzcWwsZW1xeF9iYWNrZW5kX21vbmdvLGVtcXhf
YmFja2VuZF9jYXNzYSxlbXF4X2JyaWRnZV9rYWZrYSxlbXF4X2JyaWRnZV9yYWJi
aXQwEAYJKwYBBAGDmh0DBAMMATEwEQYJKwYBBAGDmh0EBAQMAjEwMA0GCSqGSIb3
DQEBBQUAA4IBAQDHUe6+P2U4jMD23u96vxCeQrhc/rXWvpmU5XB8Q/VGnJTmv3yU
EPyTFKtEZYVX29z16xoipUE6crlHhETOfezYsm9K0DxF3fNilOLRKkg9VEWcb5hj
iL3a2tdZ4sq+h/Z1elIXD71JJBAImjr6BljTIdUCfVtNvxlE8M0D/rKSn2jwzsjI
UrW88THMtlz9sb56kmM3JIOoIJoep6xNEajIBnoChSGjtBYFNFwzdwSTCodYkgPu
JifqxTKSuwAGSlqxJUwhjWG8ulzL3/pCAYEwlWmd2+nsfotQdiANdaPnez7o0z0s
EujOCZMbK8qNfSbyo50q5iIXhz2ZIGl+4hdp
-----END CERTIFICATE-----`)

			var config v1beta3.EmqxConfig

			config = broker.GetEmqxConfig()
			config["listener.tcp.internal"] = "11883"
			config["listener.tcp.external"] = "21883"
			broker.SetEmqxConfig(config)
			broker.SetACL([]string{`{deny, all}.`})
			broker.SetModules([]v1beta3.EmqxBrokerModule{
				{
					Name:   "emqx_mod_presence",
					Enable: false,
				},
			})
			updateEmqx(broker)
			update_lwm2m(broker)

			config = enterprise.GetEmqxConfig()
			config["listener.tcp.internal"] = "11883"
			config["listener.tcp.external"] = "21883"
			enterprise.SetEmqxConfig(config)
			enterprise.SetACL([]string{`{deny, all}.`})
			enterprise.SetModules([]v1beta3.EmqxEnterpriseModule{
				{
					Name:    "internal_acl",
					Enable:  true,
					Configs: runtime.RawExtension{Raw: []byte(`{"acl_rule_file": "/mounted/acl/acl.conf"}`)},
				},
			})
			enterprise.Spec.EmqxTemplate.License.Data = license
			updateEmqx(enterprise)
			update_lwm2m(enterprise)
		})
		It("", func() {
			By("should create a configMap with ACL")
			check_acl(broker, aclString)
			check_acl(enterprise, aclString)

			By("should create a configMap with loaded modules")
			check_modules(broker, brokerModulesString)
			check_modules(enterprise, enterpriseModulesString)

			By("should create a service")
			check_service_ports(broker, append(ports, pluginPorts...), headlessPort)
			check_service_ports(enterprise, append(ports, pluginPorts...), headlessPort)

			By("should create a secret with license")
			Eventually(func() map[string][]byte {
				secret := &corev1.Secret{}
				_ = k8sClient.Get(
					context.Background(),
					types.NamespacedName{
						Namespace: enterprise.GetNamespace(),
						Name:      fmt.Sprintf("%s-license", enterprise.GetName()),
					},
					secret,
				)
				return secret.Data
			}, timeout, interval).Should(Equal(
				map[string][]byte{"emqx.lic": license}),
			)
		})

		AfterEach(func() {
			Eventually(func() error {
				plugin := &v1beta3.EmqxPlugin{}
				err := k8sClient.Get(
					context.Background(),
					types.NamespacedName{
						Name:      fmt.Sprintf("%s-%s", broker.GetName(), "lwm2m"),
						Namespace: broker.GetNamespace(),
					}, plugin,
				)
				if err != nil {
					if k8sErrors.IsNotFound(err) {
						return nil
					}
					return err
				}
				return k8sClient.Delete(context.Background(), plugin)
			}, timeout, interval).Should(Succeed())

			By("should delete plugin resource")

			Eventually(func() string {
				cm := &corev1.ConfigMap{}
				_ = k8sClient.Get(
					context.Background(),
					types.NamespacedName{
						Name:      fmt.Sprintf("%s-%s", broker.GetName(), "loaded-plugins"),
						Namespace: broker.GetNamespace(),
					}, cm,
				)
				return cm.Data["loaded_plugins"]
			}, timeout, interval).ShouldNot(ContainSubstring("emqx_lwm2m"))

			Eventually(func() []corev1.ServicePort {
				svc := &corev1.Service{}
				_ = k8sClient.Get(
					context.Background(),
					types.NamespacedName{
						Name:      broker.GetName(),
						Namespace: broker.GetNamespace(),
					},
					svc,
				)
				return svc.Spec.Ports
			}, timeout, interval).ShouldNot(ContainElements(pluginPorts))
		})
	})
})

func update_lwm2m(emqx v1beta3.Emqx) {
	Eventually(func() error {
		plugin := &v1beta3.EmqxPlugin{}
		err := k8sClient.Get(
			context.Background(),
			types.NamespacedName{
				Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "lwm2m"),
				Namespace: emqx.GetNamespace(),
			}, plugin,
		)
		if err != nil {
			return err
		}
		plugin.Spec.Config["lwm2m.bind.udp.1"] = "0.0.0.0:5687"
		plugin.Spec.Config["lwm2m.bind.udp.2"] = "0.0.0.0:5688"
		return k8sClient.Update(context.Background(), plugin)
	}, timeout, interval).Should(Succeed())
}

func check_statefulset(emqx v1beta3.Emqx) {
	sts := &appsv1.StatefulSet{}
	Eventually(func() error {
		err := k8sClient.Get(
			context.TODO(),
			types.NamespacedName{
				Name:      emqx.GetName(),
				Namespace: emqx.GetNamespace(),
			},
			sts,
		)
		return err
	}, timeout, interval).Should(Succeed())

	Expect(sts.Spec.Replicas).Should(Equal(emqx.GetReplicas()))
	Expect(sts.Spec.Template.Labels).Should(Equal(emqx.GetLabels()))
	Expect(sts.Spec.Template.Spec.Affinity).Should(Equal(emqx.GetAffinity()))
	Expect(sts.Spec.Template.Spec.Tolerations).Should(Equal(emqx.GetToleRations()))
	Expect(sts.Spec.Template.Spec.Containers).Should(HaveLen(2))
	Expect(sts.Spec.Template.Spec.Containers[0].ImagePullPolicy).Should(Equal(corev1.PullIfNotPresent))
	Expect(sts.Spec.Template.Spec.Containers[0].Resources).Should(Equal(emqx.GetResource()))
	Expect(sts.Spec.Template.Spec.Containers[0].Args).Should(Equal(emqx.GetArgs()))
	Expect(sts.Spec.Template.Spec.Containers[1].Args).Should(Equal([]string{"-u", "admin", "-p", "public", "-P", "8081"}))
	Expect(sts.Spec.Template.Spec.SecurityContext.FSGroup).Should(Equal(emqx.GetSecurityContext().FSGroup))
	Expect(sts.Spec.Template.Spec.SecurityContext.RunAsUser).Should(Equal(emqx.GetSecurityContext().RunAsUser))
	Expect(sts.Spec.Template.Spec.SecurityContext.SupplementalGroups).Should(Equal(emqx.GetSecurityContext().SupplementalGroups))
	if emqx.GetInitContainers() != nil {
		Expect(sts.Spec.Template.Spec.InitContainers[0].Name).Should(Equal(emqx.GetInitContainers()[0].Name))
		Expect(sts.Spec.Template.Spec.InitContainers[0].Image).Should(Equal(emqx.GetInitContainers()[0].Image))
		Expect(sts.Spec.Template.Spec.InitContainers[0].Args).Should(Equal(emqx.GetInitContainers()[0].Args))
	}
}

func check_service_ports(emqx v1beta3.Emqx, ports []corev1.ServicePort, headlessPort corev1.ServicePort) {
	Eventually(func() []corev1.ServicePort {
		svc := &corev1.Service{}
		_ = k8sClient.Get(
			context.Background(),
			types.NamespacedName{
				Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "headless"),
				Namespace: emqx.GetNamespace(),
			},
			svc,
		)
		return svc.Spec.Ports
	}, timeout, interval).Should(ContainElements(headlessPort))
	Eventually(func() []corev1.ServicePort {
		svc := &corev1.Service{}
		_ = k8sClient.Get(
			context.Background(),
			types.NamespacedName{
				Name:      emqx.GetName(),
				Namespace: emqx.GetNamespace(),
			},
			svc,
		)
		return svc.Spec.Ports
	}, timeout, interval).Should(ContainElements(append(ports, headlessPort)))
}

func check_acl(emqx v1beta3.Emqx, aclString string) {
	Eventually(func() map[string]string {
		cm := &corev1.ConfigMap{}
		_ = k8sClient.Get(
			context.Background(),
			types.NamespacedName{
				Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "acl"),
				Namespace: emqx.GetNamespace(),
			},
			cm,
		)
		return cm.Data
	}, timeout, interval).Should(Equal(
		map[string]string{"acl.conf": aclString},
	))

	Eventually(func() map[string]string {
		sts := &appsv1.StatefulSet{}
		_ = k8sClient.Get(
			context.Background(),
			types.NamespacedName{
				Name:      emqx.GetName(),
				Namespace: emqx.GetNamespace(),
			},
			sts,
		)
		return sts.Annotations
	}, timeout, interval).Should(
		HaveKeyWithValue(
			"ACL/Base64EncodeConfig",
			base64.StdEncoding.EncodeToString([]byte(aclString)),
		),
	)
}

func check_modules(emqx v1beta3.Emqx, loadedModulesString string) {
	Eventually(func() map[string]string {
		cm := &corev1.ConfigMap{}
		_ = k8sClient.Get(
			context.Background(),
			types.NamespacedName{
				Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "loaded-modules"),
				Namespace: emqx.GetNamespace(),
			}, cm,
		)
		return cm.Data
	}, timeout, interval).Should(Equal(
		map[string]string{"loaded_modules": loadedModulesString},
	))

	Eventually(func() map[string]string {
		sts := &appsv1.StatefulSet{}
		_ = k8sClient.Get(
			context.Background(),
			types.NamespacedName{
				Name:      emqx.GetName(),
				Namespace: emqx.GetNamespace(),
			},
			sts,
		)
		return sts.Annotations
	}, timeout, interval).Should(
		HaveKeyWithValue(
			"LoadedModules/Base64EncodeConfig",
			base64.StdEncoding.EncodeToString([]byte(loadedModulesString)),
		),
	)
}

func check_plugin(emqx v1beta3.Emqx, pluginList []string) {
	Eventually(func() []string {
		list := v1beta3.EmqxPluginList{}
		_ = k8sClient.List(
			context.Background(),
			&list,
			client.InNamespace(emqx.GetNamespace()),
			client.MatchingLabels(emqx.GetLabels()),
		)
		l := []string{}
		for _, plugin := range list.Items {
			l = append(l, plugin.Spec.PluginName)
		}
		return l
	}, timeout, interval).Should(ConsistOf(pluginList))

	for _, pluginName := range pluginList {
		Eventually(func() map[string]string {
			cm := &corev1.ConfigMap{}
			_ = k8sClient.Get(
				context.Background(),
				types.NamespacedName{
					Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "plugins-config"),
					Namespace: emqx.GetNamespace(),
				}, cm,
			)
			return cm.Data
		}, timeout, interval).Should(HaveKey(pluginName + ".conf"))

		Eventually(func() string {
			cm := &corev1.ConfigMap{}
			_ = k8sClient.Get(
				context.Background(),
				types.NamespacedName{
					Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "loaded-plugins"),
					Namespace: emqx.GetNamespace(),
				}, cm,
			)
			return cm.Data["loaded_plugins"]
		}, timeout, interval).Should(ContainSubstring(pluginName))
	}
}

func check_sts_volume(emqx v1beta3.Emqx) {
	names := v1beta3.Names{Object: emqx}
	LoadedPlugins := names.LoadedPlugins()
	PluginsConfig := names.PluginsConfig()
	acl, module, data, log := names.ACL(), names.LoadedModules(), names.Data(), names.Log()
	defaultVolumeList := []string{LoadedPlugins, PluginsConfig, acl, module, data, log}
	if emqxEnterprise, ok := emqx.(*v1beta3.EmqxEnterprise); ok {
		if !reflect.ValueOf(emqxEnterprise.GetLicense()).IsZero() {
			defaultVolumeList = append(defaultVolumeList, names.License())
		}
	}
	Eventually(func() []string {
		sts := &appsv1.StatefulSet{}
		_ = k8sClient.Get(
			context.TODO(),
			types.NamespacedName{
				Name:      emqx.GetName(),
				Namespace: emqx.GetNamespace(),
			},
			sts,
		)
		volumeList := []string{}
		for _, v := range sts.Spec.Template.Spec.Volumes {
			volumeList = append(volumeList, v.Name)
		}
		return volumeList
	}, timeout, interval).Should(ContainElements(defaultVolumeList))

	Eventually(func() []string {
		sts := &appsv1.StatefulSet{}
		_ = k8sClient.Get(
			context.TODO(),
			types.NamespacedName{
				Name:      emqx.GetName(),
				Namespace: emqx.GetNamespace(),
			},
			sts,
		)
		emqxContainer := findEmqxContainer(sts.Spec.Template.Spec.Containers)
		volumeMountList := []string{}
		for _, v := range emqxContainer.VolumeMounts {
			volumeMountList = append(volumeMountList, v.Name)
		}

		return volumeMountList
	}, timeout, interval).Should(ContainElements(defaultVolumeList))
}

func findEmqxContainer(containers []corev1.Container) corev1.Container {
	for _, c := range containers {
		if c.Name == handler.EmqxContainerName {
			return c
		}
	}
	return corev1.Container{}
}
