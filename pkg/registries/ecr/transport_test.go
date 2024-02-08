package ecr

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestECRTransport(t *testing.T) {
	transport := awsTransport{}
	// Token is invalid initially.
	assert.False(t, transport.isValidNoLock())

	// Token is valid after the first token exchange.
	expiredAt := time.Now().Add(12 * time.Hour)
	transport.expiresAt = &expiredAt
	assert.True(t, transport.isValidNoLock())

	// Token is invalid after it expired.
	expiredAt = time.Now()
	transport.expiresAt = &expiredAt
	assert.False(t, transport.isValidNoLock())
}
