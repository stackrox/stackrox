package m81to82

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
		StartingSeqNum: 81,
		VersionAfter:   &storage.Version{SeqNum: 82},
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

	// We will want to migrate only if the existing policy sections, title, description and remediation haven't changed.
	fieldsToCompare = []policymigrationhelper.FieldComparator{
		policymigrationhelper.PolicySectionComparator,
		policymigrationhelper.NameComparator,
		policymigrationhelper.DescriptionComparator,
		policymigrationhelper.RemediationComparator,
	}

	policyDiffs = []policymigrationhelper.PolicyDiff{
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "docker_sock.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "fixable_cves.json",
		},
	}
)

func updatePolicies(db *bolt.DB) error {
	return policymigrationhelper.MigratePoliciesWithDiffs(db, policyDiffFS, policyDiffs)
}

func init() {
	migrations.MustRegisterMigration(migration)
}
