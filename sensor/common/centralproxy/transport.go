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
	// Tokens contain administrator-managed permissions. A longer cache window
	// reduces token requests but delays permission reductions for up to the
	// token lifetime.
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
// Tokens are injected per-request based on the authorized profile.
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

// scopedTokenTransport injects tokens for the profile authorized by the handler.
type scopedTokenTransport struct {
	base          http.RoundTripper
	tokenProvider *tokenProvider
}

// newScopedTokenTransport creates a new transport that wraps the base transport
// and injects tokens based on the authenticated request profile.
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

	resp, err := t.doProfileRoundTrip(req, profile)
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
		return t.doProfileRoundTrip(req, profile)
	}

	return resp, nil
}

func (t *scopedTokenTransport) doProfileRoundTrip(req *http.Request, profile *proxyTokenProfile) (*http.Response, error) {
	token, err := t.tokenProvider.getTokenForProfile(req.Context(), profile)
	if err != nil {
		return nil, errors.Wrap(err, "obtaining proxy client token")
	}
	reqCopy := req.Clone(req.Context())
	reqCopy.Header.Del(stackroxNamespaceHeader)
	reqCopy.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	return t.base.RoundTrip(reqCopy) //nolint:wrapcheck
}

// tokenProvider manages dynamic token acquisition from Central.
type tokenProvider struct {
	// client holds the Central gRPC client. It is stored atomically to safeguard
	// against data races.
	client            atomic.Pointer[centralv1.TokenServiceClient]
	clusterIDGetter   clusterIDGetter
	profileTokenCache expiringcache.Cache[string, string]
	profileTokenGroup *coalescer.Coalescer[string]
}

// newTokenProvider creates a new tokenProvider.
func newTokenProvider(clusterIDGetter clusterIDGetter) *tokenProvider {
	return &tokenProvider{
		clusterIDGetter:   clusterIDGetter,
		profileTokenCache: expiringcache.NewExpiringCache[string, string](tokenCacheTTL),
		profileTokenGroup: coalescer.New[string](),
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

func profileCacheKey(profile *proxyTokenProfile) string {
	// Include the identity, scope and all signed claims. Different clients must
	// never reuse each other's bearer tokens, even with the same permission set.
	data, _ := json.Marshal(struct {
		Identity       string
		PermissionSets []*centralv1.ScopedPermissionSet
		ClusterScope   string
		LegacyConsole  bool
	}{profile.identity, profile.permissionSets, profile.clusterScope, profile.legacyConsole})
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func (p *tokenProvider) getTokenForProfile(ctx context.Context, profile *proxyTokenProfile) (string, error) {
	client := p.client.Load()
	if client == nil {
		incrementTokenRequest(tokenResultError)
		return "", errors.Wrap(errServiceUnavailable, "token provider not initialized")
	}
	key := p.clusterIDGetter.GetNoWait() + ":" + profileCacheKey(profile)
	if token, ok := p.profileTokenCache.Get(key); ok {
		incrementTokenRequest(tokenResultCacheHit)
		return token, nil
	}
	return p.profileTokenGroup.Coalesce(ctx, key, func() (string, error) { //nolint:wrapcheck
		if token, ok := p.profileTokenCache.Get(key); ok {
			incrementTokenRequest(tokenResultCacheHit)
			return token, nil
		}
		// A canceled caller must not cancel an RPC shared with other requests.
		requestCtx, cancel := context.WithTimeout(context.Background(), tokenRequestTimeout)
		defer cancel()
		token, err := p.requestProfileToken(requestCtx, *client, profile)
		if err != nil {
			incrementTokenRequest(tokenResultError)
			return "", err
		}
		p.profileTokenCache.Add(key, token)
		incrementTokenRequest(tokenResultSuccess)
		return token, nil
	})
}

func (p *tokenProvider) requestProfileToken(ctx context.Context, client centralv1.TokenServiceClient,
	profile *proxyTokenProfile,
) (string, error) {
	clusterID := p.clusterIDGetter.GetNoWait()
	if clusterID == "" {
		return "", errors.Wrap(errServiceUnavailable, "cluster ID not available")
	}
	req := &centralv1.GenerateTokenForPermissionsAndScopeRequest{
		Requester: profile.identity,
		Lifetime:  durationpb.New(tokenTTL),
	}
	if profile.legacyConsole {
		req.Permissions = map[string]centralv1.Access{
			"Deployment": centralv1.Access_READ_ACCESS,
			"Image":      centralv1.Access_READ_ACCESS,
		}
	} else {
		req.PermissionSets = profile.permissionSets
	}
	if scope := profile.clusterScope; scope != "" {
		clusterScope := &centralv1.ClusterScope{ClusterId: clusterID}
		if scope == FullClusterAccessScope {
			clusterScope.FullClusterAccess = true
		} else {
			clusterScope.Namespaces = []string{scope}
		}
		req.ClusterScopes = []*centralv1.ClusterScope{clusterScope}
	}
	response, err := client.GenerateTokenForPermissionsAndScope(ctx, req)
	if err != nil {
		return "", errors.Wrap(err, "requesting proxy client token")
	}
	if response.GetToken() == "" {
		return "", errors.New("received empty proxy client token from Central")
	}
	return response.GetToken(), nil
}

func (p *tokenProvider) invalidateProfile(profile *proxyTokenProfile) {
	key := p.clusterIDGetter.GetNoWait() + ":" + profileCacheKey(profile)
	p.profileTokenCache.Remove(key)
	p.profileTokenGroup.Forget(key)
}
