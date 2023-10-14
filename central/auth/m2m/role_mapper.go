package m2m

import (
	"context"
	"regexp"

	"github.com/pkg/errors"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
)

var (
	_ permissions.RoleMapper = (*roleMapper)(nil)

	log = logging.LoggerForModule()
)

type roleMapper struct {
	config *storage.AuthMachineToMachineConfig
	roleDS roleDataStore.DataStore
}

func newRoleMapper(config *storage.AuthMachineToMachineConfig, roleDS roleDataStore.DataStore) permissions.RoleMapper {
	return &roleMapper{
		config: config,
		roleDS: roleDS,
	}
}

func (r *roleMapper) FromUserDescriptor(ctx context.Context, user *permissions.UserDescriptor) ([]permissions.ResolvedRole, error) {
	return resolveRolesForClaims(ctx, user.Attributes, r.roleDS, r.config.GetMappings())
}

func resolveRolesForClaims(ctx context.Context, claims map[string][]string, roleDS roleDataStore.DataStore,
	mappings []*storage.AuthMachineToMachineConfig_Mapping) ([]permissions.ResolvedRole, error) {
	rolesForUser := set.NewStringSet()

	for _, mapping := range mappings {
		if valuesMatch(mapping.GetValue(), claims[mapping.GetKey()]) {
			rolesForUser.Add(mapping.GetRole())
		}
	}

	// If no roles are assigned to the user, we will return an error and short-circuit.
	if rolesForUser.Cardinality() == 0 {
		return nil, errors.New("no roles assigned")
	}

	resolvedRoles := make([]permissions.ResolvedRole, 0, rolesForUser.Cardinality())
	for role := range rolesForUser {
		resolvedRole, err := roleDS.GetAndResolveRole(ctx, role)
		// Short-circuit if _any_ role cannot be resolved that _should_ be assigned to the user.
		// This theoretically shouldn't happen.
		if err != nil {
			return nil, errors.Wrapf(err, "resolving role %q", role)
		}

		// Explicitly skip the none role, since this shouldn't be assigned.
		if resolvedRole.GetRoleName() != authn.NoneRole {
			resolvedRoles = append(resolvedRoles, resolvedRole)
		}
	}

	return resolvedRoles, nil
}

func valuesMatch(expr string, claimValues []string) bool {
	for _, claimValue := range claimValues {
		if valueMatches(expr, claimValue) {
			return true
		}
	}
	return false
}

func valueMatches(expr string, claimValue string) bool {
	// We allow either a simple string as the expression, or regular expressions.
	if regExp := convertToRegexp(expr); regExp != nil {
		return regExp.MatchString(claimValue)
	}
	// If it's not a regular expression, we do simple string comparison.
	return claimValue == expr
}

func convertToRegexp(expr string) *regexp.Regexp {
	// If expression is the default value (i.e. empty string), the compiled regular expression will match
	// everything.
	if expr == "" {
		return nil
	}

	parsedExpr, err := regexp.Compile(expr)
	// Since we allow the user to either specify a regular expression or not, this could fail in the case
	// the value is a non-regexp. Hence, we do not return an error but instead log a debug message.
	if err != nil {
		log.Debugf("Failed to compile regular expression %q: %v", expr, err)
		return nil
	}
	return parsedExpr
}
