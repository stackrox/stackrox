package m50tom51

import (
	protobufTypes "github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/types"
	"github.com/tecbot/gorocksdb"

	bolt "go.etcd.io/bbolt"
)

var (
	imageIntegrationsBucketName = []byte("imageintegration")
	notifierBucketName          = []byte("notifier")
	externalBackupsBucketName   = []byte("externalBackups")
	integrationHealthPrefix     = []byte("integrationhealth")

	writeOpts = gorocksdb.NewDefaultWriteOptions()
)

type integrationInfo struct {
	id   string
	name string
}

func insertDefaultHealthForRegistries(boltdb *bolt.DB, rocksdb *gorocksdb.DB) error {
	var imageIntegrations []*storage.ImageIntegration

	err := boltdb.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(imageIntegrationsBucketName)
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			imageIntegration := &storage.ImageIntegration{}
			if err := proto.Unmarshal(v, imageIntegration); err != nil {
				// If anything fails to unmarshal roll back the transaction and abort
				return errors.Wrapf(err, "Failed to unmarshal image integration data for key %s", k)
			}
			imageIntegrations = append(imageIntegrations, imageIntegration)
			return nil
		})
	})

	if err != nil {
		return errors.Wrap(err, "Failed to read image integrations")
	}

	integrationInfos := make([]integrationInfo, 0, len(imageIntegrations))
	for _, i := range imageIntegrations {
		integrationInfos = append(integrationInfos, integrationInfo{
			id:   i.GetId(),
			name: i.GetName(),
		})
	}

	return addHealthStatusToDB(integrationInfos, storage.IntegrationHealth_IMAGE_INTEGRATION, rocksdb)
}

func insertDefaultHealthForNotifiers(boltdb *bolt.DB, rocksdb *gorocksdb.DB) error {
	var notifiers []*storage.Notifier

	err := boltdb.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(notifierBucketName)
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			notifier := &storage.Notifier{}
			if err := proto.Unmarshal(v, notifier); err != nil {
				// If anything fails to unmarshal roll back the transaction and abort
				return errors.Wrapf(err, "Failed to unmarshal notifier data for key %s", k)
			}
			notifiers = append(notifiers, notifier)
			return nil
		})
	})

	if err != nil {
		return errors.Wrap(err, "Failed to read notifiers")
	}
	integrationInfos := make([]integrationInfo, 0, len(notifiers))
	for _, n := range notifiers {
		integrationInfos = append(integrationInfos, integrationInfo{
			id:   n.GetId(),
			name: n.GetName(),
		})
	}
	return addHealthStatusToDB(integrationInfos, storage.IntegrationHealth_NOTIFIER, rocksdb)
}

func insertDefaultHealthForBackupPlugins(boltdb *bolt.DB, rocksdb *gorocksdb.DB) error {
	var backups []*storage.ExternalBackup

	err := boltdb.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(externalBackupsBucketName)
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			backup := &storage.ExternalBackup{}
			if err := proto.Unmarshal(v, backup); err != nil {
				// If anything fails to unmarshal roll back the transaction and abort
				return errors.Wrapf(err, "Failed to unmarshal external backup data for key %s", k)
			}
			backups = append(backups, backup)
			return nil
		})
	})

	if err != nil {
		return errors.Wrap(err, "Failed to read external backup plugins")
	}
	integrationInfos := make([]integrationInfo, 0, len(backups))
	for _, n := range backups {
		integrationInfos = append(integrationInfos, integrationInfo{
			id:   n.GetId(),
			name: n.GetName(),
		})
	}
	return addHealthStatusToDB(integrationInfos, storage.IntegrationHealth_BACKUP, rocksdb)
}

func addHealthStatusToDB(integrationInfos []integrationInfo, typ storage.IntegrationHealth_Type, rocksdb *gorocksdb.DB) error {
	rocksWriteBatch := gorocksdb.NewWriteBatch()
	defer rocksWriteBatch.Destroy()

	for _, i := range integrationInfos {
		bytes, err := proto.Marshal(&storage.IntegrationHealth{
			Id:            i.id,
			Name:          i.name,
			Type:          typ,
			Status:        storage.IntegrationHealth_UNINITIALIZED,
			ErrorMessage:  "",
			LastTimestamp: protobufTypes.TimestampNow(),
		})
		if err != nil {
			return err
		}
		key := rocksdbmigration.GetPrefixedKey(integrationHealthPrefix, []byte(i.id))
		rocksWriteBatch.Put(key, bytes)
	}
	return rocksdb.Write(writeOpts, rocksWriteBatch)
}

func insertDefaultHealthStatus(boltdb *bolt.DB, rocksdb *gorocksdb.DB) error {
	err := insertDefaultHealthForRegistries(boltdb, rocksdb)
	if err != nil {
		return err
	}
	err = insertDefaultHealthForNotifiers(boltdb, rocksdb)
	if err != nil {
		return err
	}
	err = insertDefaultHealthForBackupPlugins(boltdb, rocksdb)
	if err != nil {
		return err
	}
	return nil
}

var (
	migration = types.Migration{
		StartingSeqNum: 50,
		VersionAfter:   storage.Version{SeqNum: 51},
		Run: func(databases *types.Databases) error {
			err := insertDefaultHealthStatus(databases.BoltDB, databases.RocksDB)
			if err != nil {
				return errors.Wrap(err, "adding default health status for all integrations")
			}
			return nil
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}
