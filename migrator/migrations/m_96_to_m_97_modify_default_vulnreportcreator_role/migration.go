package m96tom97

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/auth/permissions"
	permissionsUtils "github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/tecbot/gorocksdb"
)

var (
	migration = types.Migration{
		StartingSeqNum: 95,
		VersionAfter:   storage.Version{SeqNum: 96},
		Run: func(databases *types.Databases) error {
			if err := updateDefaultPermissionsForVulnCreatorRole(databases.RocksDB); err != nil {
				return errors.Wrap(err,
					"updating permissions for default vuln reporter roles")
			}
			return nil
		},
	}
	rolesBucket       = []byte("roles")
	permissionsBucket = []byte("permission_sets")

	newPermissions = []permissions.ResourceWithAccess{
		permissions.View(resources.VulnerabilityReports),   // required for vuln report configurations
		permissions.Modify(resources.VulnerabilityReports), // required for vuln report configurations
		permissions.View(resources.Role),                   // required for scopes
		permissions.View(resources.Image),                  // required to gather CVE data for the report
		permissions.View(resources.Notifier),               // required for vuln report configurations
	}
)

func getPermissionSet(db *gorocksdb.DB) (*storage.PermissionSet, error) {
	it := db.NewIterator(gorocksdb.NewDefaultReadOptions())
	defer it.Close()

	var err error
	for it.Seek(rolesBucket); it.ValidForPrefix(rolesBucket); it.Next() {
		r := &storage.Role{}
		if err := proto.Unmarshal(it.Value().Data(), r); err != nil {
			return nil, errors.Wrapf(err, "Failed to unmarshal role data for key %v", it.Key().Data())
		}

		if r.Name == role.VulnReporter {
			pit := db.NewIterator(gorocksdb.NewDefaultReadOptions())
			for pit.Seek(permissionsBucket); pit.ValidForPrefix(permissionsBucket); pit.Next() {
				p := &storage.PermissionSet{}
				if err := proto.Unmarshal(pit.Value().Data(), p); err != nil {
					return nil, errors.Wrapf(err, "Failed to unmarshal permission data for key %v", pit.Key().Data())
				}
				if p.Id == r.PermissionSetId {
					return p, nil
				}
			}
			return nil, errors.Wrapf(err, "failed to get permissions for role %s", r.Name)
		}
	}
	return nil, nil
}

func updateDefaultPermissionsForVulnCreatorRole(db *gorocksdb.DB) error {
	ps, err := getPermissionSet(db)
	if ps == nil {
		return errors.Wrapf(err, "failed to update permissions")
	}

	ps.ResourceToAccess = permissionsUtils.FromResourcesWithAccess(newPermissions...)

	rocksWriteBatch := gorocksdb.NewWriteBatch()
	defer rocksWriteBatch.Destroy()

	bytes, err := proto.Marshal(ps)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal permission data for id %q", ps.Id)
	}
	rocksWriteBatch.Put(rocksdbmigration.GetPrefixedKey(permissionsBucket, []byte(ps.Id)), bytes)

	if err := db.Write(gorocksdb.NewDefaultWriteOptions(), rocksWriteBatch); err != nil {
		return errors.Wrap(err, "failed to write to rocksdb")
	}
	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
