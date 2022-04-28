package m97tom98

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/gorm/models"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"github.com/tecbot/gorocksdb"
	"gorm.io/gorm"
)

var (
	migration = types.Migration{
		StartingSeqNum: 97,
		VersionAfter:   storage.Version{SeqNum: 98},
		Run: func(databases *types.Databases) error {
			if err := moveIntegrationHealths(databases.RocksDB, databases.PostgresDB); err != nil {
				return errors.Wrap(err,
					"moving integration health from rocksdb to postgres")
			}
			return nil
		},
	}
	rocksdbBucket = []byte("integrationhealth")
	postgresTable = []byte("integrationhealth")
)

func moveIntegrationHealths(rocksDB *gorocksdb.DB, postgresDB *gorm.DB) error {
	it := rocksDB.NewIterator(gorocksdb.NewDefaultReadOptions())
	defer it.Close()
	db := postgresDB.Table(models.IntegrationHealthTableName)
	err := db.AutoMigrate(&models.IntegrationHealth{})
	if err != nil {
		log.WriteToStderrf("failed to auto migrate %v", err)
		return err
	}
	var conv []*models.IntegrationHealth
	for it.Seek(rocksdbBucket); it.ValidForPrefix(rocksdbBucket); it.Next() {
		r := &storage.IntegrationHealth{}
		if err := proto.Unmarshal(it.Value().Data(), r); err != nil {
			return errors.Wrapf(err, "Failed to unmarshal integration health data for key %v", it.Key().Data())
		}
		conv = append(conv, &models.IntegrationHealth{
			Id:         r.GetId(),
			Serialized: it.Value().Data(),
		})
	}

	log.WriteToStderr(fmt.Sprintf("converted %d integration health", len(conv)))
	tx := db.Model(&models.IntegrationHealth{}).CreateInBatches(conv, 5000)
	if tx.Error != nil {
		tx.Rollback()
		return tx.Error
	}
	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
