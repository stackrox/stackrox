package m180tom181

import (
	"embed"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	postgresHelper "github.com/stackrox/rox/migrator/migrations/m_179_to_m_180_openshift_policy_exclusions/postgres"
	"github.com/stackrox/rox/migrator/migrations/policymigrationhelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	migration = types.Migration{
		StartingSeqNum: 180,
		VersionAfter:   &storage.Version{SeqNum: 181},
		Run: func(databases *types.Databases) error {
			err := updatePolicies(databases.PostgresDB)
			if err != nil {
				return errors.Wrap(err, "updating policies")
			}
			return nil
		},
	}

	//go:embed policies_before_and_after
	policyDiffFS embed.FS

	// We want to migrate only if the existing policy sections,name and description haven't changed.
	fieldsToCompare = []postgresHelper.FieldComparator{
		policymigrationhelper.DescriptionComparator,
		policymigrationhelper.PolicySectionComparator,
		policymigrationhelper.NameComparator,
	}

	policyDiffs = []postgresHelper.PolicyDiff{
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "dnf.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "exec-dnf.json",
		},
	}
)

func updatePolicies(db postgres.DB) error {
	return postgresHelper.MigratePoliciesWithDiffs(db, policyDiffFS, policyDiffs)
}

func init() {
	migrations.MustRegisterMigration(migration)
}
