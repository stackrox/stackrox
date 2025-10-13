package m185tom186

import (
	"embed"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	policypostgresstore "github.com/stackrox/rox/migrator/migrations/m_185_to_m_186_more_policy_migrations/policypostgresstore"
	"github.com/stackrox/rox/migrator/migrations/policymigrationhelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	migration = types.Migration{
		StartingSeqNum: 185,
		VersionAfter:   &storage.Version{SeqNum: 186},
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
			FieldsToCompare: fieldsToCompareForDescription,
			PolicyFileName:  "add_instruction.json",
		},
		{
			FieldsToCompare: fieldsToCompareForExclusions,
			PolicyFileName:  "apk.json",
		},
		{
			FieldsToCompare: fieldsToCompareForExclusions,
			PolicyFileName:  "automount_service_account_token.json",
		},
		{
			FieldsToCompare: fieldsToCompareForExclusions,
			PolicyFileName:  "dnf.json",
		},
		{
			FieldsToCompare: fieldsToCompareForExclusions,
			PolicyFileName:  "exec-dnf.json",
		},
		{
			FieldsToCompare: fieldsToCompareForExclusions,
			PolicyFileName:  "exec-remote-copy.json",
		},
		{
			FieldsToCompare: fieldsToCompareForExclusions,
			PolicyFileName:  "host_network.json",
		},
		{
			FieldsToCompare: fieldsToCompareForExclusions,
			PolicyFileName:  "no_resources_specified.json",
		},
		{
			FieldsToCompare: fieldsToCompareForExclusions,
			PolicyFileName:  "pod_exec.json",
		},
		{
			FieldsToCompare: fieldsToCompareForExclusions,
			PolicyFileName:  "privileged.json",
		},
		{
			FieldsToCompare: fieldsToCompareForExclusions,
			PolicyFileName:  "restricted_host_ports.json",
		},
		{
			FieldsToCompare: fieldsToCompareForExclusions,
			PolicyFileName:  "root_user.json",
		},
		{
			FieldsToCompare: fieldsToCompareForExclusions,
			PolicyFileName:  "secret_env.json",
		},
		{
			FieldsToCompare: fieldsToCompareForExclusions,
			PolicyFileName:  "apt.json",
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
