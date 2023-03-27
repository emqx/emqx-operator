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
	"context"
	"fmt"
	"sort"

	"github.com/emqx/emqx-operator/apis/apps/v1beta4"
	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	appscontrollersv1beta4 "github.com/emqx/emqx-operator/controllers/apps/v1beta4"
	"github.com/emqx/emqx-operator/internal/handler"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var emqxBroker = &appsv1beta4.EmqxBroker{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "emqx",
		Namespace: "e2e-test-v1beta4",
		Labels: map[string]string{
			"test": "e2e",
		},
	},
	Spec: appsv1beta4.EmqxBrokerSpec{
		Replicas: &[]int32{1}[0],
		Template: appsv1beta4.EmqxTemplate{
			Spec: appsv1beta4.EmqxTemplateSpec{
				EmqxContainer: appsv1beta4.EmqxContainer{
					Image: appsv1beta4.EmqxImage{
						Repository: "emqx",
						Version:    "4.4.15",
					},
					EmqxConfig: appsv1beta4.EmqxConfig{
						"sysmon.long_schedule": "240h",
					},
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

var emqxEnterprise = &appsv1beta4.EmqxEnterprise{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "emqx-ee",
		Namespace: "e2e-test-v1beta4",
		Labels: map[string]string{
			"test": "e2e",
		},
	},
	Spec: appsv1beta4.EmqxEnterpriseSpec{
		Replicas: &[]int32{1}[0],
		Template: appsv1beta4.EmqxTemplate{
			Spec: appsv1beta4.EmqxTemplateSpec{
				EmqxContainer: appsv1beta4.EmqxContainer{
					Image: appsv1beta4.EmqxImage{
						Repository: "emqx/emqx-ee",
						Version:    "4.4.15",
					},
					EmqxConfig: appsv1beta4.EmqxConfig{
						"sysmon.long_schedule": "240h",
					},
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

var lwm2m = &appsv1beta4.EmqxPlugin{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "apps.emqx.io/v1beta3",
		Kind:       "EmqxPlugin",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "lwm2m",
		Namespace: "e2e-test-v1beta4",
		Labels: map[string]string{
			"test": "e2e",
		},
	},
	Spec: appsv1beta4.EmqxPluginSpec{
		PluginName: "emqx_lwm2m",
		Selector: map[string]string{
			"test": "e2e",
		},
		Config: map[string]string{
			"lwm2m.lifetime_min": "1s",
			"lwm2m.lifetime_max": "86400s",
			"lwm2m.bind.udp.1":   "0.0.0.0:5685",
			"lwm2m.bind.udp.2":   "0.0.0.0:5686",
			"lwm2m.bind.dtls.1":  "0.0.0.0:5687",
			"lwm2m.bind.dtls.2":  "0.0.0.0:5688",
			"lwm2m.xml_dir":      "/opt/emqx/etc/lwm2m_xml",
		},
	},
}

var _ = Describe("Base E2E Test", func() {
	DescribeTable("",
		func(emqx appsv1beta4.Emqx, plugin *appsv1beta4.EmqxPlugin) {
			var pluginList []string
			var pluginPorts []corev1.ServicePort
			var ports []corev1.ServicePort
			var headlessPort corev1.ServicePort

			pluginList = []string{"emqx_eviction_agent", "emqx_node_rebalance", "emqx_rule_engine", "emqx_retainer", "emqx_lwm2m"}
			if _, ok := emqx.(*appsv1beta4.EmqxEnterprise); ok {
				pluginList = append(pluginList, "emqx_modules")
			}

			pluginPorts = []corev1.ServicePort{
				{
					Name:       "lwm2m-udp-5685",
					Protocol:   corev1.ProtocolUDP,
					Port:       5685,
					TargetPort: intstr.FromInt(5685),
				},
				{
					Name:       "lwm2m-udp-5686",
					Protocol:   corev1.ProtocolUDP,
					Port:       5686,
					TargetPort: intstr.FromInt(5686),
				},
				{
					Name:       "lwm2m-dtls-5687",
					Protocol:   corev1.ProtocolUDP,
					Port:       5687,
					TargetPort: intstr.FromInt(5687),
				},
				{
					Name:       "lwm2m-dtls-5688",
					Protocol:   corev1.ProtocolUDP,
					Port:       5688,
					TargetPort: intstr.FromInt(5688),
				},
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

			headlessPort = corev1.ServicePort{
				Name:       "http-management-8081",
				Port:       8081,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt(8081),
			}
			By("create EMQX CR")
			createEmqx(emqx)

			By("create EMQX Plugin")
			plugin.Labels = emqx.GetLabels()
			plugin.Namespace = emqx.GetNamespace()
			Expect(k8sClient.Create(context.TODO(), plugin)).Should(Succeed())

			By("check EMQX CR status")
			Expect(emqx.GetStatus().GetReplicas()).Should(Equal(int32(1)))
			Expect(emqx.GetStatus().GetReadyReplicas()).Should(Equal(int32(1)))
			Expect(emqx.GetStatus().GetEmqxNodes()).Should(HaveLen(1))
			Expect(emqx.GetStatus().GetCurrentStatefulSetVersion()).ShouldNot(BeEmpty())
			Expect(emqx.GetStatus().GetConditions()).ShouldNot(BeEmpty())

			By("check pod annotations")
			sts := &appsv1.StatefulSet{}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), sts)).Should(Succeed())
			Expect(sts.Spec.Template.Annotations).Should(HaveKey(handler.ManageContainersAnnotation))

			By("Checking the EMQX Pod Conditions")
			Eventually(func() []corev1.PodStatus {
				pods := &corev1.PodList{}
				_ = k8sClient.List(context.TODO(), pods,
					client.InNamespace(emqx.GetNamespace()),
					client.MatchingLabels(emqx.GetLabels()),
				)
				s := []corev1.PodStatus{}
				for _, pod := range pods.Items {
					if pod.Status.Phase == corev1.PodRunning {
						s = append(s, pod.Status)
					}
				}
				return s
			}, timeout, interval).Should(HaveEach(
				HaveField("Conditions", ContainElements(
					HaveField("Type", v1beta4.PodInCluster),
					HaveField("Type", corev1.PodReady),
					HaveField("Type", v1beta4.PodOnServing),
				))))

			By("Checking the EMQX Custom Resource's EndpointSlice")
			ep := &discoveryv1.EndpointSlice{}
			Eventually(func() []discoveryv1.Endpoint {
				_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: emqx.GetSpec().GetServiceTemplate().Name, Namespace: emqx.GetSpec().GetServiceTemplate().Namespace}, ep)
				return ep.Endpoints
			}, timeout, interval).Should(HaveLen(3))

			Eventually(func() []discoveryv1.EndpointPort {
				_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: emqx.GetSpec().GetServiceTemplate().Name, Namespace: emqx.GetSpec().GetServiceTemplate().Namespace}, ep)
				return ep.Ports
			}, timeout, interval).Should(ConsistOf([]discoveryv1.EndpointPort{
				{
					Name:     &[]string{ports[0].Name}[0],
					Port:     &[]int32{ports[0].Port}[0],
					Protocol: &[]corev1.Protocol{ports[0].Protocol}[0],
				},
				{
					Name:     &[]string{ports[1].Name}[0],
					Port:     &[]int32{ports[1].Port}[0],
					Protocol: &[]corev1.Protocol{ports[1].Protocol}[0],
				},
			}))

			By("check plugins")
			Eventually(func() []string {
				list := appsv1beta4.EmqxPluginList{}
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
			}

			By("check headless service ports")
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
			}, timeout, interval).Should(Equal(headlessPort))

			By("check service ports")
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
			}, timeout, interval).Should(ContainElements(append(pluginPorts, append(ports, headlessPort)...)))

			By("update EMQX Plugin")
			Eventually(func() error {
				p := &appsv1beta4.EmqxPlugin{}
				err := k8sClient.Get(
					context.Background(),
					types.NamespacedName{
						Name:      plugin.GetName(),
						Namespace: plugin.GetNamespace(),
					},
					p,
				)
				if err != nil {
					return err
				}
				p.Spec.Config["lwm2m.bind.udp.1"] = "0.0.0.0:5695"
				p.Spec.Config["lwm2m.bind.udp.2"] = "0.0.0.0:5696"
				p.Spec.Config["lwm2m.bind.dtls.1"] = "0.0.0.0:5697"
				p.Spec.Config["lwm2m.bind.dtls.2"] = "0.0.0.0:5698"
				return k8sClient.Update(context.Background(), p)
			}, timeout, interval).Should(Succeed())

			pluginPorts = []corev1.ServicePort{
				{
					Name:       "lwm2m-udp-5695",
					Protocol:   corev1.ProtocolUDP,
					Port:       5695,
					TargetPort: intstr.FromInt(5695),
				},
				{
					Name:       "lwm2m-udp-5696",
					Protocol:   corev1.ProtocolUDP,
					Port:       5696,
					TargetPort: intstr.FromInt(5696),
				},
				{
					Name:       "lwm2m-dtls-5697",
					Protocol:   corev1.ProtocolUDP,
					Port:       5697,
					TargetPort: intstr.FromInt(5697),
				},
				{
					Name:       "lwm2m-dtls-5698",
					Protocol:   corev1.ProtocolUDP,
					Port:       5698,
					TargetPort: intstr.FromInt(5698),
				},
			}

			By("check service ports")
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
			}, timeout, interval).Should(ContainElements(append(pluginPorts, append(ports, headlessPort)...)))

			By("delete EMQX CR and EMQX Plugin")
			deleteEmqx(emqx)
		},
		Entry(nil, emqxBroker.DeepCopy(), lwm2m.DeepCopy()),
		Entry(nil, emqxEnterprise.DeepCopy(), lwm2m.DeepCopy()),
	)
})

