package distroctx

import (
	"context"
)

// distroContextKey is the key for the distribution value in the context.
type distroContextKey struct{}

// distroContextValue holds the value of the distro in the context.
type distroContextValue struct {
	distro string
}

// Context returns a new context with the scope attached.
func Context(ctx context.Context, distro string) context.Context {
	return context.WithValue(ctx, distroContextKey{}, &distroContextValue{
		distro: distro,
	})
}

// IsImageScoped returns a boolean if a distro is set
func IsImageScoped(ctx context.Context) bool {
	return FromContext(ctx) != ""
}

// FromContext returns the distro from the input context
func FromContext(hasDistroContext context.Context) string {
	if hasDistroContext == nil {
		return ""
	}
	inter := hasDistroContext.Value(distroContextKey{})
	if inter == nil {
		return ""
	}
	s := inter.(*distroContextValue)
	return s.distro
}
