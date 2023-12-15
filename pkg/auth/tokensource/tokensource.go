package tokensource

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/oauth2"
)

// ReuseTokenSourceWithInvalidate works like oauth2.ReuseTokenSource but with
// an additional manual invalidate method that forces a token refresh.
type ReuseTokenSourceWithInvalidate struct {
	token     *oauth2.Token
	base      oauth2.TokenSource
	mutex     sync.Mutex
	isExpired bool
}

var _ oauth2.TokenSource = &ReuseTokenSourceWithInvalidate{}

// NewReuseTokenSourceWithInvalidate wraps a base token source and provides refresh and expiry functionality.
func NewReuseTokenSourceWithInvalidate(base oauth2.TokenSource) *ReuseTokenSourceWithInvalidate {
	return &ReuseTokenSourceWithInvalidate{base: base}
}

// Token returns an oauth token.
func (t *ReuseTokenSourceWithInvalidate) Token() (*oauth2.Token, error) {
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

// Invalidate forces the invalidation the cached token.
func (t *ReuseTokenSourceWithInvalidate) Invalidate() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.isExpired = true
}
