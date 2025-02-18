package testutils

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
)

func GenerateToken(t *testing.T, now *time.Time, expiration *time.Time, revoked bool) *storage.TokenMetadata {
	var truncatedNow *time.Time
	var truncatedExpiration *time.Time
	if now != nil {
		rawTruncatedNow := now.Truncate(time.Microsecond)
		truncatedNow = &rawTruncatedNow
	}
	if expiration != nil {
		rawTruncatedExpiration := expiration.Truncate(time.Microsecond)
		truncatedExpiration = &rawTruncatedExpiration
	}
	return &storage.TokenMetadata{
		Id:         uuid.NewV4().String(),
		Name:       "Generated Test Token",
		Roles:      []string{"Admin"},
		IssuedAt:   protocompat.ConvertTimeToTimestampOrNil(truncatedNow),
		Expiration: protocompat.ConvertTimeToTimestampOrNil(truncatedExpiration),
		Revoked:    revoked,
	}
}
