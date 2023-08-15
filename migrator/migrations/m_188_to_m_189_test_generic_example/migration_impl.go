package m188tom189

import (
	"embed"

	policypostgresstore "github.com/stackrox/rox/migrator/migrations/m_188_to_m_189_test_generic_example/policy/store"
	"github.com/stackrox/rox/migrator/migrations/policymigrationhelper"
	"github.com/stackrox/rox/migrator/types"
)

var (
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

func migrate(databases *types.Databases) error {
	policyStore := policypostgresstore.New(databases.PostgresDB)

	return policymigrationhelper.MigratePoliciesWithDiffsAndStore(
		policyDiffFS,
		policyDiffs,
		policyStore.Exists,
		policyStore.Get,
		policyStore.Upsert,
	)
}
