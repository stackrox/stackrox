package idcheck

import (
	"github.com/stackrox/rox/pkg/grpc/authn"
)

// fakeIDAuthorizer is a simple IdentityBasedAuthorizer that returns a preset error.
type fakeIDAuthorizer struct {
	err error
}

func (f fakeIDAuthorizer) AuthorizeByIdentity(_ authn.Identity) error {
	return f.err
}
