package oidc

import (
	"context"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// oidcProvider is an abstraction of oidc.Provider which adds a level of indirection used in testing.
type oidcProvider interface {
	Claims(v *extraDiscoveryInfo) error
	Endpoint() oauth2.Endpoint
	UserInfo(ctx context.Context, tokenSource oauth2.TokenSource) (oidcUserInfo, error)
	Verifier(config *oidc.Config) oidcIDTokenVerifier
}

// wrappedOIDCProvider simply delegates to the oidc.Provider it contains.
type wrappedOIDCProvider struct {
	provider *oidc.Provider
}

type providerFactoryFunc func(ctx context.Context, issuer string) (oidcProvider, error)

func newWrappedOIDCProvider(ctx context.Context, issuer string) (oidcProvider, error) {
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}
	return wrappedOIDCProvider{provider: provider}, nil
}

func (w wrappedOIDCProvider) Claims(v *extraDiscoveryInfo) error {
	return w.provider.Claims(v)
}

func (w wrappedOIDCProvider) Endpoint() oauth2.Endpoint {
	return w.provider.Endpoint()
}

func (w wrappedOIDCProvider) UserInfo(ctx context.Context, tokenSource oauth2.TokenSource) (oidcUserInfo, error) {
	return w.provider.UserInfo(ctx, tokenSource)
}

func (w wrappedOIDCProvider) Verifier(config *oidc.Config) oidcIDTokenVerifier {
	return wrappedOIDCIDTokenVerifier{verifier: w.provider.Verifier(config)}
}

// oidcUserInfo is an abstraction of oidc.UserInfo which adds a level of indirection used in testing.
type oidcUserInfo interface {
	Claims(u interface{}) error
}

// oidcIDTokenVerifier is an abstraction of oidc.IDTokenVerifier which adds a level of indirection used in testing.
type oidcIDTokenVerifier interface {
	Verify(client context.Context, token string) (oidcIDToken, error)
}

// wrappedOIDCIDTokenVerifier simply delegates to the oidc.IDTokenVerifier it contains.
type wrappedOIDCIDTokenVerifier struct {
	verifier *oidc.IDTokenVerifier
}

func (w wrappedOIDCIDTokenVerifier) Verify(ctx context.Context, token string) (oidcIDToken, error) {
	idToken, err := w.verifier.Verify(ctx, token)
	if err != nil {
		return nil, err
	}
	return wrappedOIDCIDToken{token: idToken}, nil
}

// oidcIDToken is an abstraction of oidc.IDToken which adds a level of indirection used in testing.
type oidcIDToken interface {
	GetNonce() string
	Claims(v interface{}) error
	GetExpiry() time.Time
}

// wrappedOIDCIDToken simply delegates to the oidc.IDToken it contains.
type wrappedOIDCIDToken struct {
	token *oidc.IDToken
}

func (w wrappedOIDCIDToken) GetNonce() string {
	return w.token.Nonce
}

func (w wrappedOIDCIDToken) Claims(v interface{}) error {
	return w.token.Claims(v)
}

func (w wrappedOIDCIDToken) GetExpiry() time.Time {
	return w.token.Expiry
}

type exchangeFunc func(ctx context.Context, oauthCfg *oauth2.Config, code string) (oauth2Token, error)

// oauthExchange is a level of indirection over oauth2.Config.Exchange used in testing.
func oauthExchange(ctx context.Context, oauthCfg *oauth2.Config, code string) (oauth2Token, error) {
	token, err := oauthCfg.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}
	return wrappedOAuth2Token{token: token}, nil
}

// oauth2Token is an abstraction of oauth2.Token which adds a level of indirection used in testing.
type oauth2Token interface {
	GetAccessToken() string
	GetRefreshToken() string
	GetExtra(s string) interface{}
}

// wrappedOAuth2Token simply delegates to the oauth2.Token it contains.
type wrappedOAuth2Token struct {
	token *oauth2.Token
}

func (w wrappedOAuth2Token) GetAccessToken() string {
	return w.token.AccessToken
}

func (w wrappedOAuth2Token) GetRefreshToken() string {
	return w.token.RefreshToken
}

func (w wrappedOAuth2Token) GetExtra(s string) interface{} {
	return w.token.Extra(s)
}
