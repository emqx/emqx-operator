/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller_suite_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/emqx/emqx-operator/apis/apps/v1beta2"
	controllers "github.com/emqx/emqx-operator/controllers/apps"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	brokerName      = "emqx"
	brokerNameSpace = "broker"

	enterpriseName      = "emqx-ee"
	enterpriseNameSpace = "enterprise"
)

var timeout, interval time.Duration
var k8sClient client.Client
var testEnv *envtest.Environment

var emqxList = func() []v1beta2.Emqx {
	return []v1beta2.Emqx{
		generateEmqxBroker(brokerName, brokerNameSpace),
		generateEmqxEnterprise(enterpriseName, enterpriseNameSpace),
	}
}

func TestSuites(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	interval = time.Millisecond * 250
	timeout = time.Minute * 1
	if os.Getenv("CI") == "true" {
		timeout = time.Minute * 5
	}

	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	Expect(os.Setenv("USE_EXISTING_CLUSTER", "true")).To(Succeed())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
		// WebhookInstallOptions: envtest.WebhookInstallOptions{
		// 	Paths: []string{filepath.Join("..", "config", "webhook")},
		// },
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = v1beta2.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	// // start webhook server using Manager
	// webhookInstallOptions := &testEnv.WebhookInstallOptions
	// k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
	// 	Scheme:             scheme.Scheme,
	// 	Host:               webhookInstallOptions.LocalServingHost,
	// 	Port:               webhookInstallOptions.LocalServingPort,
	// 	CertDir:            webhookInstallOptions.LocalServingCertDir,
	// 	LeaderElection:     false,
	// 	MetricsBindAddress: "0",
	// })
	// Expect(err).NotTo(HaveOccurred())

	// err = (&v1beta2.EmqxBroker{}).SetupWebhookWithManager(k8sManager)
	// Expect(err).NotTo(HaveOccurred())
	// err = (&v1beta2.EmqxEnterprise{}).SetupWebhookWithManager(k8sManager)
	// Expect(err).NotTo(HaveOccurred())

	newEmqxBroker := controllers.NewEmqxBrokerReconciler(k8sManager)
	err = newEmqxBroker.SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	newEmqxEnterprise := controllers.NewEmqxEnterpriseReconciler(k8sManager)
	err = newEmqxEnterprise.SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = k8sManager.AddHealthzCheck("healthz", healthz.Ping)
	Expect(err).ToNot(HaveOccurred())

	err = k8sManager.AddReadyzCheck("readyz", healthz.Ping)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

	// // wait for the webhook server to get ready
	// dialer := &net.Dialer{Timeout: time.Second}
	// addrPort := fmt.Sprintf("%s:%d", webhookInstallOptions.LocalServingHost, webhookInstallOptions.LocalServingPort)
	// Eventually(func() error {
	// 	conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true})
	// 	if err != nil {
	// 		return err
	// 	}
	// 	conn.Close()
	// 	return nil
	// }).Should(Succeed())

	for _, emqx := range emqxList() {
		namespace := generateEmqxNamespace(emqx.GetNamespace())
		Expect(k8sClient.Create(context.Background(), namespace)).Should(Succeed())

		Expect(k8sClient.Create(context.Background(), emqx)).Should(Succeed())
	}
}, 60)