var _ = Describe("Blue Green Update Test", Label("blue"), func() {
	Describe("Just check enterprise", func() {
		emqx := emqxEnterprise.DeepCopy()
		emqx.Spec.Template.Spec.EmqxContainer.Image.Version = "4.4.15"
		emqx.Spec.EmqxBlueGreenUpdate = &appsv1beta4.EmqxBlueGreenUpdate{
			InitialDelaySeconds: 5,
			EvacuationStrategy: appsv1beta4.EvacuationStrategy{
				WaitTakeover:  int32(0),
				ConnEvictRate: int32(1),
				SessEvictRate: int32(1),
			},
		}

		BeforeEach(func() {
			createEmqx(emqx)
		})

		AfterEach(func() {
			deleteEmqx(emqx)
		})

		It("blue green update", func() {
			var existedStsList *appsv1.StatefulSetList
			existedStsList = &appsv1.StatefulSetList{}
			Eventually(func() []appsv1.StatefulSet {
				_ = k8sClient.List(
					context.TODO(),
					existedStsList,
					client.InNamespace(emqx.GetNamespace()),
					client.MatchingLabels(emqx.GetLabels()),
				)
				return existedStsList.Items
			}, timeout, interval).Should(HaveLen(1))

			sts := existedStsList.Items[0].DeepCopy()
			Eventually(func() string {
				// Wait sts ready
				_ = k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Name:      sts.GetName(),
						Namespace: sts.GetNamespace(),
					},
					sts,
				)
				return sts.Status.CurrentRevision
			}, timeout, interval).ShouldNot(BeEmpty())

			By("check currentStatefulSetVersion in CR status")
			Eventually(func() string {
				ee := &appsv1beta4.EmqxEnterprise{}
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), ee)
				return ee.Status.CurrentStatefulSetVersion
			}, timeout, interval).Should(Equal(sts.Status.CurrentRevision))

			By("check emqx nodes in CR status")
			Eventually(func() string {
				ee := &appsv1beta4.EmqxEnterprise{}
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), ee)
				if len(ee.GetStatus().GetEmqxNodes()) > 0 {
					return ee.GetStatus().GetEmqxNodes()[0].Node
				}
				return ""
			}, timeout, interval).Should(Equal(fmt.Sprintf("emqx-ee@%s-0.emqx-ee-headless.%s.svc.cluster.local", sts.Name, emqx.GetNamespace())))

			By("check running condition in CR status")
			Eventually(func() corev1.ConditionStatus {
				ee := &appsv1beta4.EmqxEnterprise{}
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), ee)
				if ee.GetStatus().GetConditions()[0].Type == appsv1beta4.ConditionRunning {
					return ee.GetStatus().GetConditions()[0].Status
				}
				return corev1.ConditionUnknown
			}, timeout, interval).Should(Equal(corev1.ConditionTrue))

			By("update EMQX CR")
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), emqx)).Should(Succeed())
			emqx.Spec.Template.Spec.Volumes = append(emqx.Spec.Template.Spec.Volumes, corev1.Volume{
				Name: "test-blue-green-update",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			})
			Expect(k8sClient.Update(context.Background(), emqx)).Should(Succeed())

			By("wait create new sts")
			existedStsList = &appsv1.StatefulSetList{}
			Eventually(func() []appsv1.StatefulSet {
				_ = k8sClient.List(
					context.TODO(),
					existedStsList,
					client.InNamespace(emqx.GetNamespace()),
					client.MatchingLabels(emqx.GetLabels()),
				)
				return existedStsList.Items
			}, timeout, interval).Should(HaveLen(2))

			allSts := []*appsv1.StatefulSet{}
			for _, es := range existedStsList.Items {
				allSts = append(allSts, es.DeepCopy())
			}
			sort.Sort(appscontrollersv1beta4.StatefulSetsBySizeNewer(allSts))

			newSts := allSts[0].DeepCopy()
			Expect(newSts.UID).ShouldNot(Equal(sts.UID))
			Eventually(func() string {
				// Wait sts ready
				_ = k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Name:      newSts.GetName(),
						Namespace: newSts.GetNamespace(),
					},
					newSts,
				)
				return sts.Status.CurrentRevision
			}, timeout, interval).ShouldNot(BeEmpty())

			By("check emqx nodes in CR status")
			Eventually(func() []appsv1beta4.EmqxNode {
				ee := &appsv1beta4.EmqxEnterprise{}
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), ee)
				return ee.GetStatus().GetEmqxNodes()
			}, timeout, interval).Should(HaveLen(2))

			By("check readyReplicas in CR status")
			Eventually(func() int {
				ee := &appsv1beta4.EmqxEnterprise{}
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), ee)
				return int(ee.Status.ReadyReplicas)
			}, timeout, interval).Should(Equal(2))

			By("check blue-green status in CR status")
			blueGreenStatus := &appsv1beta4.EmqxBlueGreenUpdateStatus{}
			Eventually(func() bool {
				ee := &appsv1beta4.EmqxEnterprise{}
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), ee)
				if ee.Status.EmqxBlueGreenUpdateStatus != nil &&
					ee.Status.EmqxBlueGreenUpdateStatus.EvacuationsStatus != nil &&
					ee.Status.EmqxBlueGreenUpdateStatus.StartedAt != nil {
					blueGreenStatus = ee.Status.EmqxBlueGreenUpdateStatus.DeepCopy()
					return true
				}
				return false
			}, timeout, interval).Should(BeTrue())
			Expect(blueGreenStatus.OriginStatefulSet).Should(Equal(sts.Name))
			Expect(blueGreenStatus.CurrentStatefulSet).Should(Equal(newSts.Name))

			By("check blue-green condition in CR status")
			Eventually(func() corev1.ConditionStatus {
				ee := &appsv1beta4.EmqxEnterprise{}
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), ee)
				if ee.GetStatus().GetConditions()[0].Type == appsv1beta4.ConditionBlueGreenUpdating {
					return ee.GetStatus().GetConditions()[0].Status
				}
				return corev1.ConditionUnknown
			}, timeout, interval).Should(Equal(corev1.ConditionTrue))

			Eventually(func() []corev1.Pod {
				podList := &corev1.PodList{}
				_ = k8sClient.List(
					context.TODO(),
					podList,
					client.InNamespace(sts.GetNamespace()),
					client.MatchingLabels(map[string]string{
						"controller-revision-hash": sts.Status.CurrentRevision,
					}),
				)
				return podList.Items
			}, timeout, interval).Should(HaveLen(0))

			By("check currentStatefulSetVersion in CR status")
			Eventually(func() string {
				ee := &appsv1beta4.EmqxEnterprise{}
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), ee)
				return ee.Status.CurrentStatefulSetVersion
			}, timeout, interval).Should(Equal(newSts.Status.CurrentRevision))

			By("check emqx nodes in CR status")
			Eventually(func() string {
				ee := &appsv1beta4.EmqxEnterprise{}
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), ee)
				if len(ee.GetStatus().GetEmqxNodes()) > 0 {
					return ee.GetStatus().GetEmqxNodes()[0].Node
				}
				return ""
			}, timeout, interval).Should(Equal(fmt.Sprintf("emqx-ee@%s-0.emqx-ee-headless.%s.svc.cluster.local", newSts.Name, emqx.GetNamespace())))

			By("check running condition in CR status")
			Eventually(func() corev1.ConditionStatus {
				ee := &appsv1beta4.EmqxEnterprise{}
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), ee)
				if ee.GetStatus().GetConditions()[0].Type == appsv1beta4.ConditionRunning {
					return ee.GetStatus().GetConditions()[0].Status
				}
				return corev1.ConditionUnknown
			}, timeout, interval).Should(Equal(corev1.ConditionTrue))
		})
	})
})

