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
	"time"

	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	appscontrollersv1beta4 "github.com/emqx/emqx-operator/controllers/apps/v1beta4"
	"github.com/emqx/emqx-operator/internal/handler"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegaTypes "github.com/onsi/gomega/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
		Replicas: pointer.Int32Ptr(1),
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
		Replicas: pointer.Int32(1),
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

var ports = []corev1.ServicePort{
	{
		Name:       "http-dashboard-18083",
		Port:       18083,
		Protocol:   corev1.ProtocolTCP,
		TargetPort: intstr.FromInt(18083),
	},
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

var headlessPort = corev1.ServicePort{
	Name:       "http-management-8081",
	Port:       8081,
	Protocol:   corev1.ProtocolTCP,
	TargetPort: intstr.FromInt(8081),
}

var _ = Describe("Base E2E Test", func() {
	DescribeTable("",
		func(emqx appsv1beta4.Emqx, plugin *appsv1beta4.EmqxPlugin) {
			var pluginList []string
			var pluginPorts []corev1.ServicePort

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

			By("checking the EMQX Custom Resource's EndpointSlice", func() {
				checkPodAndEndpointsAndEndpointSlices(emqx, ports, pluginPorts, headlessPort, 1)
			})

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

			matchers := []gomegaTypes.GomegaMatcher{}
			for _, plugin := range pluginList {
				matchers = append(matchers, HaveKey(plugin+".conf"))
			}
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
			}, timeout, interval).Should(And(matchers...))

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
			}, timeout, interval).Should(ContainElements(headlessPort))

			By("check service ports", func() {
				checkService(emqx, ports, pluginPorts, headlessPort)
			})

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

			By("check service ports", func() {
				checkService(emqx, ports, pluginPorts, headlessPort)
			})

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

			By("checking the EMQX Custom Resource's EndpointSlice", func() {
				checkPodAndEndpointsAndEndpointSlices(emqx, ports, []corev1.ServicePort{}, headlessPort, 1)
			})

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
				return newSts.Status.CurrentRevision
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

			By("checking the EMQX Custom Resource's EndpointSlice when blue-green", func() {
				checkPodAndEndpointsAndEndpointSlices(emqx, ports, []corev1.ServicePort{}, headlessPort, 1)
			})

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

var rebalance = appsv1beta4.Rebalance{
	Spec: appsv1beta4.RebalanceSpec{
		RebalanceStrategy: appsv1beta4.RebalanceStrategy{
			WaitTakeover:     10,
			ConnEvictRate:    10,
			SessEvictRate:    10,
			WaitHealthCheck:  10,
			AbsSessThreshold: 100,
			RelConnThreshold: "1.2",
			AbsConnThreshold: 100,
			RelSessThreshold: "1.2",
		},
	},
}

var _ = Describe("Emqx Rebalance Test", Label("rebalance"), func() {
	Describe("Just for enterprise", func() {
		emqx := emqxEnterprise.DeepCopy()
		BeforeEach(func() {
			createEmqx(emqx)
		})

		AfterEach(func() {
			deleteEmqx(emqx)
		})

		It("emqx rebalance", func() {
			By("create a new rebalance job")
			time.Sleep(20 * time.Second)
			createRebalance(&rebalance, emqx)

			Eventually(func() []string {
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(&rebalance), &rebalance)
				return rebalance.Finalizers
			}, timeout, interval).Should(ConsistOf("apps.emqx.io/finalizer"))

			Eventually(func() string {
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(&rebalance), &rebalance)
				if len(rebalance.Status.Conditions) > 0 && rebalance.Status.Conditions[0].Type == appsv1beta4.RebalanceFailed {
					return rebalance.Status.Conditions[0].Message
				}
				return ""
			}, timeout, interval).Should(Equal("[\"nothing_to_balance\"]"))

			Eventually(func() error {
				return k8sClient.Delete(context.TODO(), &rebalance)
			}, timeout, interval).Should(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(&rebalance), &rebalance)
				return k8sErrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())

		})
	})
})

func createRebalance(rebalance *appsv1beta4.Rebalance, emqx appsv1beta4.Emqx) {
	rebalance.Name = "rebalance" + "-" + rand.String(5)
	rebalance.Namespace = emqx.GetNamespace()
	rebalance.Spec.InstanceName = emqx.GetName()
	Expect(rebalance.ValidateCreate()).Should(Succeed())
	Expect(k8sClient.Create(context.TODO(), rebalance)).Should(Succeed())
}

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

