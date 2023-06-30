package cmd

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	centralReconciler "github.com/stackrox/rox/operator/pkg/central/reconciler"
	securedClusterReconciler "github.com/stackrox/rox/operator/pkg/securedcluster/reconciler"
	"github.com/stackrox/rox/operator/pkg/utils"
	"github.com/stackrox/rox/pkg/branding"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/profiling"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	// metricsBindAddr is the address the metric endpoint binds to.
	metricsBindAddr string
	// healthProbeBindAddrress is the address the probe endpoint binds to.
	healthProbeBindAddrress string
	// readinessProbeEndpointName is the endpoint name for the readiness probe
	readinessProbeEndpointName string
	// livenessProbeEndpointName string
	livenessProbeEndpointName string
	// enableLeaderElection is whether to enable leader election for controller manager.
	enableLeaderElection bool
	// leaderElectLeaseDuration is the duration that non-leader candidates will wait to force acquire leadership
	leaderElectLeaseDuration time.Duration
	// leaderElectRenewDeadline is the duration that the acting controlplane will retry refreshing leadership before giving up
	leaderElectRenewDeadline time.Duration
	// leaderElectRetryPeriod is the duration the LeaderElector clients should wait between tries of actions
	leaderElectRetryPeriod time.Duration
	// leaderElectID is the name of the configmap that is used for holding the leader lock.
	leaderElectID string
	// leaderElectNamespace determines the namespace in which the leader election resource will be created.
	leaderElectNamespace string
	// syncPeriod determines the minimum frequency at which watched resources are reconciled. A lower period will correct entropy more quickly, but reduce responsiveness to change if there are many watched resources.
	syncPeriod time.Duration
	// webhookPort is the port the webhook endpoint binds to.
	webhookPort int
	// webhookHost is the host that the webhook binds to
	webhookHost string
	// enableWebhooks is whether to enable webhooks.
	enableWebhooks bool
	// enableProfiling is whether to enable profiling.
	enableProfiling bool
	// profilingThresholdFraction is the profiling threshold fraction.
	profilingThresholdFraction float64
	// memLimit is the memory limit in bytes.
	memLimit uint64
	// heapDumpDir is the heap dump parent directory.
	heapDumpDir string
	// gracefulShutdownTimeout is the time the manager will wait for a graceful shutdown
	gracefulShutdownTimeout time.Duration
	// dryRunClient
	dryRunClient bool
	// centralLabelSelector is a kubernetes label selector that is used to filter out Central instances
	// to be managed by this operator. If the selector is empty, all Central instances are managed.
	// see https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
	centralLabelSelector string
	// enableCentralReconciler is whether to enable the central reconciler.
	enableCentralReconciler bool
	// enableSecuredClusterReconciler is whether to enable the secured cluster reconciler.
	enableSecuredClusterReconciler bool
	// zopOptions is the zap options.
	zapOptions = &zap.Options{Development: !buildinfo.ReleaseBuild}

	setupLog = ctrl.Log.WithName("setup")

	// Default place where controller-runtime looks for TLS artifacts.
	// see https://github.com/kubernetes-sigs/controller-runtime/blob/v0.8.3/pkg/webhook/server.go#L96-L104
	defaultCertDir  = filepath.Join(os.TempDir(), "k8s-webhook-server", "serving-certs")
	defaultTLSPaths = []string{filepath.Join(defaultCertDir, "tls.crt"), filepath.Join(defaultCertDir, "tls.key")}
)

