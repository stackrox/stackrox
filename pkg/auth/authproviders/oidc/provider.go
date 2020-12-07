package oidc

import (
	"context"
	"strings"

	"github.com/coreos/go-oidc"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/contextutil"
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

func createOIDCProviderAsync(ctx context.Context, issuer string, resultC chan<- createOIDCProviderResult) {
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		if strings.HasSuffix(issuer, "/") {
			issuer = strings.TrimSuffix(issuer, "/")
		} else {
			issuer = issuer + "/"
		}
		provider, err = oidc.NewProvider(ctx, issuer)
	}

	select {
	case resultC <- createOIDCProviderResult{issuer: issuer, provider: provider, err: err}:
	case <-ctx.Done():
	}
}

func createOIDCProvider(ctx context.Context, issuer string) (*oidc.Provider, string, error) {
	resultC := make(chan createOIDCProviderResult, 1)
	go createOIDCProviderAsync(contextutil.WithValuesFrom(context.Background(), ctx), issuer, resultC)
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
	ScopesSupported        []string `json:"scopes_supported,omitempty"`
	RevocationEndpoint     string   `json:"revocation_endpoint,omitempty"`
	ResponseTypesSupported []string `json:"response_types_supported,omitempty"`
	ResponseModesSupported []string `json:"response_modes_supported,omitempty"`
}

func (i *extraDiscoveryInfo) SupportsScope(scope string) bool {
	return sliceutils.StringFind(i.ScopesSupported, scope) != -1
}

func (i *extraDiscoveryInfo) SupportsResponseType(responseType string) bool {
	return sliceutils.StringFind(i.ResponseTypesSupported, responseType) != -1
}

func (i *extraDiscoveryInfo) SupportsResponseMode(responseMode string) bool {
	if i.ResponseModesSupported == nil {
		// Some providers do not set this (Google). Assume all modes are supported.
		return true
	}
	return sliceutils.StringFind(i.ResponseModesSupported, responseMode) != -1
}

func (i *extraDiscoveryInfo) SelectResponseMode(hasClientSecret bool) (string, error) {
	preferredResponseModes := []string{"form_post", "fragment", "query"}
	// If we want to use the code flow, actually prefer query over fragment, as most providers (e.g., Auth0)
	// will refuse to transmit a code via a fragment.
	if hasClientSecret && i.SupportsResponseType("code") {
		preferredResponseModes = []string{"form_post", "query", "fragment"}
	}

	if i.ResponseModesSupported == nil {
		// Some providers do not set this (Google). Assume all modes are supported and we can pick our
		// first preference.
		return preferredResponseModes[0], nil
	}
	responseMode, ok := selectPreferred(i.ResponseModesSupported, preferredResponseModes)
	if !ok {
		return "", errors.Errorf("could not determine a suitable response mode, supported modes are: %s", strings.Join(i.ResponseModesSupported, ", "))
	}
	return responseMode, nil
}

func (i *extraDiscoveryInfo) SelectResponseType(responseMode string, hasClientSecret bool) (string, error) {
	var preferredResponseTypes []string
	if hasClientSecret && responseMode != "fragment" {
		// code flow is always preferable, but only works if we have a client secret. Worse, some providers
		// (e.g., Auth0) will actually not allow issuing a code if the response mode is fragment; so we
		// would rather avoid it.

		// Hybrid flow only works in non-query mode
		if responseMode != "query" {
			// Note: "code id_token token" is the canonical response type per
			// https://openid.net/specs/oauth-v2-multiple-response-types-1_0.html#Combinations, but some providers
			// (Google) have the order swapped, listing it as "token id_token".
			preferredResponseTypes = append(preferredResponseTypes, "code id_token token", "code token id_token", "code token", "code id_token")
		}
		preferredResponseTypes = append(preferredResponseTypes, "code")
	}
	if responseMode != "query" {
		// token and id_token are not allowed to be used with the query response mode.
		// See above regarding "id_token token" vs. "token id_token".
		preferredResponseTypes = append(preferredResponseTypes, "id_token token", "token id_token", "token", "id_token")
	} else if !hasClientSecret {
		return "", errors.New("the query response type can only be used with a client secret")
	}

	responseType, ok := selectPreferred(i.ResponseTypesSupported, preferredResponseTypes)
	if !ok {
		return "", errors.Errorf("could not determine a suitable response type, supported types are: %s", strings.Join(i.ResponseTypesSupported, ", "))
	}

	return responseType, nil
}

func selectPreferred(options, preferences []string) (string, bool) {
	for _, pref := range preferences {
		if sliceutils.StringFind(options, pref) != -1 {
			return pref, true
		}
	}
	return "", false
}
