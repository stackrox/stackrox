package centralproxy

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	centralv1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/expiringcache"
	pkghttputil "github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"
)

const (
	// tokenCacheTTL is how long tokens are cached locally before being refreshed.
	tokenCacheTTL = 3 * time.Minute

	// tokenTTL is the requested token validity duration.
	// Slightly longer than cache TTL to ensure tokens remain valid during cache lifetime.
	tokenTTL = 4 * time.Minute

	// Required read permissions for proxy requests.
	permissionDeployment = "Deployment"
	permissionImage      = "Image"

	// FullClusterAccessScope is the namespace scope value that indicates full cluster access.
	FullClusterAccessScope = "*"
)

// errServiceUnavailable indicates the proxy is temporarily unavailable,
// typically during sensor startup before Central connection is established.
// This error should result in a 503 Service Unavailable response.
var errServiceUnavailable = errors.New("service temporarily unavailable")

// clusterIDGetter provides non-blocking access to the cluster ID.
type clusterIDGetter interface {
	GetNoWait() string
}

// createBaseTransport creates the base HTTP transport for Central communication.
// This transport handles TLS but does NOT inject authorization tokens.
// Tokens are injected per-request based on the namespace scope.
func createBaseTransport(baseURL *url.URL, certs []*x509.Certificate) (http.RoundTripper, error) {
	certPool, err := verifier.SystemCertPool()
	if err != nil {
		return nil, errors.Wrap(err, "getting system cert pool")
	}
	for _, cert := range certs {
		certPool.AddCert(cert)
	}

	// Use TLSConfig without client certificate to keep authn/z based on the user's
	// Bearer token, but include serviceCertFallbackVerifier for proper StackRox service
	// cert handling.
	tlsConf, err := clientconn.TLSConfig(mtls.CentralSubject, clientconn.TLSConfigOptions{
		ServerName:    baseURL.Hostname(),
		RootCAs:       certPool,
		UseClientCert: clientconn.DontUseClientCert,
	})
	if err != nil {
		return nil, errors.Wrap(err, "creating TLS config")
	}

	baseTransport := pkghttputil.DefaultTransport()
	baseTransport.TLSClientConfig = tlsConf

	return baseTransport, nil
}

// scopedTokenTransport is an http.RoundTripper that injects scope-appropriate tokens
// into requests before forwarding them. The scope is determined by reading the
// ACS-AUTH-NAMESPACE-SCOPE header from the request.
type scopedTokenTransport struct {
	base          http.RoundTripper
	tokenProvider *tokenProvider
}

// newScopedTokenTransport creates a new transport that wraps the base transport
// and injects tokens based on the namespace scope header.
func newScopedTokenTransport(base http.RoundTripper, clusterIDGetter clusterIDGetter) *scopedTokenTransport {
	return &scopedTokenTransport{
		base:          base,
		tokenProvider: newTokenProvider(clusterIDGetter),
	}
}

// SetClient sets the gRPC client connection to Central.
// Must be called before the transport can successfully inject tokens.
func (t *scopedTokenTransport) SetClient(conn grpc.ClientConnInterface) {
	t.tokenProvider.setClient(conn)
}

// RoundTrip implements http.RoundTripper.
// It reads the namespace scope from the request, obtains an appropriate token,
// and injects it into the Authorization header before forwarding the request.
func (t *scopedTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	scope := req.Header.Get(stackroxNamespaceHeader)

	token, err := t.tokenProvider.getTokenForScope(req.Context(), scope)
	if err != nil {
		return nil, errors.Wrap(err, "obtaining authorization token")
	}

	// Clone the request to avoid modifying the original.
	reqCopy := req.Clone(req.Context())
	reqCopy.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	return t.base.RoundTrip(reqCopy) //nolint:wrapcheck
}

// tokenProvider manages dynamic token acquisition from Central.
type tokenProvider struct {
	client          centralv1.TokenServiceClient
	clusterIDGetter clusterIDGetter
	tokenCache      expiringcache.Cache[string, string]
}

// newTokenProvider creates a new tokenProvider.
func newTokenProvider(clusterIDGetter clusterIDGetter) *tokenProvider {
	return &tokenProvider{
		clusterIDGetter: clusterIDGetter,
		tokenCache:      expiringcache.NewExpiringCache[string, string](tokenCacheTTL),
	}
}

// setClient sets the gRPC client connection to Central.
func (p *tokenProvider) setClient(conn grpc.ClientConnInterface) {
	p.client = centralv1.NewTokenServiceClient(conn)
}

// getTokenForScope returns a token for the given namespace scope.
// Scope values:
//   - "" (empty): Token with empty access scope (authentication only)
//   - "<namespace>": Token scoped to the specific namespace
//   - FullClusterAccessScope ("*"): Token with full cluster access
func (p *tokenProvider) getTokenForScope(ctx context.Context, namespaceScope string) (string, error) {
	if p.client == nil {
		return "", errors.Wrap(errServiceUnavailable, "token provider not initialized: central connection not available")
	}

	if token, ok := p.tokenCache.Get(namespaceScope); ok {
		return token, nil
	}

	log.Debugf("Token cache miss for namespace scope %q, requesting from Central", namespaceScope)

	req, err := p.buildTokenRequest(namespaceScope)
	if err != nil {
		return "", errors.Wrap(err, "building token request")
	}
	resp, err := p.client.GenerateTokenForPermissionsAndScope(ctx, req)
	if err != nil {
		return "", errors.Wrapf(err, "requesting token from Central for scope %q", namespaceScope)
	}

	token := resp.GetToken()
	if token == "" {
		return "", errors.Errorf("received empty token from Central for scope %q", namespaceScope)
	}

	p.tokenCache.Add(namespaceScope, token)

	return token, nil
}

// buildTokenRequest creates the token request based on the namespace scope.
// Returns an error if the cluster ID is not available yet.
func (p *tokenProvider) buildTokenRequest(namespaceScope string) (*centralv1.GenerateTokenForPermissionsAndScopeRequest, error) {
	clusterID := p.clusterIDGetter.GetNoWait()
	if clusterID == "" {
		return nil, errors.Wrap(errServiceUnavailable, "cluster ID not available")
	}

	req := &centralv1.GenerateTokenForPermissionsAndScopeRequest{
		Permissions: map[string]centralv1.Access{
			permissionDeployment: centralv1.Access_READ_ACCESS,
			permissionImage:      centralv1.Access_READ_ACCESS,
		},
		Lifetime: durationpb.New(tokenTTL),
	}

	switch namespaceScope {
	case "":
		// Empty scope: no cluster scopes (authentication only)
		// ClusterScopes left nil

	case FullClusterAccessScope:
		// Cluster-wide access
		req.ClusterScopes = []*centralv1.ClusterScope{{
			ClusterId:         clusterID,
			FullClusterAccess: true,
		}}

	default:
		// Specific namespace
		req.ClusterScopes = []*centralv1.ClusterScope{{
			ClusterId:  clusterID,
			Namespaces: []string{namespaceScope},
		}}
	}

	return req, nil
}
