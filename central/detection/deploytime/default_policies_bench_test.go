package deploytime

import (
	"testing"

	"github.com/stackrox/stackrox/central/detection"
	"github.com/stackrox/stackrox/pkg/booleanpolicy"
	"github.com/stackrox/stackrox/pkg/defaults/policies"
	"github.com/stackrox/stackrox/pkg/detection/deploytime"
	"github.com/stackrox/stackrox/pkg/fixtures"
	pkgPolicies "github.com/stackrox/stackrox/pkg/policies"
	"github.com/stretchr/testify/require"
)

func BenchmarkDefaultPolicies(b *testing.B) {
	b.StopTimer()

	policySet = detection.NewPolicySet(nil)

	policies, err := policies.DefaultPolicies()
	require.NoError(b, err)

	for _, policy := range policies {
		if pkgPolicies.AppliesAtDeployTime(policy) {
			require.NoError(b, policySet.UpsertPolicy(policy))
		}
	}

	detection := NewDetector(policySet)

	dep := fixtures.GetDeployment()
	images := fixtures.DeploymentImages()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err := detection.Detect(deploytime.DetectionContext{}, booleanpolicy.EnhancedDeployment{
			Deployment: dep,
			Images:     images,
		})
		require.NoError(b, err)
	}
}
