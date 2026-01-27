package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/durationpb"
)

var (
	// testExpirationDuration is the constant token expiration duration.
	testExpirationDuration = &durationpb.Duration{Seconds: 300}
	// testTokenExpiry is the timestamp of the token expiration.
	testTokenExpiry = testClock().Add(testExpirationDuration.AsDuration())
	// testExpiredTraits include the expired trait.
	testExpiredTraits, _ = generateTraitsWithExpiry(testTokenExpiry.Add(rbacObjectsGraceExpiration))
)

// testClock is the clock function injection for testing purposes.
func testClock() time.Time {
	return time.Date(1989, time.November, 9, 18, 05, 35, 987654321, time.UTC)
}

func Test_clock(t *testing.T) {
	now := testClock()
	assert.Equal(t, testExpirationDuration.Seconds, int64(testTokenExpiry.Sub(now).Seconds()))
	assert.Equal(t, rbacObjectsGraceExpiration, testExpiredTraits.ExpiresAt.AsTime().Sub(testTokenExpiry))
}
