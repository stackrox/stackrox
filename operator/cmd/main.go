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
	"context"
	"crypto/tls"
	"flag"
	"os"
	"strconv"
	"time"

	"github.com/go-logr/zapr"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	centralReconciler "github.com/stackrox/rox/operator/internal/central/reconciler"
	commonLabels "github.com/stackrox/rox/operator/internal/common/labels"
	securedClusterReconciler "github.com/stackrox/rox/operator/internal/securedcluster/reconciler"
	"github.com/stackrox/rox/operator/internal/utils"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/profiling"
	"github.com/stackrox/rox/pkg/version"
	rawZap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

const (
	envCentralLabelSelector            = "CENTRAL_LABEL_SELECTOR"
	envSecuredClusterLabelSelector     = "SECURED_CLUSTER_LABEL_SELECTOR"
	envCentralReconcilerEnabled        = "CENTRAL_RECONCILER_ENABLED"
	envSecuredClusterReconcilerEnabled = "SECURED_CLUSTER_RECONCILER_ENABLED"
)

var (
	setupLog                   = ctrl.Log.WithName("setup")
	scheme                     = runtime.NewScheme()
	enableProfiling            = env.RegisterBooleanSetting("ENABLE_PROFILING", false)
	profilingThresholdFraction = env.RegisterSetting("PROFILING_THRESHOLD_FRACTION", env.WithDefault("0.8"))
	memLimit                   = env.RegisterIntegerSetting("MEMORY_LIMIT_BYTES", 0)
	// Default place to put the heap dump is the /tmp directory because the container process has rights
	// to write to this directory without creating and mounting a PVC
	heapDumpDir = env.RegisterSetting("HEAP_DUMP_PARENT_DIR", env.WithDefault("/tmp"))
	// centralLabelSelector is a kubernetes label selector that is used to filter out Central instances
	// to be managed by this operator. If the selector is empty, all Central instances are managed.
	// see https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
	centralLabelSelector = env.RegisterSetting(envCentralLabelSelector, env.WithDefault(""))
	// securedClusterLabelSelector is a kubernetes label selector that is used to filter out Secured Cluster
	// instances to be managed by this operator. If the selector is empty, all instances are managed.
	// see https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
	securedClusterLabelSelector = env.RegisterSetting(envSecuredClusterLabelSelector, env.WithDefault(""))
	// centralReconcilerEnabled enables registering central reconciler if set to true otherwise skips it
	centralReconcilerEnabled = env.RegisterBooleanSetting(envCentralReconcilerEnabled, true)
	// securedClusterReconcilerEnabled enables registering secured cluster reconciler if set to true otherwise skips it
	securedClusterReconcilerEnabled = env.RegisterBooleanSetting(envSecuredClusterReconcilerEnabled, true)
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(platform.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	if err := run(); err != nil {
		setupLog.Error(err, "fatal error")
		os.Exit(1)
	}
}

func run() error {
	setupLog.Info("Starting RHACS Operator", "version", version.GetMainVersion())

	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var enableHTTP2 bool

	flag.StringVar(&metricsAddr, "metrics-bind-address", "0.0.0.0:8443", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&enableHTTP2, "enable-http2", enableHTTP2, "If HTTP/2 should be enabled for the metrics server.")

	opts := zap.Options{
		Development: !buildinfo.ReleaseBuild,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	zapLogger := zap.NewRaw(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(zapr.NewLogger(zapLogger))
	restore, err := rawZap.RedirectStdLogAt(zapLogger, zapcore.DebugLevel)
	if err != nil {
		return errors.Wrap(err, "unable to redirect std log")
	}
	defer restore()

	var tlsOpts []func(c *tls.Config)
	if !enableHTTP2 {
		// Mitigate CVE-2023-44487 by disabling HTTP2 and forcing HTTP/1.1 until
		// the Go standard library and golang.org/x/net are fully fixed.
		// Right now, it is possible for authenticated and unauthenticated users to
		// hold open HTTP2 connections and consume huge amounts of memory.
		// See:
		// * https://github.com/kubernetes/kubernetes/pull/121120
		// * https://github.com/kubernetes/kubernetes/issues/121197
		// * https://github.com/golang/go/issues/63417#issuecomment-1758858612
		tlsOpts = append(tlsOpts, func(c *tls.Config) {
			c.NextProtos = []string{"http/1.1"}
		})
	}

	cacheLabelSelector := labels.SelectorFromSet(commonLabels.DefaultLabels())
	mgr, err := ctrl.NewManager(utils.GetRHACSConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress:    metricsAddr,
			SecureServing:  true,
			FilterProvider: filters.WithAuthenticationAndAuthorization,
			TLSOpts:        tlsOpts,
		},
		Cache: cache.Options{
			// Limit caching of Secret and ConfigMaps to labeled
			// resources because those are usually the objects
			// with highest impact on memory consumption.
			ByObject: map[ctrlClient.Object]cache.ByObject{
				&coreV1.Secret{}: {
					Label: cacheLabelSelector,
				},
				&coreV1.ConfigMap{}: {
					Label: cacheLabelSelector,
				},
			},
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "bf7ea6a2.stackrox.io",
	})
	if err != nil {
		return errors.Wrap(err, "unable to create manager")
	}

	if enableProfiling.BooleanSetting() {
		thresholdS := profilingThresholdFraction.Setting()
		thresholdF, err := strconv.ParseFloat(thresholdS, 32)
		if err != nil {
			return errors.Wrapf(err, "unable to parse PROFILING_THRESHOLD set to '%s' as a float", thresholdS)
		}
		heapProfiler := profiling.NewHeapProfiler(thresholdF, uint64(memLimit.IntegerSetting()), heapDumpDir.Setting(), profiling.DefaultHeapProfilerBackoff)
		ctx, cancelProfiler := context.WithCancel(context.Background())
		go heapProfiler.DumpHeapOnThreshhold(ctx, time.Second)
		defer cancelProfiler()
	}

	centralLabelSelector := centralLabelSelector.Setting()
	if len(centralLabelSelector) > 0 {
		setupLog.Info("using Central label selector from environment variable "+envCentralLabelSelector, "selector", centralLabelSelector)
	}

	securedClusterLabelSelector := securedClusterLabelSelector.Setting()
	if len(securedClusterLabelSelector) > 0 {
		setupLog.Info("using Secured Cluster label selector from environment variable "+envSecuredClusterLabelSelector, "selector", securedClusterLabelSelector)
	}

	// The following comment marks the place where `operator-sdk` inserts new scaffolded code.
	//+kubebuilder:scaffold:builder

	if centralReconcilerEnabled.BooleanSetting() {
		if err = centralReconciler.RegisterNewReconciler(mgr, centralLabelSelector); err != nil {
			return errors.Wrap(err, "unable to set up Central reconciler")
		}
	} else {
		setupLog.Info("skip registering central reconciler because " + envCentralReconcilerEnabled + "==false")
	}

	if securedClusterReconcilerEnabled.BooleanSetting() {
		if err = securedClusterReconciler.RegisterNewReconciler(mgr, securedClusterLabelSelector); err != nil {
			return errors.Wrap(err, "unable to set up SecuredCluster reconciler")
		}
	} else {
		setupLog.Info("skip registering secured cluster reconciler because " + envSecuredClusterReconcilerEnabled + "==false")
	}

	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return errors.Wrap(err, "unable to set up health check")
	}
	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return errors.Wrap(err, "unable to set up readiness check")
	}

	setupLog.Info("starting manager")
	if err = mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return errors.Wrap(err, "problem running manager")
	}
	return nil
}
