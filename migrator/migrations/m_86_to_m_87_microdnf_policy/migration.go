package m86tom87

import (
	"embed"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/policymigrationhelper"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

var (
	migration = types.Migration{
		StartingSeqNum: 86,
		VersionAfter:   &storage.Version{SeqNum: 87},
		Run: func(databases *types.Databases) error {
			err := updatePolicies(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "adding microdnf to policies")
			}
			return nil
		},
	}

	//go:embed policies_before_and_after
	policyDiffFS embed.FS

	// We will want to migrate only if the existing policy sections and title haven't changed.
	fieldsToCompare = []policymigrationhelper.FieldComparator{
		policymigrationhelper.PolicySectionComparator,
		policymigrationhelper.RemediationComparator,
		policymigrationhelper.NameComparator,
	}

	policyDiffs = []policymigrationhelper.PolicyDiff{
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "f95ff08d-130a-465a-a27e-32ed1fb05555.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "ddb7af9c-5ec1-45e1-a0cf-c36e3ef2b2ce.json",
		},
	}
)

func updatePolicies(db *bolt.DB) error {
	return policymigrationhelper.MigratePoliciesWithDiffs(db, policyDiffFS, policyDiffs)
}

func init() {
	migrations.MustRegisterMigration(migration)
}
