package olm

import (
	"github.com/pkg/errors"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	// Activate auth providers (OIDC/exec) so kubeconfig contexts from oc/kubectl work.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// NewDynamicClient builds a Kubernetes dynamic client from the standard kubeconfig loading
// rules (honoring KUBECONFIG and the current context). An explicit kubeconfig path overrides
// the default rules. This mirrors roxctl/helm/derivelocalvalues/live.go's loadKubeCtlConfig.
func NewDynamicClient(kubeconfig string) (dynamic.Interface, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfig != "" {
		loadingRules.ExplicitPath = kubeconfig
	}
	raw, err := loadingRules.Load()
	if err != nil {
		return nil, errors.Wrap(err, "loading kubeconfig")
	}
	config, err := clientcmd.NewDefaultClientConfig(*raw, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "building Kubernetes REST config")
	}
	client, err := dynamic.NewForConfig(config)
	return client, errors.Wrap(err, "creating Kubernetes dynamic client")
}
