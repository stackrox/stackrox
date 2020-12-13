package m52tom53

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

var (
	imageIntegrationBucket = []byte("imageintegrations")
	externalBackupsBucket  = []byte("externalBackups")
)

var (
	migration = types.Migration{
		StartingSeqNum: 52,
		VersionAfter:   storage.Version{SeqNum: 53},
		Run: func(databases *types.Databases) error {
			return migrateAWSEndpoints(databases.BoltDB)
		},
	}
)

func migrateExternalBackups(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(externalBackupsBucket)
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			var externalBackup storage.ExternalBackup
			if err := proto.Unmarshal(v, &externalBackup); err != nil {
				return err
			}
			if externalBackup.GetType() != "s3" {
				return nil
			}
			s3 := externalBackup.GetS3()
			if s3.GetEndpoint() != "" {
				return nil
			}
			s3.Endpoint = fmt.Sprintf("s3.%s.amazonaws.com", s3.GetRegion())

			newValue, err := proto.Marshal(&externalBackup)
			if err != nil {
				return errors.Wrapf(err, "error marshalling external backup %s", k)
			}
			return bucket.Put(k, newValue)
		})
	})
}

func migrateECR(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(imageIntegrationBucket)
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			var imageIntegration storage.ImageIntegration
			if err := proto.Unmarshal(v, &imageIntegration); err != nil {
				return err
			}
			if imageIntegration.GetType() != "ecr" {
				return nil
			}
			ecr := imageIntegration.GetEcr()
			if ecr.GetEndpoint() != "" {
				return nil
			}
			ecr.Endpoint = fmt.Sprintf("ecr.%s.amazonaws.com", ecr.GetRegion())
			newValue, err := proto.Marshal(&imageIntegration)
			if err != nil {
				return errors.Wrapf(err, "error marshalling external backup %s", k)
			}
			return bucket.Put(k, newValue)
		})
	})
}

func migrateAWSEndpoints(db *bolt.DB) error {
	if err := migrateExternalBackups(db); err != nil {
		return errors.Wrap(err, "migrating aws external backups")
	}
	if err := migrateECR(db); err != nil {
		return errors.Wrap(err, "migrating ecr image integrations")
	}
	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
