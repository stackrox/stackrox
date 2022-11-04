package m75to76

import (
	"bytes"
	"embed"
	"fmt"
	"path/filepath"

	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/policymigrationhelper"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

var (
	migration = types.Migration{
		StartingSeqNum: 75,
		VersionAfter:   &storage.Version{SeqNum: 76},
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
		"ddb7af9c-5ec1-45e1-a0cf-c36e3ef2b2ce": {
			FieldsToCompare: fieldsToCompare,
			ToChange: policymigrationhelper.PolicyUpdates{
				ExclusionsToAdd: []*storage.Exclusion{
					{
						Name:       "Don't alert on openshift-compliance namespace",
						Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "openshift-compliance"}},
					},
				},
			},
		},
	}
)

func updatePolicies(db *bolt.DB) error {
	comparisonPolicies, err := getComparisonPoliciesFromFiles()
	if err != nil {
		return err
	}

	return policymigrationhelper.MigratePolicies(db, policiesToMigrate, comparisonPolicies)
}

func getComparisonPoliciesFromFiles() (map[string]*storage.Policy, error) {
	comparisonPolicies := make(map[string]*storage.Policy)
	for policyID := range policiesToMigrate {
		path := filepath.Join(preMigrationPolicyFilesDir, fmt.Sprintf("%s.json", policyID))
		contents, err := preMigrationPolicyFiles.ReadFile(path)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to read file %s", path)
		}

		var policy storage.Policy
		err = jsonpb.Unmarshal(bytes.NewReader(contents), &policy)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to unmarshal policy (%s) json", policyID)
		}
		comparisonPolicies[policyID] = &policy
	}
	return comparisonPolicies, nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
