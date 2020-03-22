package m17tom18

import (
	"bytes"
	"fmt"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

var migration = types.Migration{
	StartingSeqNum: 17,
	VersionAfter:   storage.Version{SeqNum: 18},
	Run:            updateAlertRetentionConfig,
}

func init() {
	migrations.MustRegisterMigration(migration)
}

var (
	configBucket = []byte("config")
	configKey    = []byte{0}
)

func updateAlertRetentionConfig(boltDB *bolt.DB, badgerDB *badger.DB) error {
	exists, err := bolthelpers.BucketExists(boltDB, configBucket)
	if err != nil || !exists {
		return err
	}
	ref := bolthelpers.TopLevelRef(boltDB, configBucket)
	err = ref.Update(func(b *bolt.Bucket) error {
		return b.ForEach(func(k, v []byte) error {
			if !bytes.Equal(k, configKey) {
				return fmt.Errorf("Invalid config bucket key: %v", k)
			}
			var config storage.Config
			err = proto.Unmarshal(v, &config)
			if err != nil {
				return err
			}
			setAlertConfig(&config)
			data, err := proto.Marshal(&config)
			if err != nil {
				return err
			}
			return b.Put(k, data)
		})
	})
	return err
}

func setAlertConfig(config *storage.Config) {
	private := config.GetPrivateConfig()
	switch private.GetAlertRetention().(type) {
	case *storage.PrivateConfig_DEPRECATEDAlertRetentionDurationDays:
		days := private.GetDEPRECATEDAlertRetentionDurationDays()
		private.AlertRetention = &storage.PrivateConfig_AlertConfig{
			AlertConfig: &storage.AlertRetentionConfig{
				ResolvedDeployRetentionDurationDays: days,
				DeletedRuntimeRetentionDurationDays: days,
				// previous version of alert retention didn't delete runtime for undeleted deployments, so we shouldn't either.
				AllRuntimeRetentionDurationDays: 0,
			},
		}
	}
}
