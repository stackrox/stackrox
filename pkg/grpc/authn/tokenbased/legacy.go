package tokenbased

import (
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokenbased"
	"github.com/stackrox/rox/pkg/auth/tokenbased/user"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc/metadata"
)

var (
	logger = logging.LoggerForModule()
)

type legacyIdentity struct {
	provider authproviders.AuthProvider
	id       tokenbased.Identity
}

func (i *legacyIdentity) UID() string {
	return i.id.ID()
}

func (i *legacyIdentity) Role() permissions.Role {
	return i.id.Role()
}

func (i *legacyIdentity) Expiry() time.Time {
	return i.id.Expiration()
}

func (i *legacyIdentity) ExternalAuthProvider() authproviders.AuthProvider {
	return i.provider
}

func (i *legacyIdentity) Service() *v1.ServiceIdentity {
	return nil
}

func (i *legacyIdentity) FriendlyName() string {
	return i.id.ID()
}

// An legacyAuthProviderExtractor provides gRPC interceptors that authenticates users.
type legacyAuthProviderExtractor struct {
	authProviderAccessor authproviders.AuthProviderAccessor
	userRoleMapper       tokenbased.RoleMapper
}

// NewLegacyExtractor creates a new legacyAuthProviderExtractor.
func NewLegacyExtractor(authProviderAccessor authproviders.AuthProviderAccessor, userRoleMapper tokenbased.RoleMapper) authn.IdentityExtractor {
	return &legacyAuthProviderExtractor{
		authProviderAccessor: authProviderAccessor,
		userRoleMapper:       userRoleMapper,
	}
}

func (a *legacyAuthProviderExtractor) IdentityForRequest(requestInfo requestinfo.RequestInfo) authn.Identity {
	authProviders := a.authProviderAccessor.GetParsedAuthProviders()

	// Consult the auth providers and try to get a user identity.
	id, authProviderID := a.getUserIdentity(requestInfo.Metadata, authProviders)

	if id == nil {
		return nil
	}
	// If authProviderID is not in the map, it's a programming error since it's returned to us by the (private) function we call.
	authProvider := authProviders[authProviderID]
	if !authProvider.Validated() {
		// If we did find an identity, mark the auth provider that gave us the identity as validated.
		if err := a.authProviderAccessor.RecordAuthSuccess(authProviderID); err != nil {
			logger.Errorf("Failed to update auth provider status for auth %s with "+
				"loginURL %s: %s", authProviderID, authProvider.LoginURL(), err)
		}
	}

	return &legacyIdentity{
		provider: authProvider,
		id:       id,
	}
}

func (a *legacyAuthProviderExtractor) getUserIdentity(metadata metadata.MD, authProviders map[string]authproviders.AuthProvider) (u user.Identity, authProviderID string) {
	for id, authProvider := range authProviders {
		if !authProvider.Enabled() {
			continue
		}

		identity, err := authProvider.Parse(metadata, a.userRoleMapper)
		if err != nil {
			logger.Debugf("user auth error: %s", err)
			continue
		}

		return user.NewIdentity(identity, authProvider), id
	}
	return
}
