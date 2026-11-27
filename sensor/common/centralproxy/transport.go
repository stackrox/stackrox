package centralproxy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	centralv1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/coalescer"
	"github.com/stackrox/rox/pkg/expiringcache"
	pkghttputil "github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"
)

const (
	// tokenCacheTTL is how long tokens are cached locally before being refreshed.
	// Tokens contain administrator-managed permissions. Caching reduces load on
	// Central but permission reductions may take up to the token lifetime to apply.
	tokenCacheTTL = 55 * time.Minute

	// tokenTTL is the requested token lifetime.
	// Slightly longer than cache TTL to ensure tokens remain valid during cache lifetime.
	tokenTTL = tokenCacheTTL + 5*time.Minute

	// tokenRequestTimeout is the maximum time allowed for a token request RPC.
	// This ensures that token requests don't hang indefinitely when all callers have cancelled.
	tokenRequestTimeout = 30 * time.Second

	// maxRequestBodySize is the maximum size of a proxied request body that will
	// be buffered for retry. 32 MiB is generous for the query/filter payloads
	// this proxy handles while still guarding against memory exhaustion.
	maxRequestBodySize = 32 << 20

	// FullClusterAccessScope is the namespace scope value that indicates full cluster access.
	FullClusterAccessScope = "*"
)

// tokenPermissions defines the required read permissions for proxy requests.
var tokenPermissions = map[string]centralv1.Access{
	"Deployment": centralv1.Access_READ_ACCESS,
	"Image":      centralv1.Access_READ_ACCESS,
}

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
// Tokens are injected per-request based on the authorized client profile.
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

// scopedTokenTransport injects a token for the profile authorized by the handler.
type scopedTokenTransport struct {
	base          http.RoundTripper
	tokenProvider *tokenProvider
}

// newScopedTokenTransport creates a new transport that wraps the base transport
// and injects tokens based on the authenticated client profile.
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
// It reads the authorized profile from the request, obtains an appropriate token,
// and injects it into the Authorization header before forwarding the request.
// If Central returns 401 or 403, the cached token is invalidated and the request
// is retried once with a fresh token.
func (t *scopedTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	profile := proxyTokenProfileFromContext(req.Context())
	if profile == nil {
		return nil, errors.New("authorized proxy client profile is missing")
	}

	// Buffer the request body upfront so we can replay it on retry.
	var bodyBytes []byte
	if req.Body != nil && req.Body != http.NoBody {
		defer utils.IgnoreError(req.Body.Close)
		var err error
		bodyBytes, err = io.ReadAll(io.LimitReader(req.Body, maxRequestBodySize+1))
		if err != nil {
			return nil, errors.Wrap(err, "reading request body")
		}
		if len(bodyBytes) > maxRequestBodySize {
			return nil, errors.Errorf("request body exceeds maximum allowed size of %d bytes", maxRequestBodySize)
		}
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	resp, err := t.doRoundTrip(req, profile)
	if err != nil {
		return nil, err
	}

	// If Central returns Unauthorized or Forbidden, invalidate the cached token
	// and retry once with a fresh token.
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		// Consume and close the first response body to reuse the connection.
		if resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			utils.IgnoreError(resp.Body.Close)
		}

		// Restore the request body for retry.
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		t.tokenProvider.invalidateProfile(profile)
		return t.doRoundTrip(req, profile)
	}

	return resp, nil
}

// doRoundTrip performs a single round trip with token injection.
func (t *scopedTokenTransport) doRoundTrip(req *http.Request, profile *proxyTokenProfile) (*http.Response, error) {
	token, err := t.tokenProvider.getTokenForProfile(req.Context(), profile)
	if err != nil {
		return nil, errors.Wrap(err, "obtaining authorization token")
	}

	// Clone the request to avoid modifying the original.
	reqCopy := req.Clone(req.Context())
	reqCopy.Header.Del(stackroxNamespaceHeader)
	reqCopy.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	return t.base.RoundTrip(reqCopy) //nolint:wrapcheck
}

// tokenProvider manages dynamic token acquisition from Central.
type tokenProvider struct {
	// client holds the Central gRPC client. It is stored atomically to safeguard
	// against data races.
	client          atomic.Pointer[centralv1.TokenServiceClient]
	clusterIDGetter clusterIDGetter
	tokenCache      expiringcache.Cache[string, string]
	// tokenGroup coalesces concurrent token requests for the same client profile.
	tokenGroup *coalescer.Coalescer[string]
}

// newTokenProvider creates a new tokenProvider.
func newTokenProvider(clusterIDGetter clusterIDGetter) *tokenProvider {
	return &tokenProvider{
		clusterIDGetter: clusterIDGetter,
		tokenCache:      expiringcache.NewExpiringCache[string, string](tokenCacheTTL),
		tokenGroup:      coalescer.New[string](),
	}
}

// setClient sets the gRPC client connection to Central.
// This method is safe to call concurrently with getTokenForProfile.
func (p *tokenProvider) setClient(conn grpc.ClientConnInterface) {
	if conn == nil {
		return
	}
	client := centralv1.NewTokenServiceClient(conn)
	p.client.Store(&client)
}

