package m14tom15

import (
	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

var migration = types.Migration{
	StartingSeqNum: 14,
	VersionAfter:   storage.Version{SeqNum: 15},
	Run:            updateAllRoles,
}

const (
	debugMetrics = "DebugMetrics"
)

var (
	rolesBucketName = []byte("roles")
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func updateAllRoles(db *bolt.DB, _ *badger.DB) error {
	rolesBucket := bolthelpers.TopLevelRef(db, rolesBucketName)
	err := rolesBucket.Update(func(b *bolt.Bucket) error {
		return b.ForEach(func(k, v []byte) error {
			var role storage.Role
			if err := proto.Unmarshal(v, &role); err != nil {
				return errors.Wrap(err, "unmarshaling role")
			}

			if _, found := role.GetResourceToAccess()[debugMetrics]; !found {
				return nil
			}

			delete(role.GetResourceToAccess(), debugMetrics)

			bytes, err := proto.Marshal(&role)
			if err != nil {
				return err
			}

			return b.Put(k, bytes)
		})
	})

	if err != nil {
		return err
	}

	return nil
}
