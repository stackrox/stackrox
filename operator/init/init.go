package operatorinit

import (
	configv1 "github.com/openshift/api/config/v1"
	consolev1 "github.com/openshift/api/console/v1"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

// RegisterCRDTypes registers Central and SecuredCluster CRD types with the platform SchemeBuilder.
// Must be called before AddToScheme is used on the platform package.
func RegisterCRDTypes() {
	platform.SchemeBuilder.Register(&platform.Central{}, &platform.CentralList{})
	platform.SchemeBuilder.Register(&platform.SecuredCluster{}, &platform.SecuredClusterList{})
}

// InitSchemes registers all operator schemes including platform CRDs and dependent Kubernetes schemes.
// Must be called during operator startup before the manager starts.
func InitSchemes(scheme *runtime.Scheme) {
	RegisterCRDTypes()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(platform.AddToScheme(scheme))
	utilruntime.Must(consolev1.Install(scheme))
	utilruntime.Must(configv1.Install(scheme))
}
