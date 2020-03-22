package m12to13

import (
	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

var migration = types.Migration{
	StartingSeqNum: 12,
	VersionAfter:   storage.Version{SeqNum: 13},
	Run:            migrateDefaultRetentionDurations,
}

var (
	configBucket = []byte("config")
	configKey    = []byte("\x00")
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func migrateDefaultRetentionDurations(db *bolt.DB, _ *badger.DB) error {
	var existingConfig storage.Config
	var existingPublicConfig *storage.PublicConfig

	err := db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(configBucket)
		if err != nil {
			return err
		}
		val := bucket.Get(configKey)
		if val == nil {
			return nil
		}
		err = proto.Unmarshal(val, &existingConfig)
		if err != nil {
			return err
		}
		existingPublicConfig = existingConfig.GetPublicConfig()
		return nil
	})

	if err != nil {
		return err
	}

	return updateConfig(db, existingPublicConfig)
}

func updateConfig(db *bolt.DB, existingPublicConfig *storage.PublicConfig) error {
	data, err := proto.Marshal(&storage.Config{
		PublicConfig: existingPublicConfig,
		PrivateConfig: &storage.PrivateConfig{
			ImageRetentionDurationDays: 7,
		},
	})

	if err != nil {
		return err
	}
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(configBucket)
		return bucket.Put(configKey, data)
	})
}
