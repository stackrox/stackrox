package authproviders

import (
	"fmt"

	"github.com/stackrox/rox/generated/api/v1"
)

// Creator is the func stub that defines how to instantiate an auth provider integration.
type Creator func(authProvider *v1.AuthProvider) (AuthProvider, error)

// registry maps a particular auth provider to the func that can create it.
var registry = map[string]Creator{}

// Register registers an auth provider into the registry.
func Register(name string, creator Creator) {
	registry[name] = creator
}

// Create checks to make sure the integration exists and then tries to generate a new AuthProvider
// returns an error if the creation was unsuccessful.
func Create(authProvider *v1.AuthProvider) (AuthProvider, error) {
	creator, exists := registry[authProvider.GetType()]
	if !exists {
		return nil, fmt.Errorf("AuthProvider with type '%v' does not exist", authProvider.Type)
	}
	return creator(authProvider)
}

// LoginURLFromProto returns the LoginURL corresponding to this auth provider, returning
// an error if the auth provider can't be instantiated.
func LoginURLFromProto(protoAuthProvider *v1.AuthProvider) (string, error) {
	authProvider, err := Create(protoAuthProvider)
	if err != nil {
		return "", fmt.Errorf("auth provider creation: %s", err)
	}
	return authProvider.LoginURL(), nil
}
