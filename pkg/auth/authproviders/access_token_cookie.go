package authproviders

import (
	"net/http"
	"time"

	"github.com/stackrox/rox/pkg/auth/tokens"
)

const (
	// AccessTokenCookieName is the name of the cookie set by the backend which holds the access token.
	AccessTokenCookieName = "RoxAccessToken"
)

// AccessTokenCookie returns the cookie to set for the access token.
func AccessTokenCookie(token *tokens.TokenInfo) *http.Cookie {
	return &http.Cookie{
		Name:  AccessTokenCookieName,
		Value: token.Token,
		// The access token should be set for the complete page.
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(time.Until(token.Expiry()).Seconds()),
	}
}

// clearAccessTokenCookie ensures the access token cookie is unset.
// This is done by setting MaxAge to < 0.
func clearAccessTokenCookie() *http.Cookie {
	return &http.Cookie{
		Name:     AccessTokenCookieName,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	}
}

// IsAccessTokenCookie returns whether the given cookie is the AccessTokenCookie.
func IsAccessTokenCookie(cookie *http.Cookie) bool {
	return cookie.Name == AccessTokenCookieName &&
		cookie.Secure &&
		cookie.SameSite == http.SameSiteStrictMode &&
		cookie.HttpOnly
}
