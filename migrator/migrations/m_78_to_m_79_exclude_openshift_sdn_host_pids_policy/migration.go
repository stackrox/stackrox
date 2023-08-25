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
		StartingSeqNum: 78,
		VersionAfter:   &storage.Version{SeqNum: 79},
		Run: func(databases *types.Databases) error {
			err := updatePolicies(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "updating policies")
			}
			return nil
		},
	}

	// These are the policies as they were _before_ migration. If the policy in central doesn't match this, it won't get upgraded
	preMigrationPolicyFilesDir = "policies_before_migration"
	//go:embed policies_before_migration/*.json
	preMigrationPolicyFiles embed.FS

	// We will want to migrate even if the list of default exclusions have been modified, because we are just adding a new one
	fieldsToCompare = []policymigrationhelper.FieldComparator{policymigrationhelper.PolicySectionComparator}

	policiesToMigrate = map[string]policymigrationhelper.PolicyChanges{
		"436811e7-892f-4da6-a0f5-8cc459f1b954": {
			FieldsToCompare: fieldsToCompare,
			ToChange: policymigrationhelper.PolicyUpdates{
				ExclusionsToAdd: []*storage.Exclusion{
					{
						Name:       "Don't alert on the openshift-sdn namespace",
						Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "openshift-sdn"}},
					},
				},
			},
		},
	}
)

func updatePolicies(db *bolt.DB) error {
	return policymigrationhelper.MigratePoliciesWithPreMigrationFS(db, policiesToMigrate, preMigrationPolicyFiles, preMigrationPolicyFilesDir)
}

func init() {
	migrations.MustRegisterMigration(migration)
}
