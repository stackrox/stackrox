package m69tom70

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
		StartingSeqNum: 69,
		VersionAfter:   &storage.Version{SeqNum: 70},
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

	fieldsToCompare = []policymigrationhelper.FieldComparator{policymigrationhelper.PolicySectionComparator, policymigrationhelper.DescriptionComparator}

	policiesToMigrate = map[string]policymigrationhelper.PolicyChanges{
		"e9635b83-4ec5-4e7a-9be1-1bcdd6d82bb7": {
			FieldsToCompare: fieldsToCompare,
			ToChange: policymigrationhelper.PolicyUpdates{
				PolicySections: []*storage.PolicySection{
					{
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: "Process Name",
								Values:    []*storage.PolicyValue{{Value: ".*sgminer|.*cgminer|.*cpuminer|.*minerd|.*geth|.*ethminer|.*xmr-stak.*|.*xmrminer|.*cpuminer-multi|.*xmrig"}},
							},
						},
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

		policy := new(storage.Policy)
		err = jsonpb.Unmarshal(bytes.NewReader(contents), policy)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to unmarshal policy (%s) json", policyID)
		}
		comparisonPolicies[policyID] = policy
	}
	return comparisonPolicies, nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
