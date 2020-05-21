package deploytime

import (
	"testing"

	"github.com/stackrox/rox/central/detection"
	imagePolicies "github.com/stackrox/rox/image/policies"
	"github.com/stackrox/rox/pkg/defaults"
	detectionPkg "github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/detection/deploytime"
	"github.com/stackrox/rox/pkg/fixtures"
	pkgPolicies "github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/search/options/deployments"
	"github.com/stackrox/rox/pkg/searchbasedpolicies/matcher"
	"github.com/stretchr/testify/require"
)

func BenchmarkDefaultPolicies(b *testing.B) {
	b.StopTimer()

	builder := matcher.NewBuilder(
		matcher.NewRegistry(
			nil,
		),
		deployments.OptionsMap,
	)
	policySet = detection.NewPolicySet(nil, detectionPkg.NewLegacyPolicyCompiler(builder))

	defaults.PoliciesPath = imagePolicies.Directory()
	policies, err := defaults.Policies()
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
		_, err := detection.Detect(deploytime.DetectionContext{}, dep, images)
		require.NoError(b, err)
	}
}
