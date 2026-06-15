package schema

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/stackrox/rox/pkg/env"
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

// ApplyAllStartupIndexes creates indexes with Background: false before Central serves traffic.
// All indexes use CREATE INDEX CONCURRENTLY. The SQL is pre-generated in CreateSQL at codegen time.
func ApplyAllStartupIndexes(ctx context.Context, db postgres.DB) error {
	return applyIndexes(ctx, db, false, env.PostgresDefaultMigrationStatementTimeout.DurationSetting())
}

// ApplyAllBackgroundIndexes creates indexes with Background: true using CREATE INDEX CONCURRENTLY.
// Called by the background migration runner after Central is serving traffic.
func ApplyAllBackgroundIndexes(ctx context.Context, db postgres.DB) error {
	return applyIndexes(ctx, db, true, env.BackgroundIndexTimeout.DurationSetting())
}

func applyIndexes(ctx context.Context, db postgres.DB, background bool, timeout time.Duration) error {
	existing, err := getExistingIndexes(ctx, db)
	if err != nil {
		return fmt.Errorf("querying existing indexes: %w", err)
	}

	label := "startup"
	if background {
		label = "background"
	}

	var desired []*postgres.IndexDefinition
	for _, idx := range GetAllIndexDefinitions() {
		if idx.Background == background {
			desired = append(desired, idx)
		}
	}

	var toDrop, toCreate []*postgres.IndexDefinition
	for _, idx := range desired {
		if existing.invalid.Contains(idx.Name) {
			toDrop = append(toDrop, idx)
			toCreate = append(toCreate, idx)
		} else if !existing.valid.Contains(idx.Name) {
			toCreate = append(toCreate, idx)
		}
	}

	log.Infof("Reconciling %d %s indexes: %d exist, %d invalid, %d to create",
		len(desired), label, len(desired)-len(toCreate), len(toDrop), len(toCreate))

	for _, idx := range toDrop {
		if err := dropInvalidIndex(ctx, db, idx.Name); err != nil {
			return fmt.Errorf("dropping invalid index %s: %w", idx.Name, err)
		}
	}

	for _, idx := range toCreate {
		log.Infof("Creating %s index: %s", label, idx.Name)
		stmtCtx, cancel := context.WithTimeout(ctx, timeout)
		_, err := db.Exec(stmtCtx, idx.CreateSQL)
		cancel()
		if err != nil {
			return fmt.Errorf("creating %s index %s: %w", label, idx.Name, err)
		}
	}
	return nil
}

// ApplyAllIndexes creates ALL indexes immediately.
// Used by tests where all indexes including background migration indexes are needed right away.
func ApplyAllIndexes(ctx context.Context, db postgres.DB) {
	for _, idx := range GetAllIndexDefinitions() {
		if _, err := db.Exec(ctx, idx.CreateSQL); err != nil {
			log.Errorf("Failed to create index %s: %v", idx.Name, err)
		}
	}
}

// GetAllIndexDefinitions returns all index definitions across all registered tables.
func GetAllIndexDefinitions() []*postgres.IndexDefinition {
	var all []*postgres.IndexDefinition
	for _, rt := range getAllRegisteredTablesInOrder() {
		if strings.HasPrefix(rt.Schema.Table, "test_") {
			continue
		}
		all = append(all, flattenIndexes(rt.CreateStmt)...)
	}
	return all
}

func flattenIndexes(stmt *postgres.CreateStmts) []*postgres.IndexDefinition {
	var result []*postgres.IndexDefinition
	result = append(result, stmt.Indexes...)
	for _, child := range stmt.Children {
		result = append(result, flattenIndexes(child)...)
	}
	return result
}

type indexState struct {
	valid   set.StringSet
	invalid set.StringSet
}

func getExistingIndexes(ctx context.Context, db postgres.DB) (*indexState, error) {
	conn, err := db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	rows, err := conn.Query(ctx,
		"SELECT c.relname, i.indisvalid FROM pg_index i JOIN pg_class c ON c.oid = i.indexrelid")
	if err != nil {
		return nil, fmt.Errorf("querying pg_index: %w", err)
	}
	defer rows.Close()

	state := &indexState{
		valid:   set.NewStringSet(),
		invalid: set.NewStringSet(),
	}
	for rows.Next() {
		var name string
		var valid bool
		if err := rows.Scan(&name, &valid); err != nil {
			return nil, fmt.Errorf("scanning index row: %w", err)
		}
		if valid {
			state.valid.Add(name)
		} else {
			state.invalid.Add(name)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating index rows: %w", err)
	}
	return state, nil
}

func dropInvalidIndex(ctx context.Context, db postgres.DB, name string) error {
	log.Warnf("Dropping invalid index %s (leftover from crashed CONCURRENTLY)", name)
	stmtCtx, cancel := context.WithTimeout(ctx, env.BackgroundIndexTimeout.DurationSetting())
	defer cancel()
	_, err := db.Exec(stmtCtx, "DROP INDEX CONCURRENTLY IF EXISTS "+pq.QuoteIdentifier(name))
	return err
}
