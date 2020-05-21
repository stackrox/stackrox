package fake

import (
	"math/rand"

	"github.com/stackrox/rox/pkg/uuid"
	"k8s.io/apimachinery/pkg/types"
)

func newUUID() types.UID {
	return types.UID(uuid.NewV4().String())
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
