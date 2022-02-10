package authproviders

import (
	"context"
	"net/http"
	"time"

	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
)

// AuthResponse is the response by an auth provider backend that leads to a token issuance.
type AuthResponse struct {
	Claims     *tokens.ExternalUserClaim
	Expiration time.Time
	ExtraOpts  []tokens.Option

	RefreshTokenData
}

// Backend is a backend for an authentication provider.
type Backend interface {
	Config() map[string]string

	// LoginURL returns a login URL with the given client state.
	LoginURL(clientState string, ri *requestinfo.RequestInfo) (string, error)
	// RefreshURL returns a refresh URL, if supported by the auth provider.
	RefreshURL() string

	// OnEnable is called when a provider is enabled
	OnEnable(provider Provider)

	// OnDisable is called when a provider is disabled
	OnDisable(provider Provider)

	// ProcessHTTPRequest dispatches HTTP/1.1 requests intended for this provider. If the request is a callback from
	// a login page, and the login was successful, the respective AuthResponse is returned. If a non-login HTTP
	// call should be handled, a nil AuthResponse and error should be returned.
	ProcessHTTPRequest(w http.ResponseWriter, r *http.Request) (*AuthResponse, error)
	// ExchangeToken is called to exchange an external token, referring to the auth provider, against a Rox-issued
	// token.
	ExchangeToken(ctx context.Context, externalToken, state string) (*AuthResponse, string, error)

	// Validate allows an auth provider backend to mark a token as invalid and require reauthentication
	Validate(ctx context.Context, claims *tokens.Claims) error
}

// RefreshTokenEnabledBackend is an auth provider backend that supports refresh tokens.
type RefreshTokenEnabledBackend interface {
	Backend

	// RefreshAccessToken issues a new access token, using the given refresh token.
	RefreshAccessToken(ctx context.Context, refreshTokenData RefreshTokenData) (*AuthResponse, error)

	// RevokeRefreshToken revokes an issued refresh token.
	RevokeRefreshToken(ctx context.Context, refreshTokenData RefreshTokenData) error
}
