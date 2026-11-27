package m2m

import (
	"fmt"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/tokenbased"
	"github.com/stackrox/rox/pkg/auth/permissions"
)

// newProvider returns a provider that can be used to extract tokens that were issued
// based on data from machine to machine configurations.
func newProvider(providerID string, providerType string, rm permissions.RoleMapper) authproviders.Provider {
	return tokenbased.NewTokenAuthProvider(
		providerID,
		fmt.Sprintf("%s-%s", providerType, providerID),
		providerType,
		tokenbased.WithRoleMapper(rm),
	)
}
