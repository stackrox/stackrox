package graphqlgateway

import (
	"crypto/x509"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/graphqlgateway/auth"
	"github.com/stackrox/rox/sensor/common/graphqlgateway/cache"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
)

const (
	// TokenCacheCleanupInterval is how often the token cache runs cleanup
	TokenCacheCleanupInterval = 1 * time.Minute
)

// NewGraphQLGatewayHandler creates a new GraphQL gateway handler with all dependencies.
//
// Parameters:
// - centralEndpoint: The Central HTTP endpoint (e.g., "https://central.stackrox:443")
// - centralCertificates: Central's CA certificates for mTLS
// - k8sClient: Kubernetes client for RBAC validation
// - centralConn: gRPC connection to Central for token requests
// - clusterName: The name of this Sensor's cluster
// - centralSignal: Signal indicating Central connectivity status
func NewGraphQLGatewayHandler(
	centralEndpoint string,
	centralCertificates []*x509.Certificate,
	k8sClient kubernetes.Interface,
	centralConn grpc.ClientConnInterface,
	clusterName string,
	centralSignal concurrency.ReadOnlyErrorSignal,
) (*Handler, error) {
	// Create K8s validator
	k8sValidator := auth.NewK8sValidator(k8sClient)

	// Create token client
	tokenClient := auth.NewTokenClient(centralConn, clusterName)

	// Create token cache
	tokenCache := cache.NewMemoryCache(TokenCacheCleanupInterval)

	// Create token manager
	tokenManager := auth.NewTokenManager(
		k8sValidator,
		tokenClient,
		tokenCache,
		centralSignal,
	)

	// Create handler
	handler, err := NewHandler(
		centralEndpoint,
		centralCertificates,
		k8sClient,
		tokenManager,
	)
	if err != nil {
		return nil, errors.Wrap(err, "creating GraphQL gateway handler")
	}

	return handler, nil
}
