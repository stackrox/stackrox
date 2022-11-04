package m90tom91

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"github.com/tecbot/gorocksdb"
)

const (
	batchSize = 500
)

var (
	imageResource             = "Image"
	vulnMgmtRequestsResource  = "VulnerabilityManagementRequests"
	vulnMgmtApprovalsResource = "VulnerabilityManagementApprovals"

	permissionSetPrefix = []byte("permission_sets")

	migration = types.Migration{
		StartingSeqNum: 90,
		VersionAfter:   &storage.Version{SeqNum: 91},
		Run:            updateVulnSnoozePermissions,
	}

	readOpts  = gorocksdb.NewDefaultReadOptions()
	writeOpts = gorocksdb.NewDefaultWriteOptions()
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func updateVulnSnoozePermissions(databases *types.Databases) error {
	it := databases.RocksDB.NewIterator(readOpts)
	defer it.Close()

	wb := gorocksdb.NewWriteBatch()
	for it.Seek(permissionSetPrefix); it.ValidForPrefix(permissionSetPrefix); it.Next() {
		key := it.Key().Copy()

		var permSet storage.PermissionSet
		if err := proto.Unmarshal(it.Value().Data(), &permSet); err != nil {
			return errors.Wrapf(err, "unmarshaling permission set %s", key)
		}
		accessMap := permSet.GetResourceToAccess()
		if accessMap == nil {
			continue
		}
		if access, imagePermFound := accessMap[imageResource]; !imagePermFound || access != storage.Access_READ_WRITE_ACCESS {
			continue
		}

		permSet.ResourceToAccess[vulnMgmtRequestsResource] = storage.Access_READ_WRITE_ACCESS
		permSet.ResourceToAccess[vulnMgmtApprovalsResource] = storage.Access_READ_WRITE_ACCESS

		newData, err := proto.Marshal(&permSet)
		if err != nil {
			return errors.Wrapf(err, "marshaling permission set %s", key)
		}
		wb.Put(key, newData)

		if wb.Count() == batchSize {
			if err := databases.RocksDB.Write(writeOpts, wb); err != nil {
				return errors.Wrap(err, "writing to RocksDB")
			}
			wb.Clear()
		}
	}

	if wb.Count() != 0 {
		if err := databases.RocksDB.Write(writeOpts, wb); err != nil {
			return errors.Wrap(err, "writing final batch to RocksDB")
		}
	}
	return nil
}
