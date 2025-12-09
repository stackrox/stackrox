package fake

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/stackrox/rox/pkg/uuid"
	"k8s.io/apimachinery/pkg/types"
)

func newUUID() types.UID {
	return types.UID(uuid.NewV4().String())
}

// fakeVMUUID generates a deterministic UUID-like string from an index.
// This ensures the same index always produces the same ID, and the ID
// is a valid UUID format that Central will accept.
// Format: 00000000-0000-4000-8000-{12-digit-index}
func fakeVMUUID(index int) string {
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", index)
}

const charset = "abcdef0123456789"

func randStringWithLength(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func randString() string {
	b := make([]byte, 48)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// jitteredInterval returns a duration with random jitter applied.
// The result is in the range [interval * (1 - jitterPercent), interval * (1 + jitterPercent)].
// For example, with interval=60s and jitterPercent=0.05, returns a value between 57s and 63s.
func jitteredInterval(interval time.Duration, jitterPercent float64) time.Duration {
	// Calculate jitter range: interval * jitterPercent
	jitterRange := float64(interval) * jitterPercent
	// Random value in [-jitterRange, +jitterRange]
	jitter := (rand.Float64()*2 - 1) * jitterRange
	// Return interval with jitter applied
	return time.Duration(float64(interval) + jitter)
}
