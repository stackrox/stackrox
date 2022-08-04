package m107tom108

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"go.etcd.io/bbolt"
)

var (
	migration = types.Migration{
		StartingSeqNum: 107,
		VersionAfter:   storage.Version{SeqNum: 108},
		Run: func(databases *types.Databases) error {
			err := migratePS(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "updating PermissionSet schema")
			}
			return nil
		},
	}

	psBucket               = []byte("permission_sets")
	authPluginBucket       = []byte("authzPlugins")
	AuthPluginResourceName = "AuthPlugin"
)

func migratePS(db *bbolt.DB) error {
	psToMigrate := make(map[string]*storage.PermissionSet)
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(psBucket)
		if bucket == nil {
			return errors.Errorf("bucket %s not found", psBucket)
		}
		return bucket.ForEach(func(k, v []byte) error {
			ps := &storage.PermissionSet{}
			if err := proto.Unmarshal(v, ps); err != nil {
				log.WriteToStderrf("Failed to unmarshal permissionset data for key %s: %v", k, err)
				return nil
			}
			if _, ok := ps.ResourceToAccess[AuthPluginResourceName]; ok {
				psToMigrate[string(k)] = ps
			}
			return nil
		})
	})

	if err != nil {
		return errors.Wrap(err, "reading permissionset data")
	}

	if len(psToMigrate) == 0 {
		return nil // nothing to do
	}

	for _, ps := range psToMigrate {
		delete(ps.ResourceToAccess, AuthPluginResourceName)
	}

	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(psBucket)
		if bucket == nil {
			return errors.Errorf("bucket %s not found", psBucket)
		}
		for id, ps := range psToMigrate {
			bytes, err := proto.Marshal(ps)
			if err != nil {
				log.WriteToStderrf("failed to marshal migrated permissionset for key %s: %v", id, err)
				continue
			}
			if err := bucket.Put([]byte(id), bytes); err != nil {
				return err
			}
		}
		return tx.DeleteBucket(authPluginBucket)
	})
}

func init() {
	migrations.MustRegisterMigration(migration)
}
