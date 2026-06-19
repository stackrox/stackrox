package vmhelpers

import (
	"time"

	"github.com/stackrox/rox/pkg/retryablehttp"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// testutilsLogger adapts testutils.T to retryablehttp.Logger interface.
type testutilsLogger struct{ testutils.T }

func (l testutilsLogger) Printf(format string, v ...any) { l.Logf(format, v...) }

// ConfigureRetryableTransport configures a rest.Config to use retryable HTTP client
// for network resilience. This adds automatic retry logic for transient network errors.
func ConfigureRetryableTransport(t testutils.T, restCfg *rest.Config) {
	if restCfg.Timeout == 0 {
		restCfg.Timeout = 30 * time.Second
	}
	retryablehttp.ConfigureRESTConfig(restCfg,
		retryablehttp.WithLogger(&testutilsLogger{t}),
	)
}

// CreateK8sClientWithConfig creates a Kubernetes client from a prepared REST config.
func CreateK8sClientWithConfig(t testutils.T, restCfg *rest.Config) kubernetes.Interface {
	k8sClient, err := kubernetes.NewForConfig(restCfg)
	require.NoError(t, err, "creating Kubernetes client from REST config")

	return k8sClient
}
