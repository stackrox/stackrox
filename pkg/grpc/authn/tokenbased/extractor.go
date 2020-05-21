package tokenbased

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/utils"
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

	token, err := e.validator.Validate(ctx, rawToken)
	if err != nil {
		return nil, errors.Wrap(err, "token validation failed")
	}

	// All tokens should have a source.
	if len(token.Sources) != 1 {
		return nil, errors.New("tokens must originate from exactly one source")
	}
	authProviderSrc, ok := token.Sources[0].(authproviders.Provider)
	if !ok {
		return nil, errors.New("API tokens must originate from an authentication provider source")
	}
	if !authProviderSrc.Enabled() {
		return nil, fmt.Errorf("auth provider %s is not enabled", authProviderSrc.Name())
	}

	// Anonymous permission-based tokens (true bearer tokens).
	if token.Permissions != nil {
		return e.withPermissions(token, authProviderSrc)
	}

	// We need all access for retrieving roles and upserting user info. Note that this context
	// is not propagated to the user, so the user itself does not get any escalated privileges.
	// Conversely, the context can't contain any access scope information because the identity has
	// not yet been extracted, so all code called with this context *must not* depend on a user
	// identity.
	ctx = sac.WithAllAccess(ctx)

	roleNames := token.RoleNames
	if token.RoleName != "" {
		if len(roleNames) != 0 {
			return nil, errors.New("malformed token: uses both 'roles' and deprecated 'role' claims")
		}
		roleNames = []string{token.RoleName}
	}

	// Anonymous role-based tokens.
	if len(roleNames) > 0 {
		return e.withRoleNames(ctx, token, roleNames, authProviderSrc)
	}

	// External user token
	if token.ExternalUser != nil {
		return e.withExternalUser(ctx, token, authProviderSrc)
	}

	return nil, errors.New("could not determine token type")
}

func (e *extractor) withPermissions(token *tokens.TokenInfo, authProvider authproviders.Provider) (authn.Identity, error) {
	attributes := map[string][]string{"name": {token.Name}}

	pseudoRole := permissions.NewRoleWithPermissions("", token.Permissions...)
	id := &roleBasedIdentity{
		uid:          fmt.Sprintf("auth-token:%s", token.ID),
		username:     token.ExternalUser.Email,
		friendlyName: token.Subject,
		fullName:     token.ExternalUser.FullName,
		perms:        pseudoRole,
		roles:        []*storage.Role{pseudoRole},
		expiry:       token.Expiry(),
		authProvider: authProvider,
		attributes:   attributes,
	}
	if id.friendlyName == "" {
		id.friendlyName = fmt.Sprintf("anonymous bearer token (expires %v)", token.Expiry())
	}
	return id, nil
}

func (e *extractor) withRoleNames(ctx context.Context, token *tokens.TokenInfo, roleNames []string, authProvider authproviders.Provider) (authn.Identity, error) {
	roles, _, err := permissions.GetRolesFromStore(ctx, e.roleStore, roleNames)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read roles")
	}
	if len(roles) == 0 {
		return nil, utils.Should(errors.New("none of the roles referenced by the token were found"))
	}

	var email string
	if token.ExternalUser != nil {
		email = token.ExternalUser.Email
	}

	attributes := map[string][]string{"role": permissions.RoleNames(roles), "name": {token.Name}}
	id := &roleBasedIdentity{
		uid:          fmt.Sprintf("auth-token:%s", token.ID),
		username:     email,
		friendlyName: token.Subject,
		fullName:     token.Name,
		perms:        permissions.NewUnionRole(roles),
		roles:        roles,
		expiry:       token.Expiry(),
		attributes:   attributes,
		authProvider: authProvider,
	}
	if id.friendlyName == "" {
		id.friendlyName = fmt.Sprintf("anonymous bearer token with roles %s (expires %v)", strings.Join(roleNames, ","), token.Expiry())
	}
	return id, nil
}

func (e *extractor) withExternalUser(ctx context.Context, token *tokens.TokenInfo, authProvider authproviders.Provider) (authn.Identity, error) {
	if len(token.Sources) != 1 {
		return nil, errors.New("external user tokens must originate from exactly one source")
	}

	roleMapper := authProvider.RoleMapper()
	if roleMapper == nil {
		return nil, errors.New("misconfigured authentication provider: no role mapper defined")
	}

	ud := &permissions.UserDescriptor{
		UserID:     token.Claims.ExternalUser.UserID,
		Attributes: token.Claims.ExternalUser.Attributes,
	}

	roles, err := roleMapper.FromUserDescriptor(ctx, ud)
	if err != nil {
		return nil, errors.Wrap(err, "unable to load role for user")
	}
	if len(roles) == 0 {
		return nil, fmt.Errorf("external user %s has no assigned role", token.ExternalUser.UserID)
	}
	if err := authProvider.MarkAsActive(); err != nil {
		return nil, errors.Wrapf(err, "unable to mark provider %q as validated", authProvider.Name())
	}
	id := createRoleBasedIdentity(roles, token, authProvider)
	return id, nil
}

func createRoleBasedIdentity(roles []*storage.Role, token *tokens.TokenInfo, authProvider authproviders.Provider) *roleBasedIdentity {
	id := &roleBasedIdentity{
		uid:          fmt.Sprintf("sso:%s:%s", token.Sources[0].ID(), token.ExternalUser.UserID),
		username:     token.ExternalUser.Email,
		friendlyName: token.ExternalUser.FullName,
		fullName:     token.ExternalUser.FullName,
		perms:        permissions.NewUnionRole(roles),
		roles:        roles,
		expiry:       token.Expiry(),
		attributes:   token.Claims.ExternalUser.Attributes,
		authProvider: authProvider,
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
