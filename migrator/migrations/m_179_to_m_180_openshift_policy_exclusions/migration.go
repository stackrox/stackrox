package m179tom180

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
		StartingSeqNum: 179,
		VersionAfter:   &storage.Version{SeqNum: 180},
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
			PolicyFileName:  "exec-iptables.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "exec-iptables-root.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "automount_service_account_token.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "host_ipc.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "host_pids.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "pod_portforward.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "privilege_escalation.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "privileged.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "secret_env.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "sensitive_files.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "host_network.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "root_user.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "mount_propagation.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "dnf.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "latest_tag.json",
		},
	}
)

func updatePolicies(db postgres.DB) error {
	return postgresHelper.MigratePoliciesWithDiffs(db, policyDiffFS, policyDiffs)
}

func init() {
	migrations.MustRegisterMigration(migration)
}