var _ = AfterSuite(func() {
	Expect(cleanAll()).Should(Succeed())

	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func cleanAll() error {
	broker := &v1beta2.EmqxBroker{}
	if err := k8sClient.Get(
		context.Background(),
		types.NamespacedName{
			Name:      brokerName,
			Namespace: brokerNameSpace,
		},
		broker,
	); !errors.IsNotFound(err) {
		if err := k8sClient.Delete(
			context.Background(),
			&v1beta2.EmqxBroker{
				ObjectMeta: metav1.ObjectMeta{
					Name:      broker.GetName(),
					Namespace: broker.GetNamespace(),
				},
			},
		); err != nil {
			return err
		}
		// If PVC is set, then it should be retained
		if !reflect.ValueOf(broker.GetStorage()).IsZero() {
			if err := k8sClient.List(
				context.Background(),
				&corev1.PersistentVolumeClaimList{},
				&client.ListOptions{
					Namespace:     broker.GetNamespace(),
					LabelSelector: labels.SelectorFromSet(broker.GetLabels()),
				},
			); err != nil {
				return err
			}
		}
		if err := k8sClient.Delete(
			context.Background(),
			generateEmqxNamespace(brokerNameSpace),
		); err != nil {
			return err
		}
	}

	enterprise := &v1beta2.EmqxEnterprise{}
	if err := k8sClient.Get(
		context.Background(),
		types.NamespacedName{
			Name:      enterpriseName,
			Namespace: enterpriseNameSpace,
		},
		enterprise,
	); !errors.IsNotFound(err) {
		if err := k8sClient.Delete(
			context.Background(),
			&v1beta2.EmqxEnterprise{
				ObjectMeta: metav1.ObjectMeta{
					Name:      enterprise.GetName(),
					Namespace: enterprise.GetNamespace(),
				},
			},
		); err != nil {
			return err
		}
		// If PVC is set, then it should be retained
		if !reflect.ValueOf(enterprise.GetStorage()).IsZero() {
			if err := k8sClient.List(
				context.Background(),
				&corev1.PersistentVolumeClaimList{},
				&client.ListOptions{
					Namespace:     enterprise.GetNamespace(),
					LabelSelector: labels.SelectorFromSet(enterprise.GetLabels()),
				},
			); err != nil {
				return err
			}
		}
		if err := k8sClient.Delete(
			context.Background(),
			generateEmqxNamespace(enterpriseNameSpace),
		); err != nil {
			return err
		}
	}

	return nil
}

func generateEmqxNamespace(namespace string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
}

// Full
func generateEmqxBroker(name, namespace string) *v1beta2.EmqxBroker {
	defaultReplicas := int32(3)
	storageClassName := "standard"
	telegrafConf := `
[global_tags]
  instanceID = "test"

[[inputs.http]]
  urls = ["http://127.0.0.1:8081/api/v4/emqx_prometheus"]
  method = "GET"
  timeout = "5s"
  username = "admin"
  password = "public"
  data_format = "json"
[[inputs.tail]]
  files = ["/opt/emqx/log/emqx.log.[1-5]"]
  from_beginning = false
  max_undelivered_lines = 64
  character_encoding = "utf-8"
  data_format = "grok"
  grok_patterns = ['^%{TIMESTAMP_ISO8601:timestamp:ts-"2006-01-02T15:04:05.999999999-07:00"} \[%{LOGLEVEL:level}\] (?m)%{GREEDYDATA:messages}$']

[[outputs.discard]]
`
	emqx := &v1beta2.EmqxBroker{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps.emqx.io/v1beta2",
			Kind:       "EmqxBroker",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1beta2.EmqxBrokerSpec{
			Replicas: &defaultReplicas,
			Image:    "emqx/emqx:4.3.10",
			Labels: map[string]string{
				"cluster": "emqx",
			},
			Storage: corev1.PersistentVolumeClaimSpec{
				StorageClassName: &storageClassName,
				AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("20Mi"),
					},
				},
			},
			EmqxTemplate: v1beta2.EmqxBrokerTemplate{
				Listener: v1beta2.Listener{
					Type: "ClusterIP",
					Ports: v1beta2.Ports{
						MQTT:      1883,
						MQTTS:     8883,
						WS:        8083,
						WSS:       8084,
						Dashboard: 18083,
						API:       8081,
					},
					Certificate: v1beta2.Certificate{
						MQTTS: v1beta2.CertificateConf{
							Data: v1beta2.CertificateData{
								CaCert:  []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURVVENDQWptZ0F3SUJBZ0lKQVBQWUNqVG14ZHQvTUEwR0NTcUdTSWIzRFFFQkN3VUFNRDh4Q3pBSkJnTlYKQkFZVEFrTk9NUkV3RHdZRFZRUUlEQWhvWVc1bmVtaHZkVEVNTUFvR0ExVUVDZ3dEUlUxUk1ROHdEUVlEVlFRRApEQVpTYjI5MFEwRXdIaGNOTWpBd05UQTRNRGd3TmpVeVdoY05NekF3TlRBMk1EZ3dOalV5V2pBL01Rc3dDUVlEClZRUUdFd0pEVGpFUk1BOEdBMVVFQ0F3SWFHRnVaM3BvYjNVeEREQUtCZ05WQkFvTUEwVk5VVEVQTUEwR0ExVUUKQXd3R1VtOXZkRU5CTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUF6Y2dWTGV4MQpFWjlPTjY0RVg4dit3Y1Nqek9acGlFT3NBT3VTWE9FTjN3YjhGS1V4Q2RzR3JzSllCN2E1Vk0vSm90MjVNb2QyCmp1UzNPQk1nNnI4NWsyVFdqZHhVb1VzK0hpVUIvcFAvQVJhYVc2Vm50cEFFb2twaWovcHJ6V01QZ0puQkYzVXIKTWp0YkxheUg5aEdtcFFySTVjMnZtSFEycmVSWm5TRmJZKzJiOFNYWiszbFpaZ3o5K0JhUVlXZFFXZmFVV0VIWgp1RGFOaVZpVk8wT1Q4RFJqQ3VpRHAzeVlEajNpTFdiVEEvZ0RMNlRmNVh1SHVFd2NPUVVyZCtoMGh5SXBoTzhECnRzcnNIWjE0ajRBV1lMazFDUEE2cHExSElVdkVsMnJBTngybFZVTnYrbnQ2NEsvTXIzUm5WUWQ5czhiSytUWFEKS0dIZDJMdi9QQUxZdXdJREFRQUJvMUF3VGpBZEJnTlZIUTRFRmdRVUdCbVcraUR6eGN0V0FXeG1oZ2RsRThQagpFYlF3SHdZRFZSMGpCQmd3Rm9BVUdCbVcraUR6eGN0V0FXeG1oZ2RsRThQakViUXdEQVlEVlIwVEJBVXdBd0VCCi96QU5CZ2txaGtpRzl3MEJBUXNGQUFPQ0FRRUFHYmhSVWpwSXJlZDRjRkFGSjdiYllEOWhLdS95eldQV2tNUmEKRXJsQ0tIbXVZc1lrKzVkMTZKUWhKYUZ5Nk1HWGZMZ28zS1YyaXRsMGQrT1dOSDBVOVVMWGNnbFR4eTYrbmpvNQpDRnFkVUJQd04xanhoem85eXRlRE1LRjQrQUhJeGJ2Q0FKYTE3cWN3VUtSNU1LTnZ2MDlDNnB2UURKTHppZDd5CkUyZGtnU3VnZ2lrM29hMDQyN0t2Y3RGZjh1aE9WOTRSdkVEeXF2VDUrcGdOWVoyWWZnYTlwRC9qanBvSEVVbG8KODhJR1U4L3dKQ3gzRHMyeWM4K29CZy95bnhHOGYvSG1DQzFFVDZFSEhvZTJqbG84RnBVL1NnR3RnaFMxWUwzMApJV3hOc1ByVVArWHNacEJKeS9tdk9oRTVRWG82WTM1ekRxcWo4dEk3QUdtQVd1MjJqZz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"),
								TLSCert: []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURFekNDQWZ1Z0F3SUJBZ0lCQWpBTkJna3Foa2lHOXcwQkFRc0ZBREEvTVFzd0NRWURWUVFHRXdKRFRqRVIKTUE4R0ExVUVDQXdJYUdGdVozcG9iM1V4RERBS0JnTlZCQW9NQTBWTlVURVBNQTBHQTFVRUF3d0dVbTl2ZEVOQgpNQjRYRFRJd01EVXdPREE0TURjd05Wb1hEVE13TURVd05qQTRNRGN3TlZvd1B6RUxNQWtHQTFVRUJoTUNRMDR4CkVUQVBCZ05WQkFnTUNHaGhibWQ2YUc5MU1Rd3dDZ1lEVlFRS0RBTkZUVkV4RHpBTkJnTlZCQU1NQmxObGNuWmwKY2pDQ0FTSXdEUVlKS29aSWh2Y05BUUVCQlFBRGdnRVBBRENDQVFvQ2dnRUJBTE5lV1QzcEUrUUZmaVJKekttbgpBTVVyV28zSzJqL1RtMytYbmw2V0x6NjcvMHJjWXJKYmJLdlMzdXlSUC9zdFh5WEVLdzlDZXB5UTFWaUJWRmtXCkFveThxUUVPV0ZEc1pjLzVVemhYVW5iNkxYcjNxVGtGRWpObWhqKzd1enYvbGJCeGxVRzFObFl6U2VPQjYvUlQKOHpIL2xoT2VLaExuV1lQWGRYS3NhMUZMNmlqNFg4RGVETzFrWTdmdkFHbUJuL1RIaDF1VHBEaXpNNFltZUkrNwo0ZG1heUE1eFh2QVJ0ZTVoNFZ1NVNJemU3aUMwNTdOK3Z5bVRvTWsySmdrK1paRnB5WHJucSt5bzZSYUQzQU5jCmxyYzRGYmVVUVo1YTVzNVN4Z3M5YTBZM1dNRys3YzVWblZYY2JqQlJ6L2FxMk50T25RUWppa0tLUUE4R0YwODAKQlFrQ0F3RUFBYU1hTUJnd0NRWURWUjBUQkFJd0FEQUxCZ05WSFE4RUJBTUNCZUF3RFFZSktvWklodmNOQVFFTApCUUFEZ2dFQkFKZWZuTVpwYVJESFFTTlVJRUwzaXdHWEU5YzZQbUlzUVZFMnVzdHIrQ2FrQnAzVFo0bDBlbkx0CmlHTWZFVkZqdTY5Y080b3lva1d2K2hsNWVDTWtIQmYxNEt2NTF2ajQ0OGpvd1luRjF6bXpuN1NFem01VXpsc2EKc3FqdEFwcm5MeW9mNjlXdExVMWo1cllXQnVGWDg2eU9Ud1JBRk5qbTlmdmhBY3JFT05Cc1F0cWlwQldrTVJPcAppVVlNa1JxYktjUU1kd3hvditsSEJZS3E5emJXUm9xTFJPQW41NFNScWdRazZjMTVKZEVmZ09PalNoYnNPa0lIClVocWN3UmtRaWM3bjF6d0hWR1ZEZ05JWlZnbUoySWRJV0JsUEVDN29MclJyQkQvWDFpRUVYdEthYjZwNW8yMm4KS0I1bU4raVFhRStPZTJjcEdLWkppSlJkTStJcUREUT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo="),
								TLSKey:  []byte("LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb3dJQkFBS0NBUUVBczE1WlBla1Q1QVYrSkVuTXFhY0F4U3RhamNyYVA5T2JmNWVlWHBZdlBydi9TdHhpCnNsdHNxOUxlN0pFLyt5MWZKY1FyRDBKNm5KRFZXSUZVV1JZQ2pMeXBBUTVZVU94bHovbFRPRmRTZHZvdGV2ZXAKT1FVU00yYUdQN3U3Ty8rVnNIR1ZRYlUyVmpOSjQ0SHI5RlB6TWYrV0U1NHFFdWRaZzlkMWNxeHJVVXZxS1BoZgp3TjRNN1dSanQrOEFhWUdmOU1lSFc1T2tPTE16aGlaNGo3dmgyWnJJRG5GZThCRzE3bUhoVzdsSWpON3VJTFRuCnMzNi9LWk9neVRZbUNUNWxrV25KZXVlcjdLanBGb1BjQTF5V3R6Z1Z0NVJCbmxybXpsTEdDejFyUmpkWXdiN3QKemxXZFZkeHVNRkhQOXFyWTIwNmRCQ09LUW9wQUR3WVhUelFGQ1FJREFRQUJBb0lCQVFDdXZDYnI3UGQzbHZJLwpuN1ZGUUcrN3BIUmUxVkt3QXhEa3gydDhjWW9zN3kvUVdjbThQdHdxdHc1OEh6UFpHV1lyZ0dNQ1JwenprUlNGClY5ZzN3UDFTNVNjdTVDNmRCdTVZSUdjMTU3dHFOR1hCK1NwZFpkZEpRNE5jNnlHSFhZRVJsbFQwNGZmQkdjM04KV0cvb1lTLzFjU3RlaVNJcnNEeS85MUZ2R1JDaTdGUHhIM3dJZ0hzc1kvdHc2OXMxQ2Z2YXE1bHIyTlRGenhJRwp4Q3ZwSktFZFNmVmZTOUk3TFlpeW1WanN0M0lPUi93NzYvWkZZOWNSYThadG1RU1dXc20wVFVwUkMxamRjYmttClpvSnB0WVdsUCtnU3d4L2ZwTVlmdHJrSkZHT0poSEpIUWh3eFQ1WC9hakFJU2Vxamp3a1dTRUpMd25IUWQxMUMKWnkyKzI5bEJBb0dCQU5sRUFJSzRWeENxeVBYTktmb09PaTVkUzY0TmZ2eUg0QTF2MitLYUhXYzdscWFxUE40OQplemZOMm4zWCtLV3g0Y3ZpREQ5MTRZYzJKUTF2VkpqU2FIY2k3eWl2b2NEbzJPZlpEbWpCcXphTXAveStyWDFSCi9mM01taVRxTWE0NjhyamF4STlSUlp1N3ZEZ3BUUit6YTErT0JDZ016anZBbmc4ZEp1Ti81Z2psQW9HQkFOTlkKdVlQS3RlYXJCbWtxZHJTVjdlVFVlNDlOaHIwWG90TGFWQkgzN1RDVzBYdjl3ak8yeG1ibTVHYS9EQ3RQSXNCYgp5UGVZd1g5RmpvYXN1YWRVRDdoUnZiRnU2ZEJhMEhHTG1rWFJKWlRjRDdNRVgyTGh1NEJ1QzcyeURMTEZkMHIrCkVwOVdQN0Y1aUp5YWdZcUladHorNHVmN2dCdlVEZG12WHozc0dyMVZBb0dBZFhURDZlZUtlaUk2UGxoS0J6dEYKek9iM0VRT08wU3NMdjNmbm9kdTdaYUhiVWdMYW9UTVB1QjE3cjJqZ3JZTTdGS1FDQnhUTmRmR1ptbWZEamxMQgoweFo1d0w4aWJVMzBaWEw4elRsV1BFbFNUOXN0bzRCK0ZZVlZGL3ZjRzlzV2VVVWIybmNQY0ovUG8zVUFrdERHCmpZUVRUeXVOR3RTSkhwYWQvWU9aY3RrQ2dZQnRXUmFDN2JxM29mMHJKR0ZPaGRRVDlTd0l0Ti9scmZqOGh5SEEKT2pwcVRWNE5mUG1oc0F0dTZqOTZPWmFlUWMrRkh2Z1h3dDA2Y0U2UnQ0Ukc0dU5QUmx1VEZnTzdYWUZEZml0UAp2Q3Bwbm9JdzZTNUJCdkh3UFArdUloVVgyYnNpL2RtOHZ1OHRiK2dTdm80UGt3dEZoRXI2STlIZ2xCS21jbW9nCnE2d2FFUUtCZ0h5ZWNGQmVNNkxzMTFDZDY0dmJvcndKUEF1eElXN0hCQUZqL0JTOTlvZUc0VGpCeDRTejJkRmQKcnpVaWJKdDRuZG5ISXZDTjhKUWtqTkcxNGk5aEpsbitIM21Sc3M4ZmJaOXZRZHFHKzJ2T1dBRFlTenpzTkk1NQpSRlk3SmpsdUtjVmtwL3pDRGVVeFRVM082c1MrdjYvM1ZFMTFDb2I2T1lReDNsTjV3clozCi0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg=="),
							},
						},
						WSS: v1beta2.CertificateConf{
							StringData: v1beta2.CertificateStringData{
								CaCert: `-----BEGIN CERTIFICATE-----
MIIDUTCCAjmgAwIBAgIJAPPYCjTmxdt/MA0GCSqGSIb3DQEBCwUAMD8xCzAJBgNV
BAYTAkNOMREwDwYDVQQIDAhoYW5nemhvdTEMMAoGA1UECgwDRU1RMQ8wDQYDVQQD
DAZSb290Q0EwHhcNMjAwNTA4MDgwNjUyWhcNMzAwNTA2MDgwNjUyWjA/MQswCQYD
VQQGEwJDTjERMA8GA1UECAwIaGFuZ3pob3UxDDAKBgNVBAoMA0VNUTEPMA0GA1UE
AwwGUm9vdENBMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAzcgVLex1
EZ9ON64EX8v+wcSjzOZpiEOsAOuSXOEN3wb8FKUxCdsGrsJYB7a5VM/Jot25Mod2
juS3OBMg6r85k2TWjdxUoUs+HiUB/pP/ARaaW6VntpAEokpij/przWMPgJnBF3Ur
MjtbLayH9hGmpQrI5c2vmHQ2reRZnSFbY+2b8SXZ+3lZZgz9+BaQYWdQWfaUWEHZ
uDaNiViVO0OT8DRjCuiDp3yYDj3iLWbTA/gDL6Tf5XuHuEwcOQUrd+h0hyIphO8D
tsrsHZ14j4AWYLk1CPA6pq1HIUvEl2rANx2lVUNv+nt64K/Mr3RnVQd9s8bK+TXQ
KGHd2Lv/PALYuwIDAQABo1AwTjAdBgNVHQ4EFgQUGBmW+iDzxctWAWxmhgdlE8Pj
EbQwHwYDVR0jBBgwFoAUGBmW+iDzxctWAWxmhgdlE8PjEbQwDAYDVR0TBAUwAwEB
/zANBgkqhkiG9w0BAQsFAAOCAQEAGbhRUjpIred4cFAFJ7bbYD9hKu/yzWPWkMRa
ErlCKHmuYsYk+5d16JQhJaFy6MGXfLgo3KV2itl0d+OWNH0U9ULXcglTxy6+njo5
CFqdUBPwN1jxhzo9yteDMKF4+AHIxbvCAJa17qcwUKR5MKNvv09C6pvQDJLzid7y
E2dkgSuggik3oa0427KvctFf8uhOV94RvEDyqvT5+pgNYZ2Yfga9pD/jjpoHEUlo
88IGU8/wJCx3Ds2yc8+oBg/ynxG8f/HmCC1ET6EHHoe2jlo8FpU/SgGtghS1YL30
IWxNsPrUP+XsZpBJy/mvOhE5QXo6Y35zDqqj8tI7AGmAWu22jg==
-----END CERTIFICATE-----
`,
								TLSCert: `-----BEGIN CERTIFICATE-----
MIIDEzCCAfugAwIBAgIBAjANBgkqhkiG9w0BAQsFADA/MQswCQYDVQQGEwJDTjER
MA8GA1UECAwIaGFuZ3pob3UxDDAKBgNVBAoMA0VNUTEPMA0GA1UEAwwGUm9vdENB
MB4XDTIwMDUwODA4MDcwNVoXDTMwMDUwNjA4MDcwNVowPzELMAkGA1UEBhMCQ04x
ETAPBgNVBAgMCGhhbmd6aG91MQwwCgYDVQQKDANFTVExDzANBgNVBAMMBlNlcnZl
cjCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALNeWT3pE+QFfiRJzKmn
AMUrWo3K2j/Tm3+Xnl6WLz67/0rcYrJbbKvS3uyRP/stXyXEKw9CepyQ1ViBVFkW
Aoy8qQEOWFDsZc/5UzhXUnb6LXr3qTkFEjNmhj+7uzv/lbBxlUG1NlYzSeOB6/RT
8zH/lhOeKhLnWYPXdXKsa1FL6ij4X8DeDO1kY7fvAGmBn/THh1uTpDizM4YmeI+7
4dmayA5xXvARte5h4Vu5SIze7iC057N+vymToMk2Jgk+ZZFpyXrnq+yo6RaD3ANc
lrc4FbeUQZ5a5s5Sxgs9a0Y3WMG+7c5VnVXcbjBRz/aq2NtOnQQjikKKQA8GF080
BQkCAwEAAaMaMBgwCQYDVR0TBAIwADALBgNVHQ8EBAMCBeAwDQYJKoZIhvcNAQEL
BQADggEBAJefnMZpaRDHQSNUIEL3iwGXE9c6PmIsQVE2ustr+CakBp3TZ4l0enLt
iGMfEVFju69cO4oyokWv+hl5eCMkHBf14Kv51vj448jowYnF1zmzn7SEzm5Uzlsa
sqjtAprnLyof69WtLU1j5rYWBuFX86yOTwRAFNjm9fvhAcrEONBsQtqipBWkMROp
iUYMkRqbKcQMdwxov+lHBYKq9zbWRoqLROAn54SRqgQk6c15JdEfgOOjShbsOkIH
UhqcwRkQic7n1zwHVGVDgNIZVgmJ2IdIWBlPEC7oLrRrBD/X1iEEXtKab6p5o22n
KB5mN+iQaE+Oe2cpGKZJiJRdM+IqDDQ=
-----END CERTIFICATE-----
`,
								TLSKey: `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAs15ZPekT5AV+JEnMqacAxStajcraP9Obf5eeXpYvPrv/Stxi
sltsq9Le7JE/+y1fJcQrD0J6nJDVWIFUWRYCjLypAQ5YUOxlz/lTOFdSdvotevep
OQUSM2aGP7u7O/+VsHGVQbU2VjNJ44Hr9FPzMf+WE54qEudZg9d1cqxrUUvqKPhf
wN4M7WRjt+8AaYGf9MeHW5OkOLMzhiZ4j7vh2ZrIDnFe8BG17mHhW7lIjN7uILTn
s36/KZOgyTYmCT5lkWnJeuer7KjpFoPcA1yWtzgVt5RBnlrmzlLGCz1rRjdYwb7t
zlWdVdxuMFHP9qrY206dBCOKQopADwYXTzQFCQIDAQABAoIBAQCuvCbr7Pd3lvI/
n7VFQG+7pHRe1VKwAxDkx2t8cYos7y/QWcm8Ptwqtw58HzPZGWYrgGMCRpzzkRSF
V9g3wP1S5Scu5C6dBu5YIGc157tqNGXB+SpdZddJQ4Nc6yGHXYERllT04ffBGc3N
WG/oYS/1cSteiSIrsDy/91FvGRCi7FPxH3wIgHssY/tw69s1Cfvaq5lr2NTFzxIG
xCvpJKEdSfVfS9I7LYiymVjst3IOR/w76/ZFY9cRa8ZtmQSWWsm0TUpRC1jdcbkm
ZoJptYWlP+gSwx/fpMYftrkJFGOJhHJHQhwxT5X/ajAISeqjjwkWSEJLwnHQd11C
Zy2+29lBAoGBANlEAIK4VxCqyPXNKfoOOi5dS64NfvyH4A1v2+KaHWc7lqaqPN49
ezfN2n3X+KWx4cviDD914Yc2JQ1vVJjSaHci7yivocDo2OfZDmjBqzaMp/y+rX1R
/f3MmiTqMa468rjaxI9RRZu7vDgpTR+za1+OBCgMzjvAng8dJuN/5gjlAoGBANNY
uYPKtearBmkqdrSV7eTUe49Nhr0XotLaVBH37TCW0Xv9wjO2xmbm5Ga/DCtPIsBb
yPeYwX9FjoasuadUD7hRvbFu6dBa0HGLmkXRJZTcD7MEX2Lhu4BuC72yDLLFd0r+
Ep9WP7F5iJyagYqIZtz+4uf7gBvUDdmvXz3sGr1VAoGAdXTD6eeKeiI6PlhKBztF
zOb3EQOO0SsLv3fnodu7ZaHbUgLaoTMPuB17r2jgrYM7FKQCBxTNdfGZmmfDjlLB
0xZ5wL8ibU30ZXL8zTlWPElST9sto4B+FYVVF/vcG9sWeUUb2ncPcJ/Po3UAktDG
jYQTTyuNGtSJHpad/YOZctkCgYBtWRaC7bq3of0rJGFOhdQT9SwItN/lrfj8hyHA
OjpqTV4NfPmhsAtu6j96OZaeQc+FHvgXwt06cE6Rt4RG4uNPRluTFgO7XYFDfitP
vCppnoIw6S5BBvHwPP+uIhUX2bsi/dm8vu8tb+gSvo4PkwtFhEr6I9HglBKmcmog
q6waEQKBgHyecFBeM6Ls11Cd64vborwJPAuxIW7HBAFj/BS99oeG4TjBx4Sz2dFd
rzUibJt4ndnHIvCN8JQkjNG14i9hJln+H3mRss8fbZ9vQdqG+2vOWADYSzzsNI55
RFY7JjluKcVkp/zCDeUxTU3O6sS+v6/3VE11Cob6OYQx3lN5wrZ3
-----END RSA PRIVATE KEY-----
`,
							},
						},
					},
				},
				ACL: []v1beta2.ACL{
					{
						Permission: "allow",
					},
				},
				Plugins: []v1beta2.Plugin{
					{
						Name:   "emqx_management",
						Enable: true,
					},
				},
				Modules: []v1beta2.EmqxBrokerModules{
					{
						Name:   "emqx_mod_acl_internal",
						Enable: true,
					},
				},
			},
			TelegrafTemplate: &v1beta2.TelegrafTemplate{
				Image: "telegraf:1.19.3",
				Conf:  &telegrafConf,
			},
		},
	}
	emqx.Default()
	return emqx
}

