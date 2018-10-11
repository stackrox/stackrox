package tokenbased

import (
	"fmt"

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

func (e *extractor) IdentityForRequest(ri requestinfo.RequestInfo) authn.Identity {
	rawToken := ExtractToken(ri.Metadata, "Bearer")
	if rawToken == "" {
		return nil
	}

	token, err := e.validator.Validate(rawToken)
	if err != nil {
		// TODO(mi): Make this an error once auth0 tokens are gone.
		return nil
	}

	if token.Permissions != nil {
		id := &roleBasedIdentity{
			uid:          fmt.Sprintf("auth-token:%s", token.ID),
			friendlyName: token.Subject,
			role:         permissions.NewRoleWithPermissions("unnamed", token.Permissions...),
		}
		if id.friendlyName == "" {
			id.friendlyName = fmt.Sprintf("anonymous bearer token (expires %v)", token.Expiry())
		}
		return id
	}

	if token.Role != "" {
		role := e.roleStore.RoleByName(token.Role)
		if role == nil {
			// TODO(mi): Make this an error once auth0 tokens are gone.
			return nil
		}
		id := &roleBasedIdentity{
			uid:          fmt.Sprintf("auth-token:%s", token.ID),
			friendlyName: token.Subject,
			role:         role,
		}
		if id.friendlyName == "" {
			id.friendlyName = fmt.Sprintf("anonymous bearer token with role %s (expires %v)", role.Name(), token.Expiry())
		}
		return id
	}

	return nil
}

// NewExtractor returns a new token-based identity extractor.
func NewExtractor(roleStore permissions.RoleStore, tokenValidator tokens.Validator) authn.IdentityExtractor {
	return &extractor{
		roleStore: roleStore,
		validator: tokenValidator,
	}
}
