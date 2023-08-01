package n52ton53

// Code generation from pg-bindings generator disabled. To re-enable, check the gen.go file in
// central/role/store/permissionset/postgres
// central/role/store/role/postgres
// central/role/store/simpleaccessscope/postgres

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v73"
	"github.com/stackrox/rox/migrator/migrations/loghelper"
	"github.com/stackrox/rox/migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/legacypermissionsets"
	"github.com/stackrox/rox/migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/legacyroles"
	"github.com/stackrox/rox/migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/legacysimpleaccessscopes"
	pgPermissionSetStore "github.com/stackrox/rox/migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/postgrespermissionsets"
	pgReportConfigurationStore "github.com/stackrox/rox/migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/postgresreportconfigurations"
	pgRoleStore "github.com/stackrox/rox/migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/postgresroles"
	pgSimpleAccessScopeStore "github.com/stackrox/rox/migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/postgressimpleaccessscopes"
	"github.com/stackrox/rox/migrator/types"
	pkgMigrations "github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/uuid"
	"gorm.io/gorm"
)

const (
	accessScopeIDPrefix   = "io.stackrox.authz.accessscope."
	permissionSetIDPrefix = "io.stackrox.authz.permissionset."
)

var (
	startingSeqNum = pkgMigrations.BasePostgresDBVersionSeqNum() + 52 // 163

	migration = types.Migration{
		StartingSeqNum: startingSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startingSeqNum + 1)}, // 164
		Run: func(databases *types.Databases) error {
			return migrateAll(databases.DBCtx, databases.PkgRocksDB, databases.GormDB, databases.PostgresDB)
		},
	}
	batchSize = 1000
	log       = loghelper.LogWrapper{}

	accessScopeIDMapping = map[string]string{
		"denyall":      "ffffffff-ffff-fff4-f5ff-fffffffffffe",
		"unrestricted": "ffffffff-ffff-fff4-f5ff-ffffffffffff",
	}

	permissionsetIDMapping = map[string]string{
		"admin":                 "ffffffff-ffff-fff4-f5ff-ffffffffffff",
		"analyst":               "ffffffff-ffff-fff4-f5ff-fffffffffffe",
		"continuousintegration": "ffffffff-ffff-fff4-f5ff-fffffffffffd",
		"none":                  "ffffffff-ffff-fff4-f5ff-fffffffffffc",
		"scopemanager":          "ffffffff-ffff-fff4-f5ff-fffffffffffb",
		"sensorcreator":         "ffffffff-ffff-fff4-f5ff-fffffffffffa",
		"vulnmgmtapprover":      "ffffffff-ffff-fff4-f5ff-fffffffffff9",
		"vulnmgmtrequester":     "ffffffff-ffff-fff4-f5ff-fffffffffff8",
		"vulnreporter":          "ffffffff-ffff-fff4-f5ff-fffffffffff7",
	}
)

func migrateAll(ctx context.Context, rocksDatabase *rocksdb.RocksDB, gormDB *gorm.DB, postgresDB postgres.DB) error {
	legacyAccessScopeStore, err := legacysimpleaccessscopes.New(rocksDatabase)
	if err != nil {
		return err
	}
	if err := migrateAccessScopes(ctx, gormDB, postgresDB, legacyAccessScopeStore); err != nil {
		return errors.Wrap(err,
			"moving simple_access_scopes from rocksdb to postgres")
	}
	legacyPermissionSetStore, err := legacypermissionsets.New(rocksDatabase)
	if err != nil {
		return err
	}
	if err := migratePermissionSets(ctx, gormDB, postgresDB, legacyPermissionSetStore); err != nil {
		return errors.Wrap(err,
			"moving permission_sets from rocksdb to postgres")
	}
	legacyRoleStore, err := legacyroles.New(rocksDatabase)
	if err != nil {
		return err
	}
	if err := migrateRoles(ctx, gormDB, postgresDB, legacyRoleStore); err != nil {
		return errors.Wrap(err,
			"moving roles from rocksdb to postgres")
	}
	// This function call was added in 3.74 in order to cover an overlooked reference to access scope IDs.
	// Users who migrated to postgres with 3.73 may have report configurations relying on incorrect scope IDs.
	if err := migrateReportConfigurationScopeIDs(ctx, postgresDB); err != nil {
		return errors.Wrap(err, "updating access scope IDs for Report Configuration objects in postgres")
	}
	return nil
}

func convertAccessScopeID(accessScopeID string) string {
	identifierSuffix := strings.TrimPrefix(accessScopeID, accessScopeIDPrefix)
	replacement, found := accessScopeIDMapping[identifierSuffix]
	if found {
		return replacement
	}
	_, err := uuid.FromString(identifierSuffix)
	if err != nil {
		generatedID := uuid.NewV4().String()
		accessScopeIDMapping[identifierSuffix] = generatedID
		return generatedID
	}
	return identifierSuffix
}

