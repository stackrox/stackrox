package fake

import (
	"math/rand"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"k8s.io/apimachinery/pkg/types"
)

func newUUID() types.UID {
	p := make([]byte, 16)
	n, err := rand.Read(p)
	utils.Must(err)
	if n != 16 {
		utils.CrashOnError(errors.New("wrong uuid"))
	}
	return types.UID(uuid.FromBytesOrNil(p).String())
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
