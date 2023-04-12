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

// RegisteredSchemaTable is an interface to access registered schema information
type RegisteredSchemaTable interface {
	GetSchema() *walker.Schema
	GetCreateStatement() *postgres.CreateStmts
}

type registeredTable struct {
	Schema             *walker.Schema
	CreateStmt         *postgres.CreateStmts
	FeatureEnabledFunc func() bool
}

func (r *registeredTable) GetSchema() *walker.Schema {
	if r == nil {
		return nil
	}
	return r.Schema
}

func (r *registeredTable) GetCreateStatement() *postgres.CreateStmts {
	if r == nil {
		return nil
	}
	return r.CreateStmt
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

// GetAllTables provides the list of registered tables.
func GetAllTables() []*RegisteredSchemaTable {
	tables := make([]*registeredTable, 0, len(registeredTables))
	for _, v := range registeredTables {
		tables = append(tables, v)
	}
	return tables
}

// GetAllRegisteredTablesInOrder provides the list of registered tables and associated schemas.
func GetAllRegisteredTablesInOrder() []RegisteredSchemaTable {
	visited := set.NewStringSet()

	tables := make([]string, 0, len(registeredTables))
	for table := range registeredTables {
		tables = append(tables, table)
	}
	sort.Strings(tables)

	var rts []RegisteredSchemaTable
	for _, table := range tables {
		rts = append(rts, getRegisteredTablesFor(visited, table)...)
	}
	return rts
}

func getRegisteredTablesFor(visited set.StringSet, table string) []RegisteredSchemaTable {
	if visited.Contains(table) {
		return nil
	}
	var rts []RegisteredSchemaTable
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
	for _, rt := range GetAllRegisteredTablesInOrder() {
		// Exclude tests
		if strings.HasPrefix(rt.GetSchema().Table, "test_") {
			continue
		}
		log.Debugf("Applying schema for table %s", rt.GetSchema().Table)
		pgutils.CreateTableFromModel(ctx, gormDB, rt.GetCreateStatement())
	}
}

// ApplyAllSchemasIncludingTests creates or auto migrate according to the current schema including test schemas
func ApplyAllSchemasIncludingTests(ctx context.Context, gormDB *gorm.DB, _ testing.TB) {
	for _, rt := range GetAllRegisteredTablesInOrder() {
		log.Debugf("Applying schema for table %s", rt.GetSchema().Table)
		pgutils.CreateTableFromModel(ctx, gormDB, rt.GetCreateStatement())
	}
}

// ApplySchemaForTable creates or auto migrate according to the current schema
func ApplySchemaForTable(ctx context.Context, gormDB *gorm.DB, table string) {
	rts := getRegisteredTablesFor(set.NewStringSet(), table)
	for _, rt := range rts {
		pgutils.CreateTableFromModel(ctx, gormDB, rt.GetCreateStatement())
	}
}
