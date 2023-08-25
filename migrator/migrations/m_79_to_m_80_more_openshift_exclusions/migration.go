package m78to79

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
		StartingSeqNum: 79,
		VersionAfter:   &storage.Version{SeqNum: 80},
		Run: func(databases *types.Databases) error {
			err := updatePolicies(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "updating policies")
			}
			return nil
		},
	}

	//go:embed policies_before_and_after
	policyDiffFS embed.FS

	// We will want to migrate even if the list of default exclusions have been modified, because we are just modifying.
	fieldsToCompare = []policymigrationhelper.FieldComparator{policymigrationhelper.PolicySectionComparator}

	policyDiffs = []policymigrationhelper.PolicyDiff{
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "dnf.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "host_network.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "no_resources_specified.json",
		},
	}
)

func updatePolicies(db *bolt.DB) error {
	return policymigrationhelper.MigratePoliciesWithDiffs(db, policyDiffFS, policyDiffs)
}

func init() {
	migrations.MustRegisterMigration(migration)
}