// Slim
func generateEmqxEnterprise(name, namespace string) *v1beta2.EmqxEnterprise {
	defaultReplicas := int32(3)
	telegrafConf := `
[global_tags]
  instanceID = "test"

[[inputs.http]]
  urls = ["http://127.0.0.1:8081/api/v4/emqx_prometheus"]
  method = "GET"
  timeout = "5s"
  username = "admin"
  password = "public"
  data_format = "json"
[[inputs.tail]]
  files = ["/opt/emqx/log/emqx.log.[1-5]"]
  from_beginning = false
  max_undelivered_lines = 64
  character_encoding = "utf-8"
  data_format = "grok"
  grok_patterns = ['^%{TIMESTAMP_ISO8601:timestamp:ts-"2006-01-02T15:04:05.999999999-07:00"} \[%{LOGLEVEL:level}\] (?m)%{GREEDYDATA:messages}$']

[[outputs.discard]]
`
	emqx := &v1beta2.EmqxEnterprise{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps.emqx.io/v1beta2",
			Kind:       "EmqxEnterprise",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"cluster": "emqx",
			},
		},
		Spec: v1beta2.EmqxEnterpriseSpec{
			Replicas: &defaultReplicas,
			Image:    "emqx/emqx-ee:4.4.0",
			EmqxTemplate: v1beta2.EmqxEnterpriseTemplate{
				License: `
-----BEGIN CERTIFICATE-----
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
-----END CERTIFICATE-----
`,
			},

			TelegrafTemplate: &v1beta2.TelegrafTemplate{
				Image: "telegraf:1.19.3",
				Conf:  &telegrafConf,
			},
		},
	}
	emqx.Default()
	return emqx
}

func updateEmqx(emqx v1beta2.Emqx) error {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(emqx.GetObjectKind().GroupVersionKind())

	err := k8sClient.Get(
		context.TODO(),
		types.NamespacedName{
			Name:      emqx.GetName(),
			Namespace: emqx.GetNamespace(),
		},
		u,
	)

	if err != nil {
		return err
	}

	emqx.SetResourceVersion(u.GetResourceVersion())
	emqx.SetCreationTimestamp(u.GetCreationTimestamp())
	emqx.SetManagedFields(u.GetManagedFields())

	return k8sClient.Update(context.Background(), emqx)
}
