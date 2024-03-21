package tokens

import (
	"time"

	"github.com/go-jose/go-jose/v3/jwt"
)

// Option is an option that can transparently modify a token's claims.
type Option interface {
	apply(*Claims)
}

type optFunc func(*Claims)

func (f optFunc) apply(claims *Claims) {
	f(claims)
}

// WithExpiry sets the given concrete expiry for a token. If an expiry is already set, it will be updated only if the
// existing expiry is later than the specified one (i.e., the validity of a token will never be extended).
func WithExpiry(expiry time.Time) Option {
	return optFunc(func(claims *Claims) {
		if claims.Expiry == nil || expiry.Before(claims.Expiry.Time()) {
			claims.Expiry = jwt.NewNumericDate(expiry)
		}
	})
}

// WithTTL sets the given expiry for a token for `ttl` after the time this function is applied to claims. The same rules
// wrt. updating of existing expiry times as for the above function apply.
func WithTTL(ttl time.Duration) Option {
	return optFunc(func(claims *Claims) {
		expiry := time.Now().Add(ttl)
		if claims.Expiry == nil || expiry.Before(claims.Expiry.Time()) {
			claims.Expiry = jwt.NewNumericDate(expiry)
		}
	})
}

// WithDefaultTTL sets the given TTL for a token ONLY if it does not have a TTL set.
func WithDefaultTTL(ttl time.Duration) Option {
	return optFunc(func(claims *Claims) {
		if claims.Expiry == nil {
			claims.Expiry = jwt.NewNumericDate(time.Now().Add(ttl))
		}
	})
}
