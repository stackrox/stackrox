package m2m

import (
	"context"
	"regexp"

	"github.com/pkg/errors"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth"
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
	config        *storage.AuthMachineToMachineConfig
	configRegexps []*regexp.Regexp
	roleDS        roleDataStore.DataStore
}

func newRoleMapper(config *storage.AuthMachineToMachineConfig, roleDS roleDataStore.DataStore,
	configRegExps []*regexp.Regexp) *roleMapper {
	return &roleMapper{
		config:        config,
		configRegexps: configRegExps,
		roleDS:        roleDS,
	}
}

func (r *roleMapper) FromUserDescriptor(ctx context.Context, user *permissions.UserDescriptor) ([]permissions.ResolvedRole, error) {
	return resolveRolesForClaims(ctx, user.Attributes, r.roleDS, r.config.GetMappings(), r.configRegexps)
}

func resolveRolesForClaims(ctx context.Context, claims map[string][]string, roleDS roleDataStore.DataStore,
	mappings []*storage.AuthMachineToMachineConfig_Mapping, expressions []*regexp.Regexp) ([]permissions.ResolvedRole, error) {
	rolesForUser := set.NewStringSet()

	for i, mapping := range mappings {
		if valuesMatch(expressions[i], claims[mapping.GetKey()]) {
			rolesForUser.Add(mapping.GetRole())
		}
	}

	// If no roles are assigned to the user, we will return an error and short-circuit.
	if rolesForUser.Cardinality() == 0 {
		return nil, auth.ErrNoValidRole
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

func valuesMatch(expr *regexp.Regexp, claimValues []string) bool {
	for _, claimValue := range claimValues {
		if expr.MatchString(claimValue) {
			return true
		}
	}
	return false
}
