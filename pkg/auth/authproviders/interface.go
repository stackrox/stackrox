package authproviders

import "bitbucket.org/stack-rox/apollo/pkg/auth/tokenbased"

// An AuthProvider is an authenticator which is based on an external service, like auth0.
// In addition to being a tokenbased IdentityParser, it also needs to return a login URL and a refresh URL.
type AuthProvider interface {
	tokenbased.IdentityParser
	// Enabled returns whether this authenticator is enabled.
	Enabled() bool
	// Validated returns whether this auth provider has been validated.
	Validated() bool
	// LoginURL returns the URL where the user should be redirected to, to log in.
	LoginURL() string
	// RefreshURL generates the URL that the browser should refresh in the background to extend the user's access.
	RefreshURL() string
}
