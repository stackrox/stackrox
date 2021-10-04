package m81to82

import (
	"testing"

	"github.com/stackrox/rox/migrator/migrations/policymigrationhelper"
	"github.com/stretchr/testify/suite"
)

func TestPolicyMigration(t *testing.T) {
	suite.Run(t, &policyUpdatesTestSuite{
		DiffTestSuite: policymigrationhelper.DiffTestSuite{
			PolicyDiffFS: policyDiffFS,
		},
	})
}

type policyUpdatesTestSuite struct {
	policymigrationhelper.DiffTestSuite
}

// Test that all unmodified policies are migrated
func (suite *policyUpdatesTestSuite) TestMigration() {
	// For default policies, MITRE fields are always migrated because they cannot be modified by users, hence skip the modified policy test.
	suite.RunTests(updatePolicies, true)
}
