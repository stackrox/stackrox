package m187tom188

import (
	"embed"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	policypostgresstore "github.com/stackrox/rox/migrator/migrations/m_187_to_m_188_add_nftables_to_policy/policy/store"
	"github.com/stackrox/rox/migrator/migrations/policymigrationhelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	migration = types.Migration{
		StartingSeqNum: 187,
		VersionAfter:   &storage.Version{SeqNum: 188},
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

	// Add policy exclusions only if the existing name, description and policy sections haven't changed.
	fieldsToCompareForExclusions = []policymigrationhelper.FieldComparator{
		policymigrationhelper.NameComparator,
		policymigrationhelper.DescriptionComparator,
		policymigrationhelper.PolicySectionComparator,
	}

	// Update description only if the existing name, description and policy sections haven't changed.
	fieldsToCompareForDescription = []policymigrationhelper.FieldComparator{
		policymigrationhelper.NameComparator,
		policymigrationhelper.PolicySectionComparator,
	}

	policyDiffs = []policymigrationhelper.PolicyDiff{
		{
			FieldsToCompare: fieldsToCompareForExclusions,
			PolicyFileName:  "exec-iptables-root.json",
		},
		{
			FieldsToCompare: fieldsToCompareForExclusions,
			PolicyFileName:  "exec-iptables.json",
		},
	}
)

func updatePolicies(db postgres.DB) error {
	policyStore := policypostgresstore.New(db)

	return policymigrationhelper.MigratePoliciesWithDiffsAndStore(
		policyDiffFS,
		policyDiffs,
		policyStore.Exists,
		policyStore.Get,
		policyStore.Upsert,
	)
}

func init() {
	migrations.MustRegisterMigration(migration)
}
