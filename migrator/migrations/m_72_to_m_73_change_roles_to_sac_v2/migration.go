package m72tom73

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/tecbot/gorocksdb"
	"go.etcd.io/bbolt"
)

var (
	migration = types.Migration{
		StartingSeqNum: 72,
		VersionAfter:   &storage.Version{SeqNum: 73},
		Run: func(databases *types.Databases) error {
			err := migrateRoles(databases.BoltDB, databases.RocksDB)
			if err != nil {
				return errors.Wrap(err, "updating roles schema")
			}
			return nil
		},
	}

	rolesBucket = []byte("roles")
	psBucket    = []byte("permission_sets")
)

func migrateRoles(boltdb *bbolt.DB, rocksdb *gorocksdb.DB) error {
	// Collect roles which need migration.
	rolesToMigrate := make(map[string]*storage.Role)
	err := boltdb.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(rolesBucket)
		if bucket == nil {
			return errors.Errorf("bucket %s not found", rolesBucket)
		}
		return bucket.ForEach(func(k, v []byte) error {
			role := &storage.Role{}
			if err := proto.Unmarshal(v, role); err != nil {
				return errors.Wrapf(err, "failed to unmarshal role data for key %s", k)
			}
			if role.GetPermissionSetId() != "" {
				return nil // No need to migrate.
			}
			rolesToMigrate[string(k)] = role
			return nil
		})
	})
	if err != nil {
		return errors.Wrap(err, "failed to read role data")
	}

	permissionSets := generatePermissionSets(rolesToMigrate)

	// Update permission set database.
	rocksWriteBatch := gorocksdb.NewWriteBatch()
	defer rocksWriteBatch.Destroy()
	for roleName, ps := range permissionSets {
		bytes, err := proto.Marshal(ps)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal permission set for role %q", roleName)
		}
		rocksWriteBatch.Put(rocksdbmigration.GetPrefixedKey(psBucket, []byte(ps.Id)), bytes)
	}
	err = rocksdb.Write(gorocksdb.NewDefaultWriteOptions(), rocksWriteBatch)
	if err != nil {
		return errors.Wrap(err, "failed to write to rocksdb")
	}

	// Update roles database.
	return boltdb.Update(func(tx *bbolt.Tx) error {
		return updateRoles(tx, rolesToMigrate)
	})
}

func generatePermissionSets(rolesToMigrate map[string]*storage.Role) map[string]*storage.PermissionSet {
	// Modify roles and create corresponding permission sets.
	permissionSets := make(map[string]*storage.PermissionSet, len(rolesToMigrate))
	for name, role := range rolesToMigrate {
		id := "io.stackrox.authz.permissionset." + uuid.NewV4().String()
		ps := &storage.PermissionSet{
			Id:               id,
			Name:             role.GetName() + " Permissions",
			Description:      "This permission set was created as part of the migration from the old Role format to the new Role + Permission Set format.",
			ResourceToAccess: role.GetResourceToAccess(),
		}
		permissionSets[name] = ps
		role.PermissionSetId = id
		role.ResourceToAccess = nil
	}
	return permissionSets
}

func updateRoles(tx *bbolt.Tx, rolesToMigrate map[string]*storage.Role) error {
	bucket := tx.Bucket(rolesBucket)
	if bucket == nil {
		return errors.Errorf("bucket %s not found", rolesBucket)
	}
	for id, role := range rolesToMigrate {
		bytes, err := proto.Marshal(role)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal migrated role for key %q", id)
		}
		if err := bucket.Put([]byte(id), bytes); err != nil {
			return errors.Wrapf(err, "failed to write migrated role with key %q to the store", id)
		}
	}
	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
