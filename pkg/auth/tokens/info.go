package tokens

import (
	"time"

	"github.com/stackrox/rox/pkg/timeutil"
)

// TokenInfo is a user-friendly representation of everything that a token represents.
type TokenInfo struct {
	// Token is the encoded and signed token
	Token string
	// Claims are the claims of the token
	*Claims
	// Sources are the sources that generated this token.
	Sources []Source
}

// Expiry returns the expiry time of the token.
func (i *TokenInfo) Expiry() time.Time {
	if i.Claims == nil || i.Claims.Expiry == nil {
		return timeutil.Max
	}
	return i.Claims.Expiry.Time()
}

// IssuedAt returns the time at which the token was issued.
func (i *TokenInfo) IssuedAt() time.Time {
	return i.Claims.IssuedAt.Time()
}
