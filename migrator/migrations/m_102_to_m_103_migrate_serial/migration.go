package m102tom103

import (
	"strconv"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

var (
	bucketName = []byte("service_identities")

	migration = types.Migration{
		StartingSeqNum: 102,
		VersionAfter:   &storage.Version{SeqNum: 103},
		Run: func(databases *types.Databases) error {
			if err := migrateSerials(databases.BoltDB); err != nil {
				return errors.Wrap(err, "error migrating service identity serials")
			}
			return nil
		},
	}
)

func migrateSerials(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			si := &storage.ServiceIdentity{}
			if err := proto.Unmarshal(v, si); err != nil {
				return err
			}
			if serial, ok := si.Srl.(*storage.ServiceIdentity_Serial); ok && si.GetSerialStr() == "" {
				si.SerialStr = strconv.FormatInt(serial.Serial, 10)
			}
			data, err := si.Marshal()
			if err != nil {
				return errors.Wrapf(err, "marshalling service identity: %+v", si)
			}

			if err := bucket.Put(k, data); err != nil {
				return errors.Wrapf(err, "storing service identity: %+v", si)
			}
			return nil
		})
	})
}

func init() {
	migrations.MustRegisterMigration(migration)
}
