package deployment

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

const (
	// IdentityBlastRadiusHeading is the risk result name for scores calculated by this multiplier.
	IdentityBlastRadiusHeading = "Identity Blast Radius"
)

type identityBlastRadiusMultiplier struct{}

// NewIdentityBlastRadius provides a multiplier that scores deployments based on service-account blast radius.
func NewIdentityBlastRadius() Multiplier {
	return &identityBlastRadiusMultiplier{}
}

// Score evaluates risk based on the service account permission level attached to the deployment.
func (m *identityBlastRadiusMultiplier) Score(_ context.Context, deployment *storage.Deployment, _ map[string][]*storage.Risk_Result) *storage.Risk_Result {
	if !deployment.GetAutomountServiceAccountToken() {
		return nil
	}

	permissionLevel := deployment.GetServiceAccountPermissionLevel()
	var score float32
	var description string
	switch permissionLevel {
	case storage.PermissionLevel_CLUSTER_ADMIN:
		score = 4
		description = "cluster-admin permissions"
	case storage.PermissionLevel_ELEVATED_CLUSTER_WIDE:
		score = 3
		description = "cluster-wide elevated permissions"
	case storage.PermissionLevel_ELEVATED_IN_NAMESPACE:
		score = 1.5
		description = "namespace-elevated permissions"
	default:
		return nil
	}

	serviceAccount := deployment.GetServiceAccount()
	if serviceAccount == "" {
		serviceAccount = "the deployment service account"
	}
	return &storage.Risk_Result{
		Name: IdentityBlastRadiusHeading,
		Factors: []*storage.Risk_Result_Factor{
			{Message: fmt.Sprintf("Service account %q has %s and its token is automounted", serviceAccount, description)},
		},
		Score: score,
	}
}
