package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bitbucket.org/stack-rox/apollo/pkg/authproviders"
	"bitbucket.org/stack-rox/apollo/pkg/mtls"
)

var (
	// ErrNoContext is returned when we process a context, but can't find any Identity info.
	ErrNoContext = errors.New("no identity context found")
)

type contextKey struct{}

// An Identity holds the information this package is able to ascertain from
// the credentials provided by the client.
type Identity struct {
	User       User
	TLS        mtls.Identity
	Expiration time.Time
}

// User has user data and which provider gave it to us.
type User struct {
	authproviders.User
	AuthProvider authproviders.Authenticator
}

func (id Identity) String() string {
	if id.User.ID != "" {
		return fmt.Sprintf("User: %s", id.User.ID)
	}
	return id.TLS.Name.String()
}

// NewContext adds the given Identity to the Context.
func NewContext(ctx context.Context, id Identity) context.Context {
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
