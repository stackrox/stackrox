package vmhelpers

import (
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/tests/k8sutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ConfigureRetryableTransport configures a rest.Config to use retryable HTTP client
// for network resilience. This adds automatic retry logic for transient network errors.
func ConfigureRetryableTransport(t testutils.T, restCfg *rest.Config) {
	k8sutil.ConfigureRetryableTransport(t, restCfg)
}

// CreateK8sClientWithConfig creates a Kubernetes client from a prepared REST config.
func CreateK8sClientWithConfig(t testutils.T, restCfg *rest.Config) kubernetes.Interface {
	return k8sutil.CreateK8sClientWithConfig(t, restCfg)
}