var startCmd = cobra.Command{
	Use:   "start",
	Short: "Starts the " + branding.GetProductName() + " operator",
	RunE: func(cmd *cobra.Command, args []string) error {
		setupLog.Info("Starting " + fullOperatorVersion())

		if !enableCentralReconciler && !enableSecuredClusterReconciler {
			setupLog.Info("no reconcilers enabled, exiting")
			os.Exit(0)
		}

		ctrl.SetLogger(zap.New(zap.UseFlagOptions(zapOptions)))

		scheme := runtime.NewScheme()
		utilruntime.Must(clientgoscheme.AddToScheme(scheme))
		utilruntime.Must(platform.AddToScheme(scheme))

		mgr, err := ctrl.NewManager(utils.GetRHACSConfigOrDie(), ctrl.Options{
			DryRunClient:            dryRunClient,
			GracefulShutdownTimeout: &gracefulShutdownTimeout,
			HealthProbeBindAddress:  healthProbeBindAddrress,
			Host:                    webhookHost,
			LeaderElection:          enableLeaderElection,
			LeaderElectionID:        leaderElectID,
			LeaderElectionNamespace: leaderElectNamespace,
			LeaseDuration:           &leaderElectLeaseDuration,
			LivenessEndpointName:    livenessProbeEndpointName,
			MetricsBindAddress:      metricsBindAddr,
			Port:                    webhookPort,
			ReadinessEndpointName:   readinessProbeEndpointName,
			RenewDeadline:           &leaderElectRenewDeadline,
			RetryPeriod:             &leaderElectRetryPeriod,
			Scheme:                  scheme,
			SyncPeriod:              &syncPeriod,
		})
		if err != nil {
			return errors.Wrap(err, "unable to create controller manager")
		}

		if !enableWebhooks {
			setupLog.Info("webhooks are disabled")
		} else {
			maybeUseLegacyTLSFileLocation(mgr)
			if err = (&platform.Central{}).SetupWebhookWithManager(mgr); err != nil {
				return errors.Wrap(err, "unable to create Central webhook")
			}
		}

		if enableProfiling {
			heapProfiler := profiling.NewHeapProfiler(profilingThresholdFraction, memLimit, heapDumpDir, profiling.DefaultHeapProfilerBackoff)
			ctx, cancelProfiler := context.WithCancel(context.Background())
			go heapProfiler.DumpHeapOnThreshhold(ctx, time.Second)
			defer cancelProfiler()
		}

		// The following comment marks the place where `operator-sdk` inserts new scaffolded code.
		//+kubebuilder:scaffold:builder

		if enableCentralReconciler {
			if len(centralLabelSelector) > 0 {
				setupLog.Info("using Central label selector ", "selector", centralLabelSelector)
			}
			if err = centralReconciler.RegisterNewReconciler(mgr, centralLabelSelector); err != nil {
				return errors.Wrap(err, "unable to set up Central reconciler")
			}
		}

		if enableSecuredClusterReconciler {
			if err = securedClusterReconciler.RegisterNewReconciler(mgr); err != nil {
				return errors.Wrap(err, "unable to set up SecuredCluster reconciler")
			}
		}

		if err = mgr.AddHealthzCheck(livenessProbeEndpointName, healthz.Ping); err != nil {
			return errors.Wrap(err, "unable to set up health check")
		}

		if err = mgr.AddReadyzCheck(readinessProbeEndpointName, healthz.Ping); err != nil {
			return errors.Wrap(err, "unable to set up readiness check")
		}

		setupLog.Info("starting manager")
		if err = mgr.Start(ctrl.SetupSignalHandler()); err != nil {
			return errors.Wrap(err, "problem running manager")
		}
		return nil

	},
}

