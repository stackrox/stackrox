package m68tom69

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
		StartingSeqNum: 68,
		VersionAfter:   &storage.Version{SeqNum: 69},
		Run: func(databases *types.Databases) error {
			err := migrateRoles(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "updating roles schema")
			}
			return nil
		},
	}

	rolesBucket = []byte("roles")

	// AllResources contains list of all resources. Copied from central/role/resources/list.go
	// to avoid change in the behaviour
	AllResources = [45]string{
		"APIToken",
		"Alert",
		"AllComments",
		"AuthPlugin",
		"AuthProvider",
		"BackupPlugins",
		"Cluster",
		"Compliance",
		"ComplianceRunSchedule",
		"ComplianceRuns",
		"Config",
		"CVE",
		"DebugLogs",
		"Deployment",
		"Detection",
		"Group",
		"Image",
		"ImageComponent",
		"ImageIntegration",
		"Indicator",
		"K8sRole",
		"K8sRoleBinding",
		"K8sSubject",
		"Licenses",
		"LogIntegration",
		"Namespace",
		"NetworkBaseline",
		"NetworkGraph",
		"NetworkGraphConfig",
		"NetworkPolicy",
		"Node",
		"Notifier",
		"Policy",
		"ProbeUpload",
		"ProcessWhitelist",
		"Role",
		"Risk",
		"ScannerBundle",
		"ScannerDefinitions",
		"Secret",
		"SensorUpgradeConfig",
		"ServiceAccount",
		"ServiceIdentity",
		"User",
		"WatchedImage",
	}
)

func migrateRoles(db *bbolt.DB) error {
	rolesToMigrate := make(map[string]*storage.Role)
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(rolesBucket)
		if bucket == nil {
			return errors.Errorf("bucket %s not found", rolesBucket)
		}
		return bucket.ForEach(func(k, v []byte) error {
			role := &storage.Role{}
			if err := proto.Unmarshal(v, role); err != nil {
				log.WriteToStderrf("Failed to unmarshal role data for key %s: %v", k, err)
				return nil
			}
			if role.GetGlobalAccess() == storage.Access_NO_ACCESS {
				return nil // no need to migrate
			}
			rolesToMigrate[string(k)] = role
			return nil
		})
	})
	if err != nil {
		return errors.Wrap(err, "reading role data")
	}

	if len(rolesToMigrate) == 0 {
		return nil // nothing to do
	}

	for _, role := range rolesToMigrate {
		completeResourceList(role)
		role.GlobalAccess = storage.Access_NO_ACCESS
	}
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(rolesBucket)
		if bucket == nil {
			return errors.Errorf("bucket %s not found", rolesBucket)
		}
		for id, role := range rolesToMigrate {
			bytes, err := proto.Marshal(role)
			if err != nil {
				log.WriteToStderrf("failed to marshal migrated role for key %s: %v", id, err)
				continue
			}
			if err := bucket.Put([]byte(id), bytes); err != nil {
				return err
			}
		}
		return nil
	})
}

func completeResourceList(role *storage.Role) {
	if role.GetGlobalAccess() == storage.Access_NO_ACCESS {
		return
	}
	if role.GetResourceToAccess() == nil {
		role.ResourceToAccess = make(map[string]storage.Access)
	}
	for _, resource := range AllResources {
		// Evaluates to true if the resource does not exist in the map because
		// it is guaranteed here that role.GetGlobalAccess() > 0.
		if role.ResourceToAccess[resource] < role.GetGlobalAccess() {
			role.ResourceToAccess[resource] = role.GetGlobalAccess()
		}
	}
}

func init() {
	migrations.MustRegisterMigration(migration)
}
