package m110tom111

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

// Replacement resources
const (
	Access              = "Access"
	DeploymentExtension = "DeploymentExtension"
	Image               = "Image"
	Integration         = "Integration"
)

// Replaced resources
const (
	APIToken             = "APIToken"
	AuthProvider         = "AuthProvider"
	BackupPlugins        = "BackupPlugins"
	Group                = "Group"
	ImageComponent       = "ImageComponent"
	ImageIntegration     = "ImageIntegration"
	Indicator            = "Indicator"
	Licenses             = "Licenses"
	NetworkBaseline      = "NetworkBaseline"
	Notifier             = "Notifier"
	ProcessWhitelist     = "ProcessWhitelist"
	Risk                 = "Risk"
	SignatureIntegration = "SignatureIntegration"
	User                 = "User"
)

var (
	migration = types.Migration{
		StartingSeqNum: 110,
		VersionAfter:   &storage.Version{SeqNum: 111},
		Run: func(databases *types.Databases) error {
			return migrateReplacedResourcesInPermissionSets(databases.RocksDB)
		},
	}

	prefix = []byte("permission_sets")

	replacements = map[string]string{
		APIToken:             Integration,
		AuthProvider:         Access,
		BackupPlugins:        Integration,
		Group:                Access,
		ImageComponent:       Image,
		ImageIntegration:     Integration,
		Indicator:            DeploymentExtension,
		Licenses:             Access,
		NetworkBaseline:      DeploymentExtension,
		Notifier:             Integration,
		ProcessWhitelist:     DeploymentExtension,
		Risk:                 DeploymentExtension,
		SignatureIntegration: Integration,
		User:                 Access,
	}

	readOpts  = gorocksdb.NewDefaultReadOptions()
	writeOpts = gorocksdb.NewDefaultWriteOptions()
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func propagateAccessForPermission(permission string, accessLevel storage.Access, permissionSet map[string]storage.Access) storage.Access {
	oldLevel, found := permissionSet[permission]
	if !found {
		return accessLevel
	}
	if accessLevel > oldLevel {
		return oldLevel
	}
	return accessLevel
}

func migrateReplacedResourcesInPermissionSets(db *gorocksdb.DB) error {
	it := db.NewIterator(readOpts)
	defer it.Close()
	wb := gorocksdb.NewWriteBatch()
	defer wb.Destroy()
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		permissions := &storage.PermissionSet{}
		if err := proto.Unmarshal(it.Value().Data(), permissions); err != nil {
			return errors.Wrap(err, "unable to unmarshal permission set")
		}
		// Copy the permission set, removing the deprecated resource permissions, and keeping the
		// lowest access level between that of deprecated resource and their replacement
		// for the replacement resource.
		newPermissionSet := permissions.Clone()
		newPermissionSet.ResourceToAccess = make(map[string]storage.Access, len(permissions.GetResourceToAccess()))
		for resource, accessLevel := range permissions.GetResourceToAccess() {
			if _, found := replacements[resource]; found {
				resource = replacements[resource]
			}
			newPermissionSet.ResourceToAccess[resource] =
				propagateAccessForPermission(resource, accessLevel, newPermissionSet.ResourceToAccess)
		}
		data, err := proto.Marshal(newPermissionSet)
		if err != nil {
			return errors.Wrap(err, "unable to marshal permission set")
		}
		wb.Put(it.Key().Copy(), data)
		if wb.Count() == batchSize {
			if err := db.Write(writeOpts, wb); err != nil {
				return errors.Wrap(err, "writing to RocksDB")
			}
			wb.Clear()
		}
	}
	if wb.Count() != 0 {
		if err := db.Write(writeOpts, wb); err != nil {
			return errors.Wrap(err, "writing final batch to RocksDB")
		}
	}
	return nil
}