func createEmqx(emqx appsv1beta4.Emqx) {
	emqx.SetNamespace(emqx.GetNamespace() + "-" + rand.String(5))
	emqx.Default()
	Expect(emqx.ValidateCreate()).Should(Succeed())
	Expect(k8sClient.Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: emqx.GetNamespace(),
			Labels: map[string]string{
				"test": "e2e",
			},
		},
	})).Should(Succeed())
	Expect(k8sClient.Create(context.TODO(), emqx)).Should(Succeed())
	Eventually(func() bool {
		_ = k8sClient.Get(
			context.TODO(),
			types.NamespacedName{
				Name:      emqx.GetName(),
				Namespace: emqx.GetNamespace(),
			},
			emqx,
		)
		return len(emqx.GetStatus().GetEmqxNodes()) > 0
	}, timeout, interval).Should(BeTrue())
}

func deleteEmqx(emqx appsv1beta4.Emqx) {
	finalizer := "apps.emqx.io/finalizer"
	plugins := &appsv1beta4.EmqxPluginList{}
	_ = k8sClient.List(
		context.Background(),
		plugins,
		client.InNamespace("default"),
	)
	for _, plugin := range plugins.Items {
		controllerutil.RemoveFinalizer(&plugin, finalizer)
		Expect(k8sClient.Update(context.Background(), &plugin)).Should(Succeed())
		Expect(k8sClient.Delete(context.Background(), &plugin)).Should(Succeed())
	}
	Expect(k8sClient.Delete(context.TODO(), emqx)).Should(Succeed())
	Expect(k8sClient.Delete(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: emqx.GetNamespace(),
		},
	})).Should(Succeed())

	Eventually(func() bool {
		err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: emqx.GetNamespace()}, &corev1.Namespace{})
		return k8sErrors.IsNotFound(err)
	}, timeout, interval).Should(BeTrue())
}