func init() {
	rootCmd.AddCommand(&startCmd)
	startCmd.Flags().DurationVar(&syncPeriod, "sync-period", time.Hour*10, "Determines the minimum frequency at which watched resources are reconciled. A lower period will correct entropy more quickly, but reduce responsiveness to change if there are many watched resources.")
	startCmd.Flags().StringVar(&metricsBindAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	startCmd.Flags().StringVar(&healthProbeBindAddrress, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	startCmd.Flags().StringVar(&readinessProbeEndpointName, "readiness-probe-endpoint-name", "/readyz", "Endpoint for the readiness probe")
	startCmd.Flags().StringVar(&livenessProbeEndpointName, "liveness-probe-endpoint-name", "/healthz", "Endpoint for the liveness probe")
	startCmd.Flags().BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	startCmd.Flags().DurationVar(&leaderElectLeaseDuration, "leader-elect-lease-duration", 15*time.Second, "Leader election lease duration")
	startCmd.Flags().DurationVar(&leaderElectRenewDeadline, "leader-elect-renew-deadline", 10*time.Second, "Leader election renew deadline")
	startCmd.Flags().DurationVar(&leaderElectRetryPeriod, "leader-elect-retry-period", 2*time.Second, "Leader election retry period")
	startCmd.Flags().StringVar(&leaderElectNamespace, "leader-elect-namespace", "", "Determines the namespace in which the leader election resource will be created.")
	startCmd.Flags().StringVar(&leaderElectID, "leader-elect-id", "bf7ea6a2.stackrox.io", "Name of the configmap that is used for holding the leader lock.")
	startCmd.Flags().IntVar(&webhookPort, "webhook-bind-port", 9443, "The port the webhook endpoint binds to.")
	startCmd.Flags().StringVar(&webhookHost, "webhook-bind-host", "", "The port the webhook endpoint binds to.")
	startCmd.Flags().BoolVar(&enableWebhooks, "enable-webhooks", true, "Enable webhooks")
	startCmd.Flags().BoolVar(&enableProfiling, "enable-profiling", false, "Enable profiling")
	startCmd.Flags().Float64Var(&profilingThresholdFraction, "profiling-threshold-fraction", 0.8, "Profiling threshold fraction")
	startCmd.Flags().Uint64Var(&memLimit, "memory-limit-bytes", 0, "Memory limit in bytes")
	startCmd.Flags().StringVar(&heapDumpDir, "heap-dump-parent-dir", os.TempDir(), "Heap dump parent directory")
	startCmd.Flags().StringVar(&centralLabelSelector, "central-label-selector", "", "Label selector for Central instances to manage. If empty, all Central instances will be managed.")
	startCmd.Flags().DurationVar(&gracefulShutdownTimeout, "graceful-shutdown-timeout", time.Second*30, "The duration given to runnable to stop before the manager actually returns on stop. To disable graceful shutdown, set to time.Duration(0). To use graceful shutdown without timeout, set to a negative duration, e.G. time.Duration(-1)")
	startCmd.Flags().BoolVar(&dryRunClient, "dry-run-client", false, "Specifies whether the client should be configured to enforce dry-run mode")
	startCmd.Flags().BoolVar(&enableCentralReconciler, "enable-central-reconciler", true, "Specifies whether the Central reconciler should be enabled")
	startCmd.Flags().BoolVar(&enableSecuredClusterReconciler, "enable-secured-cluster-reconciler", true, "Specifies whether the SecuredCluster reconciler should be enabled")

	var goFlags = flag.CommandLine
	ctrl.RegisterFlags(goFlags)
	zapOptions.BindFlags(flag.CommandLine)
	startCmd.Flags().AddGoFlagSet(goFlags)
}

func maybeUseLegacyTLSFileLocation(mgr manager.Manager) {
	// OLM before version 0.17.0 (such as the one shipped with OpenShift 4.6) does not
	// provide the TLS certificate/key in the location referenced by default by the controller runtime
	// (i.e. /tmp/k8s-webhook-server/serving-certs/...).
	// See ROX-8304 for details.
	// If the files are missing at the default location, then we explicitly set the settings as follows
	// to force usage of the legacy location, which is provided both by old and new OLM, but not
	// by the "make deploy" scaffolding.
	if ok, _ := fileutils.AllExist(defaultTLSPaths...); ok {
		return
	}
	setupLog.Info("Webhook key and/or certificate missing at default paths, attempting use of legacy path.", "defaultTLSPaths", defaultTLSPaths)
	server := mgr.GetWebhookServer()
	server.CertDir = "/apiserver.local.config/certificates"
	server.CertName = "apiserver.crt"
	server.KeyName = "apiserver.key"
}
