package auth

import (
	"context"
	"errors"
	"math/big"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

var (
	// ErrNoContext is returned when we process a context, but can't find any Identity info.
	ErrNoContext = errors.New("no identity context found")
)

type contextKey struct{}

// An Identity holds the information this package is able to ascertain from
// the credentials provided by the client.
type Identity struct {
	User         string
	Identifier   string
	IdentityType IdentityType
	Serial       *big.Int
}

// IdentityType describes the type of the identity.
// Either EndUser will be true or ServiceType will be a nonzero value, but not both.
// If all members are zero, no assertion is made about the identity.
type IdentityType struct {
	ServiceType v1.ServiceType
	EndUser     bool
}

func newContext(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

// FromContext retrieves identity information from the given context.
// The context must have been passed through the interceptors provided by this package.
func FromContext(ctx context.Context) (Identity, error) {
	val, ok := ctx.Value(contextKey{}).(Identity)
	if !ok {
		return Identity{}, ErrNoContext
	}
	return val, nil
}
