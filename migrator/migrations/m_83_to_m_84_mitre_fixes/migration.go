package m83to84

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
		StartingSeqNum: 83,
		VersionAfter:   &storage.Version{SeqNum: 84},
		Run: func(databases *types.Databases) error {
			err := updatePolicies(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "updating policies with MITRE ATT&CK")
			}
			return nil
		},
	}

	//go:embed policies_before_and_after
	policyDiffFS embed.FS

	// MITRE fields are locked. Therefore, no field comparison pre-condition.

	policyDiffs = []policymigrationhelper.PolicyDiff{
		{
			PolicyFileName: "access_central_secret.json",
		},
		{
			PolicyFileName: "exec-addgroup.json",
		},
		{
			PolicyFileName: "exec-adduser.json",
		},
		{
			PolicyFileName: "exec-make.json",
		},
		{
			PolicyFileName: "exec-remote-copy.json",
		},
		{
			PolicyFileName: "exec-sshd.json",
		},
		{
			PolicyFileName: "pod_portforward.json",
		},
		{
			PolicyFileName: "setuid_binaries.json",
		},
	}
)

func updatePolicies(db *bolt.DB) error {
	return policymigrationhelper.MigratePoliciesWithDiffs(db, policyDiffFS, policyDiffs)
}

func init() {
	migrations.MustRegisterMigration(migration)
}
