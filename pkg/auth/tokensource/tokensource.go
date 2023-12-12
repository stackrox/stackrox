package tokensource

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/oauth2"
)

// ReuseTokenSourceWithForceRefresh works like oauth2.ReuseTokenSource but with
// and additional manual expiry method that forces a token refresh.
type ReuseTokenSourceWithForceRefresh struct {
	token     *oauth2.Token
	base      oauth2.TokenSource
	mutex     sync.Mutex
	isExpired bool
}

var _ oauth2.TokenSource = &ReuseTokenSourceWithForceRefresh{}

// NewReuseTokenSourceWithForceRefresh wraps a base token source and provides refresh and expiry functionality.
func NewReuseTokenSourceWithForceRefresh(base oauth2.TokenSource) *ReuseTokenSourceWithForceRefresh {
	return &ReuseTokenSourceWithForceRefresh{base: base}
}

// Token returns an oauth token.
func (t *ReuseTokenSourceWithForceRefresh) Token() (*oauth2.Token, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if !t.isExpired && t.token.Valid() {
		return t.token, nil
	}
	token, err := t.base.Token()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get token")
	}
	t.token = token
	t.isExpired = false
	return t.token, nil
}

// Expire forces the invalidation the cached token.
func (t *ReuseTokenSourceWithForceRefresh) Expire() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.isExpired = true
}