var _ = Describe("Emqx Rebalance Test", Label("rebalance"), func() {
	Describe("Just for enterprise", func() {
		emqx := emqxEnterprise.DeepCopy()
		emqx.Spec.Template.Spec.EmqxContainer.Image.Version = "4.4.16"
		BeforeEach(func() {
			createEmqx(emqx)
		})

		AfterEach(func() {
			deleteEmqx(emqx)
		})

		By("check emqx running condition in CR status")
		Eventually(func() corev1.ConditionStatus {
			ee := &appsv1beta4.EmqxEnterprise{}
			_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), ee)
			if ee.GetStatus().GetConditions()[0].Type == appsv1beta4.ConditionRunning {
				return ee.GetStatus().GetConditions()[0].Status
			}
			return corev1.ConditionUnknown
		}, timeout, interval).Should(Equal(corev1.ConditionTrue))

		By("create emqx rebalance CR ")
		createRebalance(emqx)

		By("check emqx rebalance condition in CR status")
		Eventually(func() v1beta4.Condition {
			emqxRebalance := &appsv1beta4.EmqxRebalance{}
			_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqxRebalance), emqxRebalance)
			if emqxRebalance.Status.Conditions[0].Type == appsv1beta4.ConditionComplete {
				return emqxRebalance.Status.Conditions[0]
			}
			return v1beta4.Condition{}
		}, timeout, interval).Should(Equal(v1beta4.Condition{
			Type:    appsv1beta4.ConditionComplete,
			Status:  v1.ConditionFalse,
			Reason:  "Complete",
			Message: "[\"nothing_to_balance\"]",
		}))

		By("check emqx rebalance condition in CR status")
		Eventually(func() v1beta4.Condition {
			emqxRebalance := &appsv1beta4.EmqxRebalance{}
			_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqxRebalance), emqxRebalance)
			if emqxRebalance.Status.Conditions[0].Type == appsv1beta4.ConditionComplete {
				return emqxRebalance.Status.Conditions[0]
			}
			return v1beta4.Condition{}
		}, timeout, interval).Should(Equal(v1beta4.Condition{
			Type:    appsv1beta4.ConditionComplete,
			Status:  v1.ConditionFalse,
			Reason:  "Complete",
			Message: "emqx rebalance has already completed",
		}))

	})
})

func createRebalance(emqx appsv1beta4.Emqx) {
	emqxRebalance := appsv1beta4.EmqxRebalance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.String(5),
			Namespace: emqx.GetNamespace(),
		},
		Spec: appsv1beta4.EmqxRebalanceSpec{
			EmqxInstance: emqx.GetName(),
			RebalanceStrategy: &appsv1beta4.RebalanceStrategy{
				WaitTakeover:     pointer.Int32(10),
				ConnEvictRate:    pointer.Int32(10),
				SessEvictRate:    pointer.Int32(10),
				WaitHealthCheck:  pointer.Int32(10),
				AbsSessThreshold: pointer.Int32(100),
				RelConnThreshold: pointer.String("1.2"),
				AbsConnThreshold: pointer.Int32(100),
				RelSessThreshold: pointer.String("1.2"),
			},
		},
	}
	Expect(emqxRebalance.ValidateCreate()).Should(Succeed())
	Expect(k8sClient.Create(context.TODO(), &emqxRebalance)).Should(Succeed())
}
