package m172tom173

import (
	"embed"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	postgresHelper "github.com/stackrox/rox/migrator/migrations/m_172_to_m_173_openshift_policy_exclusion/postgres"
	"github.com/stackrox/rox/migrator/migrations/policymigrationhelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	migration = types.Migration{
		StartingSeqNum: 172,
		VersionAfter:   &storage.Version{SeqNum: 173},
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

	// We want to migrate only if the existing policy sections and title haven't changed.
	fieldsToCompare = []postgresHelper.FieldComparator{
		policymigrationhelper.DescriptionComparator,
	}

	policyDiffs = []postgresHelper.PolicyDiff{
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "containers_should_run_as_a_non-root_user.json",
		},
	}
)

func updatePolicies(db *postgres.DB) error {
	return postgresHelper.MigratePoliciesWithDiffs(db, policyDiffFS, policyDiffs)
}

func init() {
	migrations.MustRegisterMigration(migration)
}
