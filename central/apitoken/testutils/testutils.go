package testutils

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
)

// GenerateToken creates a token for testing.
func GenerateToken(t *testing.T, now time.Time, expiration time.Time, revoked bool) *storage.TokenMetadata {
	truncatedNow := now.Truncate(time.Microsecond)
	truncatedExpiration := expiration.Truncate(time.Microsecond)
	return &storage.TokenMetadata{
		Id:         uuid.NewV4().String(),
		Name:       "Generated Test Token",
		Roles:      []string{"Admin"},
		IssuedAt:   protocompat.ConvertTimeToTimestampOrNil(&truncatedNow),
		Expiration: protocompat.ConvertTimeToTimestampOrNil(&truncatedExpiration),
		Revoked:    revoked,
	}
}
