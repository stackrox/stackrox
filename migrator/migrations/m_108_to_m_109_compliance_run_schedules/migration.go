package m108tom109

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"github.com/tecbot/gorocksdb"
)

var (
	migration = types.Migration{
		StartingSeqNum: 108,
		VersionAfter:   &storage.Version{SeqNum: 109},
		Run: func(databases *types.Databases) error {
			return removeComplianceRunScheduleFromPermissionSets(databases.RocksDB)
		},
	}

	permissionName = "ComplianceRunSchedule"
	prefix         = []byte("permission_sets")

	readOpts  = gorocksdb.NewDefaultReadOptions()
	writeOpts = gorocksdb.NewDefaultWriteOptions()
)

func removeComplianceRunScheduleFromPermissionSets(db *gorocksdb.DB) error {
	it := db.NewIterator(readOpts)
	defer it.Close()
	wb := gorocksdb.NewWriteBatch()
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		ps := &storage.PermissionSet{}
		if err := proto.Unmarshal(it.Value().Data(), ps); err != nil {
			return errors.Wrap(err, "unable to unmarshal permission set")
		}
		if _, ok := ps.ResourceToAccess[permissionName]; !ok {
			continue
		}
		delete(ps.ResourceToAccess, permissionName)
		data, err := proto.Marshal(ps)
		if err != nil {
			return errors.Wrap(err, "unable to marshal permission set")
		}
		wb.Put(it.Key().Copy(), data)
	}
	if err := db.Write(writeOpts, wb); err != nil {
		return errors.Wrap(err, "writing to RocksDB")
	}
	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