// getTokenForProfile coalesces concurrent requests for the same authorized client.
func (p *tokenProvider) getTokenForProfile(ctx context.Context, profile *proxyTokenProfile) (string, error) {
	client := p.client.Load()
	if client == nil {
		incrementTokenRequest(tokenResultError)
		return "", errors.Wrap(errServiceUnavailable, "token provider not initialized: central connection not available")
	}
	req, err := p.buildProfileTokenRequest(profile)
	if err != nil {
		incrementTokenRequest(tokenResultError)
		return "", errors.Wrap(err, "building token request")
	}
	cacheKey := p.profileCacheKey(profile)

	// Fast path: check cache first.
	if token, ok := p.tokenCache.Get(cacheKey); ok {
		incrementTokenRequest(tokenResultCacheHit)
		return token, nil
	}

	// Slow path: coalesce concurrent requests for the same profile.
	return p.tokenGroup.Coalesce(ctx, cacheKey, func() (string, error) { //nolint:wrapcheck
		// Double-check cache inside coalesce to avoid redundant API calls.
		if token, ok := p.tokenCache.Get(cacheKey); ok {
			incrementTokenRequest(tokenResultCacheHit)
			return token, nil
		}

		log.Debugf("Token cache miss for proxy client %q, requesting from Central", profile.identity)

		// Use a background context with timeout to ensure the shared function is independent
		// of the initial request context while still having a bounded lifetime.
		ctx, cancel := context.WithTimeout(context.Background(), tokenRequestTimeout)
		defer cancel()
		token, err := p.requestToken(ctx, *client, req)
		if err != nil {
			incrementTokenRequest(tokenResultError)
			return "", err
		}

		p.tokenCache.Add(cacheKey, token)
		incrementTokenRequest(tokenResultSuccess)
		return token, nil
	})
}

// requestToken performs the RPC call to Central to generate a token for the given scope.
func (p *tokenProvider) requestToken(ctx context.Context, client centralv1.TokenServiceClient, req *centralv1.GenerateTokenForPermissionsAndScopeRequest) (string, error) {
	resp, err := client.GenerateTokenForPermissionsAndScope(ctx, req)
	if err != nil {
		return "", errors.Wrap(err, "requesting token from Central")
	}

	token := resp.GetToken()
	if token == "" {
		return "", errors.New("received empty token from Central")
	}

	return token, nil
}

// invalidateProfile removes the cached token for the authorized client.
// It also removes the coalescer key so subsequent callers will
// trigger a fresh token request rather than joining any in-progress request.
// This does not cancel already running requests.
func (p *tokenProvider) invalidateProfile(profile *proxyTokenProfile) {
	cacheKey := p.profileCacheKey(profile)
	p.tokenCache.Remove(cacheKey)
	p.tokenGroup.Forget(cacheKey)
}

func (p *tokenProvider) profileCacheKey(profile *proxyTokenProfile) string {
	// Include identity and all signed claims so different clients never share tokens.
	data, _ := json.Marshal(struct {
		Identity            string
		PermissionSets      []*centralv1.SelectedPermissionSet
		SinglePermissionSet bool
		ClusterScope        string
		Legacy              bool
	}{profile.identity, profile.permissionSets, profile.singlePermissionSet, profile.clusterScope, profile.legacy})
	return fmt.Sprintf("%s:%x", p.clusterIDGetter.GetNoWait(), sha256.Sum256(data))
}

// buildProfileTokenRequest creates a token request from the authenticated profile.
// Returns an error if the cluster ID is not available yet.
func (p *tokenProvider) buildProfileTokenRequest(profile *proxyTokenProfile) (*centralv1.GenerateTokenForPermissionsAndScopeRequest, error) {
	clusterID := p.clusterIDGetter.GetNoWait()
	if clusterID == "" {
		return nil, errors.Wrap(errServiceUnavailable, "cluster ID not available")
	}

	req := &centralv1.GenerateTokenForPermissionsAndScopeRequest{
		Lifetime:  durationpb.New(tokenTTL),
		Requester: profile.identity,
	}
	if profile.legacy || profile.singlePermissionSet {
		if len(profile.permissionSets) > 0 &&
			(!safeConsoleFallback(profile.permissionSets) || (profile.legacy && profile.singlePermissionSet)) {
			return nil, errors.New("fallback token flow is only available to safely scoped console clients")
		}
	}
	if profile.legacy {
		// Older Centrals only know the original token forms: no permissions for
		// authentication-only grants and the console grant's Deployment/Image
		// read permissions otherwise.
		if len(profile.permissionSets) > 0 {
			req.Permissions = tokenPermissions
		}
	} else if profile.singlePermissionSet {
		if len(profile.permissionSets) > 0 {
			grant := profile.permissionSets[0]
			if grant.GetId() != "" {
				req.PermissionSets = []*centralv1.SelectedPermissionSet{{
					Selector: &centralv1.SelectedPermissionSet_Id{Id: grant.GetId()},
				}}
			} else {
				req.PermissionSets = []*centralv1.SelectedPermissionSet{{
					Selector: &centralv1.SelectedPermissionSet_Name{Name: grant.GetName()},
				}}
			}
		}
	} else {
		req.PermissionSets = profile.permissionSets
	}

	switch profile.clusterScope {
	case "":
		// Empty scope: no cluster scopes (authentication only).
	case FullClusterAccessScope:
		// Cluster-wide access.
		req.ClusterScopes = []*centralv1.ClusterScope{{
			ClusterId:         clusterID,
			FullClusterAccess: true,
		}}
	default:
		// Specific namespace.
		req.ClusterScopes = []*centralv1.ClusterScope{{
			ClusterId:  clusterID,
			Namespaces: []string{profile.clusterScope},
		}}
	}

	return req, nil
}
