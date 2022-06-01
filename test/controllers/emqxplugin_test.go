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

package controller_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	//+kubebuilder:scaffold:imports
)

var _ = Describe("", func() {
	Context("Check plugin", func() {
		It("Check default plugin", func() {
			for _, emqx := range emqxList() {
				Eventually(func() map[string]string {
					cm := &corev1.ConfigMap{}
					_ = k8sClient.Get(
						context.Background(),
						types.NamespacedName{
							Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "loaded-plugins"),
							Namespace: emqx.GetNamespace(),
						}, cm,
					)
					return cm.Data
				}, timeout, interval).Should(Equal(
					map[string]string{
						"loaded_plugins": "{emqx_management, true}.\n{emqx_dashboard, true}.\n{emqx_retainer, true}.\n{emqx_rule_engine, true}.\n",
					},
				))

				for _, pluginName := range []string{"emqx_management", "emqx_dashboard", "emqx_rule_engine", "emqx_retainer"} {
					Eventually(func() error {
						plugin := &v1beta3.EmqxPlugin{}
						err := k8sClient.Get(
							context.Background(),
							types.NamespacedName{
								Name:      strings.Replace(strings.Replace(pluginName, "emqx", emqx.GetName(), 1), "_", "-", -1),
								Namespace: emqx.GetNamespace(),
							}, plugin,
						)
						return err
					}, timeout, interval).Should(Succeed())

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
			}
		})

		It("Check lwm2m plugin", func() {
			for _, emqx := range emqxList() {
				// Create plugin
				lwm2m := &v1beta3.EmqxPlugin{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "apps.emqx.io/v1beta3",
						Kind:       "EmqxPlugin",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "lwm2m"),
						Namespace: emqx.GetNamespace(),
					},
					Spec: v1beta3.EmqxPluginSpec{
						PluginName: "emqx_lwm2m",
						Selector:   emqx.GetLabels(),
						Config: map[string]string{
							"lwm2m.lifetime_min":      "1s",
							"lwm2m.lifetime_max":      "86400s",
							"lwm2m.mountpoint":        "lwm2m/%e/",
							"lwm2m.topics.command":    "dn/#",
							"lwm2m.topics.response":   "up/resp",
							"lwm2m.topics.notify":     "up/notify",
							"lwm2m.topics.register":   "up/resp",
							"lwm2m.topics.update":     "up/resp",
							"lwm2m.xml_dir":           " etc/lwm2m_xml",
							"lwm2m.bind.udp.1":        "0.0.0.0:5683",
							"lwm2m.opts.buffer":       "1024KB",
							"lwm2m.opts.recbuf":       "1024KB",
							"lwm2m.opts.sndbuf":       "1024KB",
							"lwm2m.opts.read_packets": "20",
						},
					},
				}
				Expect(k8sClient.Create(context.Background(), lwm2m)).Should(Succeed())

				Eventually(func() error {
					plugin := &v1beta3.EmqxPlugin{}
					err := k8sClient.Get(
						context.Background(),
						types.NamespacedName{
							Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "lwm2m"),
							Namespace: emqx.GetNamespace(),
						}, plugin,
					)
					return err
				}, timeout, interval).Should(Succeed())

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
				}, timeout, interval).Should(ContainSubstring("{emqx_lwm2m, true}"))

				pluginConfig := &corev1.ConfigMap{}
				Eventually(func() map[string]string {
					_ = k8sClient.Get(
						context.Background(),
						types.NamespacedName{
							Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "plugins-config"),
							Namespace: emqx.GetNamespace(),
						}, pluginConfig,
					)
					return pluginConfig.Data
				}, timeout, interval).Should(HaveKey("emqx_lwm2m.conf"))

				// Delete plugin
				Expect(k8sClient.Delete(
					context.Background(),
					&v1beta3.EmqxPlugin{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "lwm2m"),
							Namespace: emqx.GetNamespace(),
						},
					},
				)).Should(Succeed())

				Eventually(func() map[string]string {
					cm := &corev1.ConfigMap{}
					_ = k8sClient.Get(
						context.Background(),
						types.NamespacedName{
							Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "loaded-plugins"),
							Namespace: emqx.GetNamespace(),
						}, cm,
					)
					return cm.Data
				}, timeout, interval).Should(Equal(
					map[string]string{
						"loaded_plugins": "{emqx_management, true}.\n{emqx_dashboard, true}.\n{emqx_retainer, true}.\n{emqx_rule_engine, true}.\n",
					},
				))

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
				}, timeout, interval).ShouldNot(HaveKey("emqx_lwm2m.conf"))
			}
		})
	})
})
