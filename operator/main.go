/*
Copyright 2021 Red Hat.

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

	"github.com/stackrox/rox/image"
	centralv1Alpha1 "github.com/stackrox/rox/operator/api/central/v1alpha1"
	securedClusterv1Alpha1 "github.com/stackrox/rox/operator/api/securedcluster/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/securedcluster/values/translation"
	helmReconciler "github.com/stackrox/rox/pkg/operator/helm/reconciler"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	centralKind        = "Central"
	securedClusterKind = "SecuredCluster"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme
}

type translator struct{}

func (translator) Translate(u *unstructured.Unstructured) (chartutil.Values, error) {
	central := centralv1Alpha1.Central{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &central)
	if err != nil {
		return nil, err
	}

	// TODO(ROX-7088): replace this placeholder translation
	v := chartutil.Values{}

	return v, err
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var enablev1HelmOperator bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&enablev1HelmOperator, "helmv1", false, "Use the v1 helm-operator from operator-sdk")
	opts := zap.Options{
		Development: true,
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
		LeaderElectionID:       "bf7ea6a2.platform.stackrox.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	centralGVK := schema.GroupVersionKind{Group: centralv1Alpha1.GroupVersion.Group, Version: centralv1Alpha1.GroupVersion.Version, Kind: centralKind}
	if err := helmReconciler.SetupReconcilerWithManager(mgr, centralGVK, image.CentralServicesChartPrefix, translator{}); err != nil {
		setupLog.Error(err, "unable to setup central reconciler")
		os.Exit(1)
	}

	securedClusterGVK := schema.GroupVersionKind{Group: securedClusterv1Alpha1.GroupVersion.Group, Version: securedClusterv1Alpha1.GroupVersion.Version, Kind: securedClusterKind}
	if err := helmReconciler.SetupReconcilerWithManager(mgr, securedClusterGVK, image.SecuredClusterServicesChartPrefix, translation.Translator{Config: mgr.GetConfig()}); err != nil {
		setupLog.Error(err, "unable to setup secured cluster reconciler")
		os.Exit(1)
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
