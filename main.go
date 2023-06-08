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

package main

import (
	"flag"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	"go.uber.org/zap/zapcore"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/utils/pointer"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	appsv1beta3 "github.com/emqx/emqx-operator/apis/apps/v1beta3"
	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	appscontrollersv1beta4 "github.com/emqx/emqx-operator/controllers/apps/v1beta4"
	appscontrollersv2alpha2 "github.com/emqx/emqx-operator/controllers/apps/v2alpha2"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(appsv1beta3.AddToScheme(scheme))
	utilruntime.Must(appsv1beta4.AddToScheme(scheme))
	utilruntime.Must(appsv2alpha1.AddToScheme(scheme))
	utilruntime.Must(appsv2alpha2.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=persistentvolumes,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=pods/exec,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=pods/portforward,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=pods/status,verbs=patch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=apps,resources=replicasets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=endpoints,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "19fd6fcc.emqx.io",
		LeaseDuration:          pointer.Duration(time.Second * 30),
		RenewDeadline:          pointer.Duration(time.Second * 20),
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&appscontrollersv1beta4.EmqxBrokerReconciler{
		EmqxReconciler: appscontrollersv1beta4.NewEmqxReconciler(mgr),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EmqxBroker")
		os.Exit(1)
	}

	if err = (&appscontrollersv1beta4.EmqxEnterpriseReconciler{
		EmqxReconciler: appscontrollersv1beta4.NewEmqxReconciler(mgr),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EmqxEnterprise")
		os.Exit(1)
	}

	if err = appscontrollersv1beta4.NewEmqxPluginReconciler(mgr).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EmqxPlugin")
		os.Exit(1)
	}

	if err = appscontrollersv2alpha2.NewEMQXReconciler(mgr).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EMQX")
		os.Exit(1)
	}

	if err = appscontrollersv1beta4.NewRebalanceReconciler(mgr).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Rebalance")
	}

	//+kubebuilder:scaffold:builder

	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&appsv1beta4.EmqxPlugin{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "EmqxPlugin")
			os.Exit(1)
		}
		if err = (&appsv1beta4.Rebalance{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Rebalance")
			os.Exit(1)
		}
		if err = (&appsv1beta4.EmqxBroker{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "EmqxBroker")
			os.Exit(1)
		}
		if err = (&appsv1beta4.EmqxEnterprise{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "EmqxEnterprise")
			os.Exit(1)
		}

		if err = (&appsv2alpha2.EMQX{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "EMQX")
			os.Exit(1)
		}
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
