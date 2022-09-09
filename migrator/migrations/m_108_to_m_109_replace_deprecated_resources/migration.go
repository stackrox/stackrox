package m108tom109

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/types"
	"github.com/tecbot/gorocksdb"
)

const (
	batchSize = 500

	// Replacement resources
	Access              = "Access"
	Administration      = "Administration"
	Compliance          = "Compliance"
	DeploymentExtension = "DeploymentExtension"
	Image               = "Image"
	Integration         = "Integration"
	// Replaced resources
	AllComments           = "AllComments"
	APIToken              = "APIToken"
	AuthProvider          = "AuthProvider"
	BackupPlugins         = "BackupPlugins"
	ComplianceRuns        = "ComplianceRuns"
	ComplianceRunSchedule = "ComplianceRunSchedule"
	Config                = "Config"
	DebugLogs             = "DebugLogs"
	Group                 = "Group"
	ImageComponent        = "ImageComponent"
	ImageIntegration      = "ImageIntegration"
	Indicator             = "Indicator"
	Licenses              = "Licenses"
	NetworkBaseline       = "NetworkBaseline"
	NetworkGraphConfig    = "NetworkGraphConfig"
	Notifier              = "Notifier"
	ProbeUpload           = "ProbeUpload"
	ProcessWhitelist      = "ProcessWhitelist"
	Risk                  = "Risk"
	Role                  = "Role"
	ScannerBundle         = "ScannerBundle"
	ScannerDefinitions    = "ScannerDefinitions"
	SensorUpgradeConfig   = "SensorUpgradeConfig"
	ServiceIdentity       = "ServiceIdentity"
	SignatureIntegration  = "SignatureIntegration"
	User                  = "User"
)

var (
	migration = types.Migration{
		StartingSeqNum: 108,
		VersionAfter:   storage.Version{SeqNum: 109},
		Run: func(databases *types.Databases) error {
			return migatePermissionSets(databases.RocksDB)
		},
	}

	prefix = []byte("permission_sets")

	replacements = map[string]string{
		AllComments:           Administration,
		APIToken:              Integration,
		AuthProvider:          Access,
		BackupPlugins:         Integration,
		ComplianceRuns:        Compliance,
		ComplianceRunSchedule: Administration,
		Config:                Administration,
		DebugLogs:             Administration,
		Group:                 Access,
		ImageComponent:        Image,
		ImageIntegration:      Integration,
		Indicator:             DeploymentExtension,
		Licenses:              Access,
		NetworkBaseline:       DeploymentExtension,
		NetworkGraphConfig:    Administration,
		Notifier:              Integration,
		ProbeUpload:           Administration,
		ProcessWhitelist:      DeploymentExtension,
		Risk:                  DeploymentExtension,
		Role:                  Access,
		ScannerBundle:         Administration,
		ScannerDefinitions:    Administration,
		SensorUpgradeConfig:   Administration,
		ServiceIdentity:       Administration,
		SignatureIntegration:  Integration,
		User:                  Access,
	}

	readOpts  = gorocksdb.NewDefaultReadOptions()
	writeOpts = gorocksdb.NewDefaultWriteOptions()
)

func propagatePermission(resource string, accessLevel storage.Access, permissions map[string]storage.Access) {
	if _, found := permissions[resource]; !found {
		permissions[resource] = accessLevel
	} else {
		oldLevel := permissions[resource]
		if accessLevel > oldLevel {
			permissions[resource] = accessLevel
		}
	}
}

func migatePermissionSets(db *gorocksdb.DB) error {
	it := db.NewIterator(readOpts)
	defer it.Close()
	wb := gorocksdb.NewWriteBatch()
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		permissions := &storage.PermissionSet{}
		if err := proto.Unmarshal(it.Value().Data(), permissions); err != nil {
			return errors.Wrap(err, "unable to unmarshal permission set")
		}
		// Copy the permission set, removing the deprecated resource permissions, and keeping the
		// highest access level between that of deprecated resource and their replacement
		// for the replacement resource.
		newPermissionSet := &storage.PermissionSet{}
		newPermissionSet.Id = permissions.GetId()
		newPermissionSet.Name = permissions.GetName()
		newPermissionSet.Description = permissions.GetDescription()
		if len(permissions.GetResourceToAccess()) > 0 {
			newPermissionSet.ResourceToAccess = make(map[string]storage.Access)
		}
		for resource, accessLevel := range permissions.GetResourceToAccess() {
			newResource := resource
			if _, found := replacements[resource]; found {
				newResource = replacements[resource]
			}
			propagatePermission(newResource, accessLevel, newPermissionSet.ResourceToAccess)
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
