package app

import (
	configv1alpha1 "github.com/stackrox/rox/config-controller/api/v1alpha1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

// Init registers all schemes and types required by config-controller.
// Called explicitly from Run() instead of package init().
func Init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(configv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme

	configv1alpha1.RegisterSecurityPolicy()
}
