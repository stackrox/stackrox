package service

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/mtls"
)

var _ authn.Identity = (*identity)(nil)

type identity struct {
	id mtls.Identity
}

func (i identity) TenantID() string {
	//TODO implement me
	panic("implement me")
}

func (i identity) Service() *storage.ServiceIdentity {
	return i.id.V1()
}

func (i identity) UID() string {
	return fmt.Sprintf("mtls:%s@%v", i.id.Subject.Identifier, i.id.Serial)
}

func (i identity) FriendlyName() string {
	return i.id.Subject.CN()
}

func (i identity) FullName() string {
	return i.id.Subject.CN()
}

func (i identity) Permissions() map[string]storage.Access {
	return nil
}

func (i identity) Roles() []permissions.ResolvedRole {
	return nil // services do not have roles
}

func (i identity) User() *storage.UserInfo {
	return nil // services is not a user
}

func (i identity) ValidityPeriod() (time.Time, time.Time) {
	return i.id.NotBefore, i.id.Expiry
}

func (i identity) ExternalAuthProvider() authproviders.Provider {
	return nil
}

func (i identity) Attributes() map[string][]string {
	return nil
}

// WrapMTLSIdentity wraps an mTLS identity.
func WrapMTLSIdentity(id mtls.Identity) authn.Identity {
	return identity{
		id: id,
	}
}
