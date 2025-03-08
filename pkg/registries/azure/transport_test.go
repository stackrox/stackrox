package azure

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAzureTransport(t *testing.T) {
	transport := azureTransport{}
	// Token is invalid initially.
	assert.False(t, transport.isValidNoLock())

	// Token is valid after the first token exchange.
	expiredAt := time.Now().Add(12 * time.Hour)
	transport.expiresAt = &expiredAt
	assert.True(t, transport.isValidNoLock())

	// Token is valid if more than the expiry delta is left.
	expiredAt = time.Now().Add(360 * time.Second)
	transport.expiresAt = &expiredAt
	assert.True(t, transport.isValidNoLock())

	// Token is invalid if less than the expiry delta is left.
	expiredAt = time.Now().Add(180 * time.Second)
	transport.expiresAt = &expiredAt
	assert.False(t, transport.isValidNoLock())

	// Token is invalid after it expired.
	expiredAt = time.Now()
	transport.expiresAt = &expiredAt
	assert.False(t, transport.isValidNoLock())
}
