package tokenbased

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	permissionsUtils "github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
)

const (
	cacheSize          = 500
	rateLimitFrequency = 5 * time.Minute
	logBurstSize       = 5
)

var (
	log = logging.NewRateLimitLogger(logging.LoggerForModule(), cacheSize, 1, rateLimitFrequency, logBurstSize)
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
	rawToken := authn.ExtractToken(ri.Metadata, "Bearer")
	if rawToken == "" {
		return nil, nil
	}
	token, err := e.validator.Validate(ctx, rawToken)
	if err != nil {
		log.WarnL(ri.Hostname, "Token validation failed for hostname %v: %v", ri.Hostname, err)
		return nil, errors.New("token validation failed")
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

func (e *extractor) withRoleNames(ctx context.Context, token *tokens.TokenInfo, roleNames []string, authProvider authproviders.Provider) (authn.Identity, error) {
	resolvedRoles, _, err := permissions.GetResolvedRolesFromStore(ctx, e.roleStore, roleNames)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read roles")
	}
	// Ensure there are no invalid roles listed in the token.
	filteredRoles := authn.FilterOutNoneRole(resolvedRoles)
	var email string
	if token.ExternalUser != nil {
		email = token.ExternalUser.Email
	}

	attributes := map[string][]string{"role": permissionsUtils.RoleNames(filteredRoles), "name": {token.Name}}
	id := &roleBasedIdentity{
		uid:           fmt.Sprintf("auth-token:%s", token.ID),
		username:      email,
		friendlyName:  token.Subject,
		fullName:      token.Name,
		resolvedRoles: filteredRoles,
		expiry:        token.Expiry(),
		attributes:    attributes,
		authProvider:  authProvider,
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

	// We expect `FromUserDescriptor()` to filter out invalid roles.
	resolvedRoles, err := roleMapper.FromUserDescriptor(ctx, ud)
	if err != nil {
		return nil, errors.Wrap(err, "unable to load role for user")
	}
	if err := authProvider.MarkAsActive(); err != nil {
		return nil, errors.Wrapf(err, "unable to mark provider %q as validated", authProvider.Name())
	}
	id := createRoleBasedIdentity(resolvedRoles, token, authProvider)
	return id, nil
}

func createRoleBasedIdentity(roles []permissions.ResolvedRole, token *tokens.TokenInfo, authProvider authproviders.Provider) *roleBasedIdentity {
	id := &roleBasedIdentity{
		uid:           fmt.Sprintf("sso:%s:%s", token.Sources[0].ID(), token.ExternalUser.UserID),
		username:      token.ExternalUser.Email,
		friendlyName:  token.ExternalUser.FullName,
		fullName:      token.ExternalUser.FullName,
		resolvedRoles: roles,
		expiry:        token.Expiry(),
		attributes:    token.Claims.ExternalUser.Attributes,
		authProvider:  authProvider,
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
