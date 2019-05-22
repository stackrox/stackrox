package tokenbased

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	log = logging.LoggerForModule()
)

// NewExtractor returns a new token-based identity extractor.
func NewExtractor(roleStore permissions.RoleStore, tokenValidator tokens.Validator) authn.IdentityExtractor {
	return &extractor{
		roleStore: roleStore,
		validator: tokenValidator,
	}
}

type extractor struct {
	roleStore permissions.RoleStore
	validator tokens.Validator
}

func (e *extractor) IdentityForRequest(ctx context.Context, ri requestinfo.RequestInfo) (authn.Identity, error) {
	rawToken := ExtractToken(ri.Metadata, "Bearer")
	if rawToken == "" {
		return nil, nil
	}

	token, err := e.validator.Validate(rawToken)
	if err != nil {
		return nil, errors.Wrap(err, "token validation failed")
	}

	// Anonymous permission-based tokens (true bearer tokens).
	if token.Permissions != nil {
		return e.withPermissions(token)
	}

	// We need all access for retrieving roles and upserting user info. Note that this context
	// is not propagated to the user, so the user itself does not get any escalated privileges.
	// Conversely, the context can't contain any access scope information because the identity has
	// not yet been extracted, so all code called with this context *must not* depend on a user
	// identity.
	ctx = sac.WithAllAccess(ctx)

	// Anonymous role-based tokens.
	if token.RoleName != "" {
		return e.withRoleName(ctx, token)
	}

	// External user token
	if token.ExternalUser != nil {
		return e.withExternalUser(ctx, token)
	}

	return nil, errors.New("could not determine token type")
}

func (e *extractor) withPermissions(token *tokens.TokenInfo) (authn.Identity, error) {
	id := &roleBasedIdentity{
		uid:          fmt.Sprintf("auth-token:%s", token.ID),
		username:     token.ExternalUser.Email,
		friendlyName: token.Subject,
		role:         permissions.NewRoleWithPermissions("unnamed", token.Permissions...),
		expiry:       token.Expiry(),
	}
	if id.friendlyName == "" {
		id.friendlyName = fmt.Sprintf("anonymous bearer token (expires %v)", token.Expiry())
	}
	return id, nil
}

func (e *extractor) withRoleName(ctx context.Context, token *tokens.TokenInfo) (authn.Identity, error) {
	role, err := e.roleStore.GetRole(ctx, token.RoleName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read role role %q", token.RoleName)
	}
	if role == nil {
		return nil, fmt.Errorf("token referenced invalid role %q", token.RoleName)
	}
	var email string
	if token.ExternalUser != nil {
		email = token.ExternalUser.Email
	}
	id := &roleBasedIdentity{
		uid:          fmt.Sprintf("auth-token:%s", token.ID),
		username:     email,
		friendlyName: token.Subject,
		role:         role,
		expiry:       token.Expiry(),
	}
	if id.friendlyName == "" {
		id.friendlyName = fmt.Sprintf("anonymous bearer token with role %s (expires %v)", role.GetName(), token.Expiry())
	}
	return id, nil
}

func (e *extractor) withExternalUser(ctx context.Context, token *tokens.TokenInfo) (authn.Identity, error) {
	if len(token.Sources) != 1 {
		return nil, errors.New("external user tokens must originate from exactly one source")
	}

	authProviderSrc, ok := token.Sources[0].(authproviders.Provider)
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

	role, err := roleMapper.FromTokenClaims(ctx, token.Claims)
	if err != nil {
		return nil, errors.Wrap(err, "unable to load role for user")
	}
	if role == nil {
		return nil, fmt.Errorf("external user %s has no assigned role", token.ExternalUser.UserID)
	}

	id := createRoleBasedIdentity(role, token)
	return id, nil
}

func createRoleBasedIdentity(role *storage.Role, token *tokens.TokenInfo) *roleBasedIdentity {
	id := &roleBasedIdentity{
		uid:          fmt.Sprintf("sso:%s:%s", token.Sources[0].ID(), token.ExternalUser.UserID),
		username:     token.ExternalUser.Email,
		friendlyName: token.ExternalUser.FullName,
		role:         role,
		expiry:       token.Expiry(),
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
	return id
}
