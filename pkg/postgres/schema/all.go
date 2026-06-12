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
// Unique indexes use blocking CREATE INDEX. All others use CREATE INDEX CONCURRENTLY.
// The SQL variant is pre-generated in CreateSQL at codegen time.
func ApplyAllStartupIndexes(ctx context.Context, db postgres.DB) error {
	return applyIndexes(ctx, db, false, env.PostgresDefaultMigrationStatementTimeout.DurationSetting())
}

// ApplyAllBackgroundIndexes creates indexes with Background: true using CREATE INDEX CONCURRENTLY.
// Called by the background migration runner after Central is serving traffic.
// Uses raw postgres.DB because CREATE INDEX CONCURRENTLY cannot run inside a transaction.
func ApplyAllBackgroundIndexes(ctx context.Context, db postgres.DB) error {
	return applyIndexes(ctx, db, true, env.BackgroundIndexTimeout.DurationSetting())
}

func applyIndexes(ctx context.Context, db postgres.DB, background bool, timeout time.Duration) error {
	invalidIndexes, err := getInvalidIndexNames(ctx, db)
	if err != nil {
		return fmt.Errorf("checking for invalid indexes: %w", err)
	}

	label := "startup"
	if background {
		label = "background"
	}

	var firstErr error
	for _, idx := range GetAllIndexDefinitions() {
		if idx.Background != background {
			continue
		}

		if invalidIndexes[idx.Name] {
			if err := dropInvalidIndex(ctx, db, idx.Name); err != nil {
				log.Errorf("Failed to drop invalid index %s: %v", idx.Name, err)
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
		}

		log.Infof("Creating %s index: %s", label, idx.Name)
		stmtCtx, cancel := context.WithTimeout(ctx, timeout)
		_, execErr := db.Exec(stmtCtx, idx.CreateSQL)
		cancel()
		if execErr != nil {
			log.Errorf("Failed to create %s index %s: %v", label, idx.Name, execErr)
			if firstErr == nil {
				firstErr = execErr
			}
		}
	}
	return firstErr
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

func getInvalidIndexNames(ctx context.Context, db postgres.DB) (map[string]bool, error) {
	conn, err := db.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	rows, err := conn.Query(ctx,
		"SELECT c.relname FROM pg_index i JOIN pg_class c ON c.oid = i.indexrelid WHERE NOT i.indisvalid")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		result[name] = true
	}
	return result, rows.Err()
}

func dropInvalidIndex(ctx context.Context, db postgres.DB, name string) error {
	log.Warnf("Dropping invalid index %s (leftover from crashed CONCURRENTLY)", name)
	stmtCtx, cancel := context.WithTimeout(ctx, env.BackgroundIndexTimeout.DurationSetting())
	defer cancel()
	_, err := db.Exec(stmtCtx, "DROP INDEX CONCURRENTLY IF EXISTS "+pq.QuoteIdentifier(name))
	return err
}
