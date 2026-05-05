package m2m

import (
	"fmt"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/tokenbased"
	"github.com/stackrox/rox/pkg/auth/permissions"
)

// newProviderFromConfig returns a provider that can be used to extract tokens issued based on machine to machine
// configs.
func newProviderFromConfig(configID string, configType string, rm permissions.RoleMapper) authproviders.Provider {
	return tokenbased.NewTokenAuthProvider(
		configID,
		fmt.Sprintf("%s-%s", configType, configID),
		configType,
		tokenbased.WithRoleMapper(rm),
	)
}
