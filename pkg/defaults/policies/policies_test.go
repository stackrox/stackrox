package policies

import (
	"bufio"
	"os"
	"testing"

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
