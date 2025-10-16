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
	tm := &storage.TokenMetadata{}
	tm.SetId(uuid.NewV4().String())
	tm.SetName("Generated Test Token")
	tm.SetRoles([]string{"Admin"})
	tm.SetIssuedAt(protocompat.ConvertTimeToTimestampOrNil(&truncatedNow))
	tm.SetExpiration(protocompat.ConvertTimeToTimestampOrNil(&truncatedExpiration))
	tm.SetRevoked(revoked)
	return tm
}
