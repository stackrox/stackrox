package m106to107

import (
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/tecbot/gorocksdb"
	bolt "go.etcd.io/bbolt"
)

var (
	policiesBucket   = []byte("policies")
	categoriesBucket = []byte("policy_categories")

	migration = types.Migration{
		StartingSeqNum: 106,
		VersionAfter:   &storage.Version{SeqNum: 107},
		Run: func(databases *types.Databases) error {
			return addUserDefinedCategories(databases.BoltDB, databases.RocksDB)
		},
	}

	defaultCategories = set.NewStringSet(
		"Anomalous Activity",
		"Cryptocurrency Mining",
		"DevOps Best Practices",
		"Docker CIS",
		"Network Tools",
		"Package Management",
		"Privileges",
		"Kubernetes",
		"Kubernetes Events",
		"Security Best Practices",
		"System Modification",
		"Vulnerability Management",
	)
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func addUserDefinedCategories(boltdb *bolt.DB, rocksdb *gorocksdb.DB) error {
	/*
	 * Read all policies from bolt, and create a set of categories
	 * For each non default category, add it to the new table
	 */
	categoriesToAdd, err := fetchCategoriesToAdd(boltdb)
	if err != nil {
		return errors.Wrap(err, "failed to fetch user defined categories to be added")
	}
	// Add roles to rocksdb database.
	rocksWriteBatch := gorocksdb.NewWriteBatch()
	defer rocksWriteBatch.Destroy()
	for _, categoryName := range categoriesToAdd.AsSlice() {
		category := &storage.PolicyCategory{
			Id:        uuid.NewV4().String(),
			Name:      categoryName,
			IsDefault: false,
		}
		bytes, err := proto.Marshal(category)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal category data for name %q", category)
		}
		rocksWriteBatch.Put(rocksdbmigration.GetPrefixedKey(categoriesBucket, []byte(category.GetId())), bytes)
	}
	err = rocksdb.Write(gorocksdb.NewDefaultWriteOptions(), rocksWriteBatch)
	if err != nil {
		return errors.Wrap(err, "failed to write new categories to rocksdb")
	}
	return nil
}

func fetchCategoriesToAdd(db *bolt.DB) (set.FrozenStringSet, error) {
	categories := set.NewStringSet()
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policiesBucket)
		return bucket.ForEach(func(k, v []byte) error {
			policy := &storage.Policy{}
			if err := proto.Unmarshal(v, policy); err != nil {
				log.WriteToStderrf("Failed to unmarshal policy data for key %s: %v", k, err)
				return nil
			}
			for _, c := range policy.GetCategories() {
				categories.Add(strings.Title(c))
			}
			return nil
		})
	})
	if err != nil {
		return set.NewFrozenStringSet(), err
	}

	categoriesToAdd := categories.Difference(defaultCategories)
	return set.NewFrozenStringSet(categoriesToAdd.AsSlice()...), nil
}
