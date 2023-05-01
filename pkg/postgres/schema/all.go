package schema

import (
	"context"
	"sort"
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
	Schema             *walker.Schema
	CreateStmt         *postgres.CreateStmts
	FeatureEnabledFunc func() bool
}

// RegisterTable maps a table to an object type for the purposes of metrics gathering
func RegisterTable(schema *walker.Schema, stmt *postgres.CreateStmts, featureFlagFuncs ...func() bool) {
	if _, ok := registeredTables[schema.Table]; ok {
		log.Fatalf("table %q is already registered for %s", schema.Table, schema.Type)
		return
	}
	featureFlagFunc := func() bool { return true }
	if len(featureFlagFuncs) != 0 {
		featureFlagFunc = featureFlagFuncs[0]
	}
	registeredTables[schema.Table] = &registeredTable{Schema: schema, CreateStmt: stmt, FeatureEnabledFunc: featureFlagFunc}
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
	visited := set.NewStringSet()

	tables := make([]string, 0, len(registeredTables))
	for table := range registeredTables {
		tables = append(tables, table)
	}
	sort.Strings(tables)

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
	for _, ref := range rt.Schema.References {
		rts = append(rts, getRegisteredTablesFor(visited, ref.OtherSchema.Table)...)
	}
	rts = append(rts, rt)
	visited.Add(table)
	return rts
}

// ApplyAllSchemas creates or auto migrate according to the current schema
func ApplyAllSchemas(ctx context.Context, gormDB *gorm.DB) {
	for _, rt := range getAllRegisteredTablesInOrder() {
		// Exclude tests
		if strings.HasPrefix(rt.Schema.Table, "test_") {
			continue
		}
		log.Debugf("Applying schema for table %s", rt.Schema.Table)
		pgutils.CreateTableFromModel(ctx, gormDB, rt.CreateStmt)
	}
}

// ApplyAllSchemasIncludingTests creates or auto migrate according to the current schema including test schemas
func ApplyAllSchemasIncludingTests(ctx context.Context, gormDB *gorm.DB, _ testing.TB) {
	for _, rt := range getAllRegisteredTablesInOrder() {
		log.Debugf("Applying schema for table %s", rt.Schema.Table)
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
