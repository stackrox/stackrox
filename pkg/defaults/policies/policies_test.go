package policies

import (
	"bufio"
	"os"
	"testing"

	"github.com/stackrox/rox/pkg/mitre"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
