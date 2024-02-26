package tokensource

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

var futureTime = time.Date(3000, time.January, 0, 0, 0, 0, 0, time.Local)

type fakeTokenSource struct {
	called int
}

func (f *fakeTokenSource) Token() (*oauth2.Token, error) {
	f.called++
	return &oauth2.Token{AccessToken: strconv.Itoa(f.called), Expiry: futureTime}, nil
}

func TestReuseTokenSourceWithForceRefresh(t *testing.T) {
	t.Parallel()
	earlyExpiry := time.Minute
	ts := NewReuseTokenSourceWithInvalidate(&fakeTokenSource{}, earlyExpiry)

	token, err := ts.Token()
	assert.Equal(t, token.AccessToken, "1")
	assert.Equal(t, token.Expiry, futureTime.Add(-earlyExpiry))
	require.NoError(t, err)
	token, err = ts.Token()
	assert.Equal(t, token.AccessToken, "1")
	assert.Equal(t, token.Expiry, futureTime.Add(-earlyExpiry))
	require.NoError(t, err)

	ts.Invalidate()

	token, err = ts.Token()
	assert.Equal(t, token.AccessToken, "2")
	assert.Equal(t, token.Expiry, futureTime.Add(-earlyExpiry))
	require.NoError(t, err)
}
