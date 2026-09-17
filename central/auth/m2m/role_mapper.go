package m2m

import (
	"context"
	"regexp"
	"slices"

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

type parsedMapping struct {
	Key        string
	Expression *regexp.Regexp
	Role       string
}

type roleMapper struct {
	mappings []parsedMapping
	roleDS   roleDataStore.DataStore
}

func newRoleMapperFromMappings(
	mappings []*storage.AuthMachineToMachineConfig_Mapping,
	roleDS roleDataStore.DataStore) *roleMapper {
	roleMappings := parseMappings(mappings)
	return &roleMapper{
		mappings: roleMappings,
		roleDS:   roleDS,
	}
}

func parseMappings(mappings []*storage.AuthMachineToMachineConfig_Mapping) []parsedMapping {
	result := make([]parsedMapping, 0, len(mappings))
	for _, m := range mappings {
		result = append(result, parsedMapping{
			Key:        m.GetKey(),
			Expression: regexp.MustCompile(m.GetValueExpression()),
			Role:       m.GetRole(),
		})
	}
	return result
}

func (r *roleMapper) FromUserDescriptor(ctx context.Context, user *permissions.UserDescriptor) ([]permissions.ResolvedRole, error) {
	return resolveRolesForClaims(ctx, user.Attributes, r.roleDS, r.mappings)
}

func resolveRolesForClaims(
	ctx context.Context,
	claims map[string][]string,
	roleDS roleDataStore.DataStore,
	mappings []parsedMapping,
) ([]permissions.ResolvedRole, error) {
	rolesForUser := set.NewStringSet()

	for _, mapping := range mappings {
		if anyValueMatches(mapping.Expression, claims[mapping.Key]) {
			rolesForUser.Add(mapping.Role)
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

func anyValueMatches(expr *regexp.Regexp, claimValues []string) bool {
	return slices.ContainsFunc(claimValues, expr.MatchString)
}
