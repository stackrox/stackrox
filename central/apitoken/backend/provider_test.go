package backend

import (
	"testing"

	"github.com/stackrox/rox/pkg/auth/authproviders"
)

// The API Token source is cast as an auth provider so this ensures
// that is implements the interface
func TestEnsureImplementsAuthProvider(t *testing.T) {
	src := new(sourceImpl)
	_ = authproviders.Provider(src)
}
