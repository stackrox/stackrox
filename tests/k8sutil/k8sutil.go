// Package k8sutil provides small client-go helpers shared across StackRox
// e2e test suites, such as configuring a retryable REST transport before
// creating a Kubernetes client.
package k8sutil

import (
	"time"

	"github.com/stackrox/rox/pkg/retryablehttp"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// testutilsLogger adapts testutils.T to retryablehttp.Logger interface.
//
// WHY THIS EXISTS (and why it's unfortunate):
// The retryablehttp library requires a logger implementing its Logger interface with Printf method.
// Go's *testing.T has Logf but NOT Printf (different method names, same functionality).
// We cannot add Printf to testutils.T because that would break compatibility with *testing.T,
// which is used throughout the codebase (for example, centralgrpc.GRPCConnectionToCentral(t testutils.T)).
//
// CLEANER ALTERNATIVE:
// Accept *testing.T directly in helper functions and create testutils.T wrappers locally where needed
// (specifically for the retry mechanism). This would avoid the interface constraint propagating everywhere.
// However, this would be a larger refactor affecting many test helper functions.
//
// CURRENT COMPROMISE:
// Use this tiny adapter ONLY where retryablehttp requires it. Everywhere else uses testutils.T naturally.
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