func checkService(emqx appsv1beta4.Emqx, ports, pluginPorts []corev1.ServicePort, headlessPort corev1.ServicePort) {
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
}

func checkPodAndEndpointsAndEndpointSlices(emqx appsv1beta4.Emqx, ports, pluginPorts []corev1.ServicePort, headlessPort corev1.ServicePort, count int) {
	podList := &corev1.PodList{}
	Eventually(func() []corev1.Pod {
		_ = k8sClient.List(context.TODO(), podList,
			client.InNamespace(emqx.GetNamespace()),
			client.MatchingLabels(emqx.GetSpec().GetTemplate().Labels),
		)
		return podList.Items
	}, timeout, interval).Should(
		And(
			HaveLen(count),
			HaveEach(
				HaveField("Status", And(
					HaveField("Phase", corev1.PodRunning),
					HaveField("Conditions", ContainElements(
						HaveField("Type", appsv1beta4.PodOnServing),
						HaveField("Type", corev1.PodReady),
					))),
				)),
		),
	)

	endPointsMatcher := []gomegaTypes.GomegaMatcher{}
	endpointSliceMatcher := []gomegaTypes.GomegaMatcher{}
	for _, p := range podList.Items {
		pod := p.DeepCopy()
		ep := And(
			HaveField("IP", pod.Status.PodIP),
			HaveField("NodeName", HaveValue(Equal(pod.Spec.NodeName))),
			HaveField("TargetRef", And(
				HaveField("Kind", "Pod"),
				HaveField("UID", pod.GetUID()),
				HaveField("Name", pod.GetName()),
				HaveField("Namespace", pod.GetNamespace()),
			)),
		)
		endPointsMatcher = append(endPointsMatcher, ep)

		eps := And(
			HaveField("Addresses", ConsistOf([]string{pod.Status.PodIP})),
			HaveField("NodeName", HaveValue(Equal(pod.Spec.NodeName))),
			HaveField("Conditions", And(
				HaveField("Ready", HaveValue(BeTrue())),
				HaveField("Serving", BeNil()),
				HaveField("Terminating", BeNil()),
			)),
			HaveField("TargetRef", And(
				HaveField("Kind", "Pod"),
				HaveField("UID", pod.GetUID()),
				HaveField("Name", pod.GetName()),
				HaveField("Namespace", pod.GetNamespace()),
			)),
		)
		endpointSliceMatcher = append(endpointSliceMatcher, eps)
	}

	servicePorts := append(pluginPorts, append(ports, headlessPort)...)
	endpointsPorts := []corev1.EndpointPort{}
	endpointSlicePorts := []discoveryv1.EndpointPort{}
	for _, port := range servicePorts {
		endpointsPorts = append(endpointsPorts, corev1.EndpointPort{
			Name:     port.Name,
			Port:     port.Port,
			Protocol: port.Protocol,
		})
		endpointSlicePorts = append(endpointSlicePorts, discoveryv1.EndpointPort{
			Name:     pointer.String(port.Name),
			Port:     pointer.Int32(port.Port),
			Protocol: &[]corev1.Protocol{port.Protocol}[0],
		})
	}

	Eventually(func() *corev1.Endpoints {
		ep := &corev1.Endpoints{}
		_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: emqx.GetSpec().GetServiceTemplate().Name, Namespace: emqx.GetSpec().GetServiceTemplate().Namespace}, ep)
		return ep
	}, timeout, interval).Should(HaveField("Subsets",
		And(
			HaveLen(1),
			ContainElement(
				HaveField("Addresses", ConsistOf(endPointsMatcher)),
			),
			ContainElement(
				HaveField("Ports", ConsistOf(endpointsPorts)),
			),
		),
	))

	Eventually(func() []discoveryv1.EndpointSlice {
		list := &discoveryv1.EndpointSliceList{}
		_ = k8sClient.List(
			context.TODO(), list,
			client.InNamespace(emqx.GetNamespace()),
			client.MatchingLabels(
				map[string]string{
					"kubernetes.io/service-name": emqx.GetSpec().GetServiceTemplate().Name,
				},
			),
		)
		return list.Items
	}, timeout, interval).Should(
		And(
			HaveLen(1),
			ContainElement(
				HaveField("Endpoints", ConsistOf(endpointSliceMatcher)),
			),
			ContainElement(
				HaveField("Ports", ConsistOf(endpointSlicePorts)),
			),
		),
	)
}
