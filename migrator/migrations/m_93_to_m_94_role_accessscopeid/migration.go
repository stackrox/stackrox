package m93tom94

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/types"
	"github.com/tecbot/gorocksdb"
)

const unrestrictedScopeID = "io.stackrox.authz.accessscope.unrestricted"

var (
	migration = types.Migration{
		StartingSeqNum: 93,
		VersionAfter:   &storage.Version{SeqNum: 94},
		Run: func(databases *types.Databases) error {
			if err := updateRoles(databases.RocksDB); err != nil {
				return errors.Wrap(err,
					"set unrestricted scope ID for the roles without scope")
			}
			return nil
		},
	}
	rolesBucket = []byte("roles")
)

func getRolesToUpdate(db *gorocksdb.DB) ([]*storage.Role, error) {
	it := db.NewIterator(gorocksdb.NewDefaultReadOptions())
	defer it.Close()

	var roles []*storage.Role
	for it.Seek(rolesBucket); it.ValidForPrefix(rolesBucket); it.Next() {
		role := &storage.Role{}
		if err := proto.Unmarshal(it.Value().Data(), role); err != nil {
			return nil, errors.Wrapf(err, "Failed to unmarshal role data for key %v", it.Key().Data())
		}

		if role.AccessScopeId == "" {
			role.AccessScopeId = unrestrictedScopeID
			roles = append(roles, role)
		}
	}
	return roles, nil
}

func updateRoles(db *gorocksdb.DB) error {
	roles, err := getRolesToUpdate(db)
	if err != nil {
		return err
	}
	rocksWriteBatch := gorocksdb.NewWriteBatch()
	defer rocksWriteBatch.Destroy()
	for _, role := range roles {
		name := role.GetName()
		bytes, err := proto.Marshal(role)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal role data for name %q", name)
		}
		rocksWriteBatch.Put(rocksdbmigration.GetPrefixedKey(rolesBucket, []byte(name)), bytes)
	}
	if err := db.Write(gorocksdb.NewDefaultWriteOptions(), rocksWriteBatch); err != nil {
		return errors.Wrap(err, "failed to write to rocksdb")
	}
	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
