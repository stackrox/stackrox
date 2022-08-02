package manager

import (
	"os"

	"github.com/stackrox/rox/pkg/logging"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	authproviderv1beta1 "github.com/stackrox/rox/apiskubebuilder/api/v1beta1"
	"github.com/stackrox/rox/apiskubebuilder/controllers"
)

var (
	scheme = runtime.NewScheme()
	log    = logging.CreatePersistentLogger(logging.CurrentModule(), 0)
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(authproviderv1beta1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func StartControllerManager() {
	log.Info("Start controller manager")
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		//MetricsBindAddress:     metricsAddr,
		Port: 9443,
		//HealthProbeBindAddress: probeAddr,
		LeaderElection:   false,
		LeaderElectionID: "c51fd077.central.stackrox.io",
	})

	if err != nil {
		log.Info("Failed to start controller manager")
		os.Exit(1)
	}

	if err = (&controllers.AuthProviderReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create controller", "controller", "AuthProvider")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "problem running manager")
		os.Exit(1)
	}
}
