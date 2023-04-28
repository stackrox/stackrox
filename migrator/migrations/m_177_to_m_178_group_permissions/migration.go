package m177tom178

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	permissionsetpostgresstore "github.com/stackrox/rox/migrator/migrations/m_177_to_m_178_group_permissions/permissionsetpostgresstore"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
)

const (
	batchSize = 500

	startSeqNum = 177
)

// Replacement resources
const (
	Administration = "Administration"
	Compliance     = "Compliance"
)

// Replaced resources
const (
	AllComments         = "AllComments"
	ComplianceRuns      = "ComplianceRuns"
	Config              = "Config"
	DebugLogs           = "DebugLogs"
	NetworkGraphConfig  = "NetworkGraphConfig"
	ProbeUpload         = "ProbeUpload"
	ScannerBundle       = "ScannerBundle"
	ScannerDefinitions  = "ScannerDefinitions"
	SensorUpgradeConfig = "SensorUpgradeConfig"
	ServiceIdentity     = "ServiceIdentity"
)

var (
	migration = types.Migration{
		StartingSeqNum: startSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startSeqNum + 1)}, // 178
		Run: func(database *types.Databases) error {
			return migrateReplacedResourcesInPermissionSets(database.PostgresDB)
		},
	}

	replacements = map[string]string{
		AllComments:         Administration,
		ComplianceRuns:      Compliance,
		Config:              Administration,
		DebugLogs:           Administration,
		NetworkGraphConfig:  Administration,
		ProbeUpload:         Administration,
		ScannerBundle:       Administration,
		ScannerDefinitions:  Administration,
		SensorUpgradeConfig: Administration,
		ServiceIdentity:     Administration,
	}
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

func migrateReplacedResourcesInPermissionSets(db postgres.DB) error {
	ctx := sac.WithAllAccess(context.Background())
	store := permissionsetpostgresstore.New(db)

	migratedPermissionSets := make([]*storage.PermissionSet, 0, batchSize)
	err := store.Walk(ctx, func(obj *storage.PermissionSet) error {
		changed := false
		// Copy the permission set, removing the deprecated resource permissions, and keeping the
		// lowest access level between that of deprecated resource and their replacement
		// for the replacement resource.
		newPermissionSet := obj.Clone()
		newPermissionSet.ResourceToAccess = make(map[string]storage.Access, len(obj.GetResourceToAccess()))
		for resource, accessLevel := range obj.GetResourceToAccess() {
			if replacement, found := replacements[resource]; found {
				changed = true
				resource = replacement
			}
			newPermissionSet.ResourceToAccess[resource] =
				propagateAccessForPermission(resource, accessLevel, newPermissionSet.GetResourceToAccess())
		}
		if !changed {
			return nil
		}
		migratedPermissionSets = append(migratedPermissionSets, newPermissionSet)
		if len(migratedPermissionSets) >= batchSize {
			err := store.UpsertMany(ctx, migratedPermissionSets)
			if err != nil {
				return err
			}
			migratedPermissionSets = migratedPermissionSets[:0]
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(migratedPermissionSets) > 0 {
		return store.UpsertMany(ctx, migratedPermissionSets)
	}
	return nil
}
