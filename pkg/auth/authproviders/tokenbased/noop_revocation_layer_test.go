package tokenbased

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stretchr/testify/assert"
)

func TestNoopRevocationLayer(t *testing.T) {
	testTokenID1 := "abcd"
	testTokenID2 := "efgh"

	revocationLayer := &noopRevocationLayer{}

	assert.NoError(t, revocationLayer.Validate(t.Context(), nil))
	assert.NoError(t, revocationLayer.Validate(t.Context(), &tokens.Claims{}))

	assert.False(t, revocationLayer.IsRevoked(testTokenID1))
	revocationLayer.Revoke(testTokenID1, time.Now().Add(-1*time.Minute))
	assert.False(t, revocationLayer.IsRevoked(testTokenID1))

	assert.False(t, revocationLayer.IsRevoked(testTokenID2))
	revocationLayer.Revoke(testTokenID2, time.Now().Add(-1*time.Hour))
	assert.False(t, revocationLayer.IsRevoked(testTokenID2))
}
