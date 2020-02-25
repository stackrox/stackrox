package oidc

import (
	"context"
	"strings"

	"github.com/coreos/go-oidc"
	"github.com/stackrox/rox/pkg/sliceutils"
)

// The go-oidc library has two annoying characteristics when it comes to creating backendImpl instances:
// - The context is passed on to the remoteKeySource that is being created. Hence, we can't use a short-lived context
//   (such as the request context), as otherwise subsequent verifications will fail because the keys have not been
//   retrieved.
// - The check for the issuer is done strictly, not even tolerating a trailing slash (which makes it very hard to omit
//   the `https://` prefix, as is common).
// We therefore add a wrapper method that calls `oidc.NewProvider` with the background context and writes the result to
// a channel, and retries in case of an error with a trailing slash added or removed.
//
type createOIDCProviderResult struct {
	issuer   string
	provider *oidc.Provider
	err      error
}

func createOIDCProviderAsync(issuer string, resultC chan<- createOIDCProviderResult) {
	provider, err := oidc.NewProvider(context.Background(), issuer)
	if err != nil {
		if strings.HasSuffix(issuer, "/") {
			issuer = strings.TrimSuffix(issuer, "/")
		} else {
			issuer = issuer + "/"
		}
		provider, err = oidc.NewProvider(context.Background(), issuer)
	}
	resultC <- createOIDCProviderResult{issuer: issuer, provider: provider, err: err}
}

func createOIDCProvider(ctx context.Context, issuer string) (*oidc.Provider, string, error) {
	resultC := make(chan createOIDCProviderResult, 1)
	go createOIDCProviderAsync(issuer, resultC)
	select {
	case res := <-resultC:
		return res.provider, res.issuer, res.err
	case <-ctx.Done():
		return nil, "", ctx.Err()
	}
}

type provider struct {
	*oidc.Provider
	extraDiscoveryInfo
}

func wrapProvider(rawProvider *oidc.Provider) *provider {
	p := &provider{
		Provider: rawProvider,
	}
	if err := p.Provider.Claims(&p.extraDiscoveryInfo); err != nil {
		log.Warnf("Failed to parse additional provider discovery claims: %v", err)
	}
	return p
}

type extraDiscoveryInfo struct {
	ScopesSupported    []string `json:"scopes_supported,omitempty"`
	RevocationEndpoint string   `json:"revocation_endpoint,omitempty"`
}

func (i *extraDiscoveryInfo) SupportsScope(scope string) bool {
	return sliceutils.StringFind(i.ScopesSupported, scope) != -1
}
