package oidc

import (
	"context"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/authproviders/oidc/internal/endpoint"
	"github.com/stackrox/rox/pkg/sliceutils"
)

// In the go-oidc library the check for the issuer is done strictly, not even tolerating a trailing slash
// (which makes it very hard to omit the `https://` prefix, as is common).
// So far, the go-oidc maintainers refuse to accommodate for this annoyance:
// - https://github.com/coreos/go-oidc/issues/238
// - https://github.com/coreos/go-oidc/issues/221 (and the long list of similar cases listed there)
// We therefore call `oidc.NewProvider` and retry in case of an error, with a trailing
// slash added or removed.
func createOIDCProvider(ctx context.Context, helper *endpoint.Helper, providerFactory providerFactoryFunc) (*informedProvider, error) {
	var err error
	ctx = oidc.ClientContext(ctx, helper.HTTPClient())
	for _, issuer := range helper.URLsForDiscovery() {
		var provider oidcProvider
		if provider, err = providerFactory(ctx, issuer); err == nil {
			// TODO(porridge): as an optimization, we could tell the helper which issuer URL turned out to be correct,
			// so that it can be persisted. This way subsequent instantiations of the backend would hit the correct
			// issuer on the first try, reducing latency.
			// However we would likely still need this logic here indefinitely since the provider could conceivably
			// add or remove a slash in the discovery document at a later time (however unlikely that would be).
			return toInformedProvider(provider), nil
		}
	}
	return nil, err
}

// informedProvider contains a go-oidc provider object and some additional information from discovery
type informedProvider struct {
	oidcProvider
	extraDiscoveryInfo
}

func toInformedProvider(rawProvider oidcProvider) *informedProvider {
	p := &informedProvider{
		oidcProvider: rawProvider,
	}
	if err := p.Claims(&p.extraDiscoveryInfo); err != nil {
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
	return sliceutils.Find(i.ScopesSupported, scope) != -1
}

func (i *extraDiscoveryInfo) SupportsResponseType(responseType string) bool {
	return sliceutils.Find(i.ResponseTypesSupported, responseType) != -1
}

func (i *extraDiscoveryInfo) SupportsResponseMode(responseMode string) bool {
	if i.ResponseModesSupported == nil {
		// Some providers do not set this (Google). Assume all modes are supported.
		return true
	}
	return sliceutils.Find(i.ResponseModesSupported, responseMode) != -1
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
	// code flow is always preferable, but only works if we have a client secret. Worse, some providers
	// (e.g., Auth0) will actually not allow issuing a code if the response mode is fragment; so we
	// would rather additionally avoid it in that case.
	if hasClientSecret && responseMode != "fragment" {
		// Some providers (e.g. Ping Federate and KeyCloak) disallow implicit and hybrid flows unless
		// explicitly permitted by the administrator. Therefore we prefer pure code flow over hybrid.
		// See https://stack-rox.atlassian.net/browse/ROX-6497 for background.
		// TODO(mowsiany): consider exposing a knob which lets administrator prefer hybrid flow if they know what they are doing.
		// Not sure if hybrid flow actually buys us anything, since we go to the token endpoint anyway,
		// nullifying any latency gains from having received the (id_)token in authorization response.
		preferredResponseTypes = append(preferredResponseTypes, "code")

		// Hybrid flow only works in non-query mode
		if responseMode != "query" {
			// Note: "code id_token token" is the canonical response type per
			// https://openid.net/specs/oauth-v2-multiple-response-types-1_0.html#Combinations, but some providers
			// (e.g. Google, Auth0, PingFederate) have the order swapped, listing it as "token id_token".
			preferredResponseTypes = append(preferredResponseTypes, "code id_token token", "code token id_token", "code token", "code id_token")
		}
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
		if sliceutils.Find(options, pref) != -1 {
			return pref, true
		}
	}
	return "", false
}
