package m78to79

import (
	"testing"

	"github.com/stackrox/stackrox/migrator/migrations/policymigrationhelper"
	"github.com/stretchr/testify/suite"
)

func TestPolicyMigration(t *testing.T) {
	suite.Run(t, &policyUpdatesTestSuite{
		TestSuite: policymigrationhelper.TestSuite{
			ExpectedPoliciesDir: "testdata",
			PoliciesToMigrate:   policiesToMigrate,
			PreMigPoliciesFS:    preMigrationPolicyFiles,
			PreMigPoliciesDir:   preMigrationPolicyFilesDir,
		},
	})
}

type policyUpdatesTestSuite struct {
	policymigrationhelper.TestSuite
}

// Test that all unmodified policies are migrated
func (suite *policyUpdatesTestSuite) TestMigration() {
	suite.RunTests(updatePolicies)
}
