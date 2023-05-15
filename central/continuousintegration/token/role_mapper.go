package token

import (
	"context"
	"regexp"

	"github.com/pkg/errors"
	continuousIntegrationDataStore "github.com/stackrox/rox/central/continuousintegration/datastore"
	rolesDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
)

var (
	_ permissions.RoleMapper = (*roleMapper)(nil)
)

type roleMapper struct {
	continuousIntegrationDataStore continuousIntegrationDataStore.DataStore
	integrationType                storage.ContinuousIntegrationType
	rolesDataStore                 rolesDataStore.DataStore
}

// newRoleMapper creates a new role mapper for the continuous integration types.
func newRoleMapper(integrationType storage.ContinuousIntegrationType) permissions.RoleMapper {
	return &roleMapper{
		continuousIntegrationDataStore: continuousIntegrationDataStore.Singleton(),
		rolesDataStore:                 rolesDataStore.Singleton(),
		integrationType:                integrationType,
	}
}

func (r *roleMapper) FromUserDescriptor(ctx context.Context, user *permissions.UserDescriptor) ([]permissions.ResolvedRole, error) {
	configs, err := r.getConfigsForType(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving continuous integration configs")
	}
	return r.getRolesForUser(ctx, configs, user)
}

func (r *roleMapper) getConfigsForType(ctx context.Context) ([]*storage.ContinuousIntegrationConfig, error) {
	configCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration)))
	configs, err := r.continuousIntegrationDataStore.GetAllContinuousIntegrationConfigs(configCtx)
	if err != nil {
		return nil, err
	}

	configsForType := make([]*storage.ContinuousIntegrationConfig, 0, len(configs))

	for _, config := range configs {
		if config.GetType() == r.integrationType {
			configsForType = append(configsForType, config)
		}
	}
	return configsForType, nil
}

func (r *roleMapper) getRolesForUser(ctx context.Context, configs []*storage.ContinuousIntegrationConfig, user *permissions.UserDescriptor) ([]permissions.ResolvedRole, error) {
	rolesToAssign := set.NewStringSet()
	for _, cfg := range configs {
		for _, mapping := range cfg.GetMappings() {
			if valuesMatch(user.Attributes["sub"][0], mapping.GetValue()) && !rolesToAssign.Contains(mapping.GetRole()) {
				rolesToAssign.Add(mapping.GetRole())
			}
		}
	}
	if rolesToAssign.Cardinality() == 0 {
		return nil, errors.New("no roles are assigned to the user")
	}

	var resolvedRoles = make([]permissions.ResolvedRole, 0, rolesToAssign.Cardinality())
	for role := range rolesToAssign {
		resolvedRole, err := r.rolesDataStore.GetAndResolveRole(ctx, role)
		if err != nil {
			return nil, errors.Wrapf(err, "resolving role %q", role)
		}
		if resolvedRole != nil && resolvedRole.GetRoleName() != authn.NoneRole {
			resolvedRoles = append(resolvedRoles, resolvedRole)
		}
	}

	return resolvedRoles, nil
}

func checkIfRegexp(expr string) *regexp.Regexp {
	parsedExpr, err := regexp.Compile(expr)
	if err != nil {
		return nil
	}
	return parsedExpr
}

func valuesMatch(claimValue string, expr string) bool {
	// The expression is either a simple string value or a regular expression.
	if regExpr := checkIfRegexp(expr); regExpr != nil {
		return regExpr.MatchString(claimValue)
	}
	// Otherwise if it is not a regular expression, fall back to string comparison.
	return claimValue == expr
}
