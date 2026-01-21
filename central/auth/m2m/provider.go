package m2m

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/tokenbased"
	"github.com/stackrox/rox/pkg/auth/authproviders/tokenbasedsource"
	"github.com/stackrox/rox/pkg/auth/permissions"
)

// newProviderFromConfig returns a provider that can be used to extract tokens issued based on machine to machine
// configs.
func newProviderFromConfig(config *storage.AuthMachineToMachineConfig, rm permissions.RoleMapper) authproviders.Provider {
	return tokenbased.NewTokenAuthProvider(
		config.GetId(),
		fmt.Sprintf("%s-%s", config.GetType().String(), config.GetId()),
		config.GetType().String(),
		tokenbased.WithRoleMapper(rm),
	)
}
