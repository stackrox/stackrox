package m86tom87

import (
	"testing"

	"github.com/stackrox/stackrox/migrator/migrations/policymigrationhelper"
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
	suite.RunTests(updatePolicies, true)
}
