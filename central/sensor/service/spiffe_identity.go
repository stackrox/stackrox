package service

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authn"
)

var _ authn.Identity = (*spiffeIdentity)(nil)

type spiffeIdentity struct {
	serviceType storage.ServiceType
	spiffeID    string
}

func (s *spiffeIdentity) Service() *storage.ServiceIdentity {
	return &storage.ServiceIdentity{
		Type: s.serviceType,
		Id:   s.spiffeID,
	}
}

func (s *spiffeIdentity) UID() string {
	return fmt.Sprintf("spiffe:%s", s.spiffeID)
}

func (s *spiffeIdentity) FriendlyName() string {
	return fmt.Sprintf("SPIFFE:%s", s.serviceType.String())
}

func (s *spiffeIdentity) FullName() string {
	return s.spiffeID
}

func (s *spiffeIdentity) Permissions() map[string]storage.Access {
	return nil
}

func (s *spiffeIdentity) Roles() []permissions.ResolvedRole {
	return nil // services do not have roles
}

func (s *spiffeIdentity) User() *storage.UserInfo {
	return nil // services are not users
}

func (s *spiffeIdentity) ValidityPeriod() (time.Time, time.Time) {
	// SPIRE SVIDs are short-lived, but we don't have the cert here to check
	// Return zero times to indicate unknown/not applicable
	return time.Time{}, time.Time{}
}

func (s *spiffeIdentity) ExternalAuthProvider() authproviders.Provider {
	return nil
}

func (s *spiffeIdentity) Attributes() map[string][]string {
	return map[string][]string{
		"spiffe.id": {s.spiffeID},
	}
}
