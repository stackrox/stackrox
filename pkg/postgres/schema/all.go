package schema

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/set"
	"gorm.io/gorm"
)

var (
	log = logging.LoggerForModule()
	// registeredTables is map of sql table name to go schema of the sql table.
	registeredTables = make(map[string]*registeredTable)
)

type registeredTable struct {
	Name               string // table name, always set (even before lazy schema construction)
	Schema             *walker.Schema
	CreateStmt         *postgres.CreateStmts
	FeatureEnabledFunc func() bool
	resolve            func() *walker.Schema // lazy schema constructor, set by RegisterTableStmt
}

func (rt *registeredTable) tableName() string {
	if rt.Schema != nil {
		return rt.Schema.Table
	}
	return rt.Name
}

// RegisterTableStmt registers a table's create statement and lazy schema resolver
// at init time. The full walker.Schema is built lazily on first access.
func RegisterTableStmt(tableName string, stmt *postgres.CreateStmts, resolver func() *walker.Schema, featureFlagFuncs ...func() bool) {
	if _, ok := registeredTables[tableName]; ok {
		return
	}
	featureFlagFunc := func() bool { return true }
	if len(featureFlagFuncs) != 0 {
		featureFlagFunc = featureFlagFuncs[0]
	}
	registeredTables[tableName] = &registeredTable{Name: tableName, CreateStmt: stmt, FeatureEnabledFunc: featureFlagFunc, resolve: resolver}
}

// RegisterTable maps a table to an object type for the purposes of metrics gathering.
// Updates the schema on an existing registration (from RegisterTableStmt) or creates a new one.
func RegisterTable(schema *walker.Schema, stmt *postgres.CreateStmts, featureFlagFuncs ...func() bool) {
	featureFlagFunc := func() bool { return true }
	if len(featureFlagFuncs) != 0 {
		featureFlagFunc = featureFlagFuncs[0]
	}
	if rt, ok := registeredTables[schema.Table]; ok {
		// Update existing registration with the full schema
		rt.Schema = schema
		return
	}
	registeredTables[schema.Table] = &registeredTable{Name: schema.Table, Schema: schema, CreateStmt: stmt, FeatureEnabledFunc: featureFlagFunc}
}

// GetSchemaForTable return the schema registered for specified table name.
func GetSchemaForTable(tableName string) *walker.Schema {
	if rt, ok := registeredTables[tableName]; ok {
		return rt.Schema
	}
	return nil
}

func getAllTables() []*registeredTable {
	tables := make([]*registeredTable, 0, len(registeredTables))
	for _, v := range registeredTables {
		tables = append(tables, v)
	}
	return tables
}

func getAllRegisteredTablesInOrder() []*registeredTable {
	// Resolve all lazy schemas so that References are available for
	// dependency ordering. Each resolve() call triggers sync.OnceValue
	// which calls RegisterTable to populate rt.Schema.
	for _, rt := range registeredTables {
		if rt.Schema == nil && rt.resolve != nil {
			rt.resolve()
		}
	}

	visited := set.NewStringSet()

	tables := make([]string, 0, len(registeredTables))
	for table := range registeredTables {
		tables = append(tables, table)
	}
	slices.Sort(tables)

	var rts []*registeredTable
	for _, table := range tables {
		rts = append(rts, getRegisteredTablesFor(visited, table)...)
	}
	return rts
}

func getRegisteredTablesFor(visited set.StringSet, table string) []*registeredTable {
	if visited.Contains(table) {
		return nil
	}
	var rts []*registeredTable
	rt := registeredTables[table]
	if !rt.FeatureEnabledFunc() {
		return nil
	}
	// Schema may be nil if only the create statement was registered at init time
	// (sync.OnceValue defers full schema construction). References are only
	// available after the schema is fully built.
	if rt.Schema != nil {
		for _, ref := range rt.Schema.References {
			rts = append(rts, getRegisteredTablesFor(visited, ref.OtherSchema.Table)...)
		}
	}
	rts = append(rts, rt)
	visited.Add(table)
	return rts
}

// ApplyAllSchemas creates or auto migrate according to the current schema
func ApplyAllSchemas(ctx context.Context, gormDB *gorm.DB) {
	for _, rt := range getAllRegisteredTablesInOrder() {
		tableName := rt.tableName()
		// Exclude tests
		if strings.HasPrefix(tableName, "test_") {
			continue
		}
		log.Debugf("Applying schema for table %s", tableName)
		pgutils.CreateTableFromModel(ctx, gormDB, rt.CreateStmt)
	}
}

// ApplyAllSchemasIncludingTests creates or auto migrate according to the current schema including test schemas
func ApplyAllSchemasIncludingTests(ctx context.Context, gormDB *gorm.DB, _ testing.TB) {
	for _, rt := range getAllRegisteredTablesInOrder() {
		log.Debugf("Applying schema for table %s", rt.tableName())
		pgutils.CreateTableFromModel(ctx, gormDB, rt.CreateStmt)
	}
}

// ApplySchemaForTable creates or auto migrate according to the current schema
func ApplySchemaForTable(ctx context.Context, gormDB *gorm.DB, table string) {
	rts := getRegisteredTablesFor(set.NewStringSet(), table)
	for _, rt := range rts {
		pgutils.CreateTableFromModel(ctx, gormDB, rt.CreateStmt)
	}
}
