package service

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/mtls"
)

type identity struct {
	id mtls.Identity
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

func (i identity) Role() *storage.Role {
	return nil // services do not have roles
}

func (i identity) Expiry() time.Time {
	return i.id.Expiry
}

func (i identity) ExternalAuthProvider() authproviders.AuthProvider {
	return nil
}
