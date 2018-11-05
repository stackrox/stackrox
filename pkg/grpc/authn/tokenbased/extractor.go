package tokenbased

import (
	"errors"
	"fmt"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

type extractor struct {
	roleStore permissions.RoleStore
	validator tokens.Validator
}

func (e *extractor) IdentityForRequest(ri requestinfo.RequestInfo) (authn.Identity, error) {
	rawToken := ExtractToken(ri.Metadata, "Bearer")
	if rawToken == "" {
		return nil, nil
	}

	token, err := e.validator.Validate(rawToken)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %v", err)
	}

	// Anonymous permission-based tokens (true bearer tokens).
	if token.Permissions != nil {
		id := &roleBasedIdentity{
			uid:          fmt.Sprintf("auth-token:%s", token.ID),
			friendlyName: token.Subject,
			role:         permissions.NewRoleWithPermissions("unnamed", token.Permissions...),
		}
		if id.friendlyName == "" {
			id.friendlyName = fmt.Sprintf("anonymous bearer token (expires %v)", token.Expiry())
		}
		return id, nil
	}

	// Anonymous role-based tokens.
	if token.RoleName != "" {
		role := e.roleStore.RoleByName(token.RoleName)
		if role == nil {
			return nil, fmt.Errorf("token referenced invalid role %q", token.RoleName)
		}
		id := &roleBasedIdentity{
			uid:          fmt.Sprintf("auth-token:%s", token.ID),
			friendlyName: token.Subject,
			role:         role,
		}
		if id.friendlyName == "" {
			id.friendlyName = fmt.Sprintf("anonymous bearer token with role %s (expires %v)", role.GetName(), token.Expiry())
		}
		return id, nil
	}

	// External user token
	if token.ExternalUser != nil {
		if len(token.Sources) != 1 {
			return nil, errors.New("external user tokens must originate from exactly one source")
		}
		authProviderSrc, ok := token.Sources[0].(authproviders.AuthProvider)
		if !ok {
			return nil, errors.New("external user tokens must originate from an authentication provider source")
		}
		if !authProviderSrc.Enabled() {
			return nil, fmt.Errorf("auth provider %s is not enabled", authProviderSrc.Name())
		}
		roleMapper := authProviderSrc.RoleMapper()
		if roleMapper == nil {
			return nil, errors.New("misconfigured authentication provider: no role mapper defined")
		}
		role := roleMapper.Role(token.ExternalUser.UserID)
		if role == nil {
			return nil, fmt.Errorf("external user %s has no assigned role", token.ExternalUser.UserID)
		}
		id := &roleBasedIdentity{
			uid:          fmt.Sprintf("sso:%s:%s", token.Sources[0].ID(), token.ExternalUser.UserID),
			friendlyName: token.ExternalUser.FullName,
			role:         role,
		}
		if id.friendlyName == "" {
			if token.ExternalUser.Email != "" {
				id.friendlyName = token.ExternalUser.Email
			} else {
				id.friendlyName = token.ExternalUser.UserID
			}
		} else if token.ExternalUser.Email != "" {
			id.friendlyName += fmt.Sprintf(" (%s)", token.ExternalUser.Email)
		}
		if err := authProviderSrc.RecordSuccess(); err != nil {
			log.Errorf("Could not record success for authentication provider %s: %v", authProviderSrc.Name(), err)
		}
		return id, nil
	}

	return nil, errors.New("could not determine token type")
}

// NewExtractor returns a new token-based identity extractor.
func NewExtractor(roleStore permissions.RoleStore, tokenValidator tokens.Validator) authn.IdentityExtractor {
	return &extractor{
		roleStore: roleStore,
		validator: tokenValidator,
	}
}
