package fake

import (
	"fmt"
	"math/rand"

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
