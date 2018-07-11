package authproviders

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// Creator is the func stub that defines how to instantiate an auth provider integration.
type Creator func(authProvider *v1.AuthProvider) (Authenticator, error)

// Registry maps a particular auth provider to the func that can create it.
var Registry = map[string]Creator{}

// Create checks to make sure the integration exists and then tries to generate a new AuthProvider
// returns an error if the creation was unsuccessful.
func Create(authProvider *v1.AuthProvider) (Authenticator, error) {
	creator, exists := Registry[authProvider.GetType()]
	if !exists {
		return nil, fmt.Errorf("AuthProvider with type '%v' does not exist", authProvider.Type)
	}
	return creator(authProvider)
}

// LoginURLFromProto returns the LoginURL corresponding to this auth provider, returning
// an error if the auth provider can't be instantiated.
func LoginURLFromProto(authProvider *v1.AuthProvider) (string, error) {
	authenticator, err := Create(authProvider)
	if err != nil {
		return "", fmt.Errorf("auth provider creation: %s", err)
	}
	return authenticator.LoginURL(), nil
}
