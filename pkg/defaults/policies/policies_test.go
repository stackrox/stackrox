package policies

import (
	"bufio"
	"os"
	"testing"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/mitre"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_DefaultPolicies_FilterByFeatureFlag(t *testing.T) {
	// Don't run test on release mode since all feature flags will be force-disabled
	if buildinfo.ReleaseBuild {
		t.Skip("Skipping this test for release build because it relies on feature flags")
	}

	// This is required here to be able to check if the policy is present in the final list of default policies
	// since the feature flag filtering is done by filename other than policy name.
	fileToPolicyName := map[string]string{
		"deployment_has_ingress_network_policy.json": "Deployments should have at least one ingress Network Policy",
	}

	for filename, ff := range featureFlagFileGuard {
		t.Setenv(ff.EnvVar(), "false")
		require.False(t, checkPoliciesContain(t, fileToPolicyName[filename]))
		t.Setenv(ff.EnvVar(), "true")
		require.True(t, checkPoliciesContain(t, fileToPolicyName[filename]))
	}
}

func checkPoliciesContain(t *testing.T, policyNameToCheck string) bool {
	policies, err := DefaultPolicies()
	require.NoError(t, err)
	for _, p := range policies {
		if p.Name == policyNameToCheck {
			return true
		}
	}
	return false
}

// This test ensures that anyone adding a new policy is aware of MITRE ATT&CK section.
// If this test fails please add MITRE ATT&CK to the policy (Ref: https://attack.mitre.org/matrices/enterprise/).
// If MITRE ATT&CK is not applicable, add the policy to exception list-"pkg/defaults/policies/mitre_exception_list".
func TestMitre(t *testing.T) {
	policies, err := DefaultPolicies()
	require.NoError(t, err)

	file, err := os.Open("mitre_exception_list")
	require.NoError(t, err)
	scanner := bufio.NewScanner(file)
	allowList := set.NewStringSet()
	for scanner.Scan() {
		allowList.Add(scanner.Text())
	}

	for _, policy := range policies {
		if len(policy.GetMitreAttackVectors()) == 0 {
			assert.Truef(t, allowList.Contains(policy.GetId()), "policy %s does not have MITRE ATT&CK vectors. "+
				"Please add MITRE ATT&CK to the policy (Ref: https://attack.mitre.org/matrices/enterprise/). "+
				"If MITRE ATT&CK is not applicable, add the policy to 'pkg/defaults/policies/mitre_exception_list'",
				policy.GetId())
		}
	}
}

func TestMitreIDsAreValid(t *testing.T) {
	policies, err := DefaultPolicies()
	require.NoError(t, err)

	mitreBundle, err := mitre.GetMitreBundle()
	require.NoError(t, err)
	vectors := mitre.FlattenMitreMatrices(mitreBundle.GetMatrices()...)

	tactics := make(map[string]struct{})
	techniques := make(map[string]struct{})
	for _, vector := range vectors {
		tactics[vector.GetTactic().GetId()] = struct{}{}
		for _, technique := range vector.GetTechniques() {
			techniques[technique.GetId()] = struct{}{}
		}
	}

	for _, policy := range policies {
		for _, vector := range policy.GetMitreAttackVectors() {
			assert.NotNil(t, tactics[vector.GetTactic()], "MITRE Tactic %s in policy %s is invalid", vector.GetTactic(), policy.GetName())
			for _, technique := range vector.GetTechniques() {
				assert.NotNil(t, techniques[technique], "MITRE Technique %s in policy %s is invalid", technique, policy.GetName())
			}
		}
	}
}