func migrateAccessScopes(ctx context.Context, gormDB *gorm.DB, postgresDB postgres.DB, legacyStore legacysimpleaccessscopes.Store) error {
	store := pgSimpleAccessScopeStore.New(postgresDB)
	pgutils.CreateTableFromModel(context.Background(), gormDB, frozenSchema.CreateTableSimpleAccessScopesStmt)

	var simpleAccessScopes []*storage.SimpleAccessScope
	err := walkAccessScopes(ctx, legacyStore, func(obj *storage.SimpleAccessScope) error {
		accessScopeID := convertAccessScopeID(obj.GetId())
		obj.Id = accessScopeID
		simpleAccessScopes = append(simpleAccessScopes, obj)
		if len(simpleAccessScopes) == batchSize {
			if err := store.UpsertMany(ctx, simpleAccessScopes); err != nil {
				log.WriteToStderrf("failed to persist simple_access_scopes to store %v", err)
				return err
			}
			simpleAccessScopes = simpleAccessScopes[:0]
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(simpleAccessScopes) > 0 {
		if err = store.UpsertMany(ctx, simpleAccessScopes); err != nil {
			log.WriteToStderrf("failed to persist simple_access_scopes to store %v", err)
			return err
		}
	}
	return nil
}

func convertPermissionSetID(permissionSetID string) string {
	identifierSuffix := strings.TrimPrefix(permissionSetID, permissionSetIDPrefix)
	replacement, found := permissionsetIDMapping[identifierSuffix]
	if found {
		return replacement
	}
	_, err := uuid.FromString(identifierSuffix)
	if err != nil {
		generatedID := uuid.NewV4().String()
		permissionsetIDMapping[identifierSuffix] = generatedID
		return generatedID
	}
	return identifierSuffix
}

func migratePermissionSets(ctx context.Context, gormDB *gorm.DB, postgresDB postgres.DB, legacyStore legacypermissionsets.Store) error {
	pgutils.CreateTableFromModel(context.Background(), gormDB, frozenSchema.CreateTablePermissionSetsStmt)

	store := pgPermissionSetStore.New(postgresDB)

	var permissionSets []*storage.PermissionSet
	err := walkPermissionSets(ctx, legacyStore, func(obj *storage.PermissionSet) error {
		permissionSetID := convertPermissionSetID(obj.GetId())
		obj.Id = permissionSetID
		permissionSets = append(permissionSets, obj)
		if len(permissionSets) == batchSize {
			if err := store.UpsertMany(ctx, permissionSets); err != nil {
				log.WriteToStderrf("failed to persist permission_sets to store %v", err)
				return err
			}
			permissionSets = permissionSets[:0]
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(permissionSets) > 0 {
		if err = store.UpsertMany(ctx, permissionSets); err != nil {
			log.WriteToStderrf("failed to persist permission_sets to store %v", err)
			return err
		}
	}
	return nil
}

func getRoleAccessScopeID(role *storage.Role) (string, error) {
	roleAccessScopeID := strings.TrimPrefix(role.GetAccessScopeId(), accessScopeIDPrefix)
	if replacement, found := accessScopeIDMapping[roleAccessScopeID]; found {
		roleAccessScopeID = replacement
	}
	_, accessIDParseErr := uuid.FromString(roleAccessScopeID)
	if accessIDParseErr != nil {
		log.WriteToStderrf("failed to convert role to postgres format, bad access scope ID. Role [%s], error %v", role.GetName(), accessIDParseErr)
		return "", accessIDParseErr
	}
	return roleAccessScopeID, nil
}

func getRolePermissionSetID(role *storage.Role) (string, error) {
	rolePermissionSetID := strings.TrimPrefix(role.GetPermissionSetId(), permissionSetIDPrefix)
	if replacement, found := permissionsetIDMapping[rolePermissionSetID]; found {
		rolePermissionSetID = replacement
	}
	_, permissionSetIDParseErr := uuid.FromString(rolePermissionSetID)
	if permissionSetIDParseErr != nil {
		log.WriteToStderrf("failed to convert role to postgres format, bad permission set ID. Role [%s], error %v", role.GetName(), permissionSetIDParseErr)
		return "", permissionSetIDParseErr
	}
	return rolePermissionSetID, nil
}

func migrateRoles(ctx context.Context, gormDB *gorm.DB, postgresDB postgres.DB, legacyStore legacyroles.Store) error {
	pgutils.CreateTableFromModel(context.Background(), gormDB, frozenSchema.CreateTableRolesStmt)

	store := pgRoleStore.New(postgresDB)

	var roles []*storage.Role
	err := walkRoles(ctx, legacyStore, func(obj *storage.Role) error {
		roleAccessScopeID, accessScopeConversionErr := getRoleAccessScopeID(obj)
		if accessScopeConversionErr != nil {
			return accessScopeConversionErr
		}
		obj.AccessScopeId = roleAccessScopeID
		rolePermissionSetID, permissionSetConversionErr := getRolePermissionSetID(obj)
		if permissionSetConversionErr != nil {
			return permissionSetConversionErr
		}
		obj.PermissionSetId = rolePermissionSetID
		roles = append(roles, obj)
		if len(roles) == batchSize {
			if err := store.UpsertMany(ctx, roles); err != nil {
				log.WriteToStderrf("failed to persist roles to store %v", err)
				return err
			}
			roles = roles[:0]
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(roles) > 0 {
		if err = store.UpsertMany(ctx, roles); err != nil {
			log.WriteToStderrf("failed to persist roles to store %v", err)
			return err
		}
	}
	return nil
}

func getReportConfigurationScopeID(reportConfiguration *storage.ReportConfiguration) (string, error) {
	reportConfigurationScopeID := strings.TrimPrefix(reportConfiguration.GetScopeId(), accessScopeIDPrefix)
	if replacement, found := accessScopeIDMapping[reportConfigurationScopeID]; found {
		reportConfigurationScopeID = replacement
	}
	if _, accessIDParseErr := uuid.FromString(reportConfigurationScopeID); accessIDParseErr != nil {
		log.WriteToStderrf("failed to convert report configuration to postgres format, bad scope ID. "+
			"Report Configuration ID: [%s] Name:[%s], error %v",
			reportConfiguration.GetId(),
			reportConfiguration.GetName(),
			accessIDParseErr)
		return "", accessIDParseErr
	}
	return reportConfigurationScopeID, nil
}

func storeReportsConfigurationBatch(ctx context.Context, store pgReportConfigurationStore.Store, reportConfigs []*storage.ReportConfiguration) {
	err := store.UpsertMany(ctx, reportConfigs)
	if err != nil {
		batchIDs := make([]string, 0, len(reportConfigs))
		for _, reportConfig := range reportConfigs {
			batchIDs = append(batchIDs, reportConfig.GetId())
		}
		log.WriteToStderrf("failed to persist report configurations with IDs [%s] to store %v", strings.Join(batchIDs, " "), err)
	}
}

// This function was added in 3.74 in order to cover an overlooked reference to access scope IDs.
// Users who migrated to postgres with 3.73 may have report configurations relying on incorrect scope IDs.
func migrateReportConfigurationScopeIDs(ctx context.Context, postgresDB postgres.DB) error {
	store := pgReportConfigurationStore.New(postgresDB)
	var reportConfigs []*storage.ReportConfiguration
	err := walkReportConfigurations(ctx, store, func(obj *storage.ReportConfiguration) error {
		reportConfigurationScopeID, conversionErr := getReportConfigurationScopeID(obj)
		if conversionErr != nil {
			// The goal here is to fail open, and to log reports for which the report configuration update failed.
			// The case where the scope ID update failed is logged in the called function.
			return nil
		}
		obj.ScopeId = reportConfigurationScopeID
		reportConfigs = append(reportConfigs, obj)
		if len(reportConfigs) < batchSize {
			return nil
		}
		storeReportsConfigurationBatch(ctx, store, reportConfigs)
		reportConfigs = reportConfigs[:0]
		return nil
	})
	if err != nil {
		log.WriteToStderrf("error while updating scope IDs for report configurations in store %v", err)
	}
	if len(reportConfigs) > 0 {
		storeReportsConfigurationBatch(ctx, store, reportConfigs)
	}
	return nil
}

func walkAccessScopes(ctx context.Context, s legacysimpleaccessscopes.Store, fn func(obj *storage.SimpleAccessScope) error) error {
	return s.Walk(ctx, fn)
}

func walkPermissionSets(ctx context.Context, s legacypermissionsets.Store, fn func(obj *storage.PermissionSet) error) error {
	return s.Walk(ctx, fn)
}

func walkRoles(ctx context.Context, s legacyroles.Store, fn func(obj *storage.Role) error) error {
	return s.Walk(ctx, fn)
}

func walkReportConfigurations(ctx context.Context, s pgReportConfigurationStore.Store, fn func(obj *storage.ReportConfiguration) error) error {
	return s.Walk(ctx, fn)
}

func init() {
	migrations.MustRegisterMigration(migration)
}
