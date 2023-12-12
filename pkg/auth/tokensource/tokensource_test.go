package tokensource

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

type fakeTokenSource struct {
	called int
}

func (f *fakeTokenSource) Token() (*oauth2.Token, error) {
	f.called++
	return &oauth2.Token{AccessToken: strconv.Itoa(f.called), Expiry: time.Date(3000, time.January, 0, 0, 0, 0, 0, time.Local)}, nil
}

func TestReuseTokenSourceWithForceRefresh(t *testing.T) {
	t.Parallel()
	ts := NewReuseTokenSourceWithForceRefresh(&fakeTokenSource{})

	token, err := ts.Token()
	assert.Equal(t, token.AccessToken, "1")
	require.NoError(t, err)
	token, err = ts.Token()
	assert.Equal(t, token.AccessToken, "1")
	require.NoError(t, err)

	ts.Expire()

	token, err = ts.Token()
	assert.Equal(t, token.AccessToken, "2")
	require.NoError(t, err)
}
