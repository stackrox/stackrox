package authproviders

import (
	"context"
	"net/http"

	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
)

// Backend is a backend for an authentication provider.
type Backend interface {
	// LoginURL returns a login URL with the given client state.
	LoginURL(clientState string, ri *requestinfo.RequestInfo) string
	// RefreshURL returns a refresh URL, if supported by the auth provider.
	RefreshURL() string

	// ProcessHTTPRequest dispatches HTTP/1.1 requests intended for this provider. If the request is a callback from
	// a login page, and the login was successful, the respective ExternalUserClaim is returned. If a non-login HTTP
	// call should be handled, a nil claim and error should be returned.
	ProcessHTTPRequest(w http.ResponseWriter, r *http.Request) (*tokens.ExternalUserClaim, []tokens.Option, string, error)
	// ExchangeToken is called to exchange an external token, referring to the auth provider, against a Rox-issued
	// token.
	ExchangeToken(ctx context.Context, externalToken, state string) (*tokens.ExternalUserClaim, []tokens.Option, string, error)
}
