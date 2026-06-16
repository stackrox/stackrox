# Startup Migrations Guide

Startup migrations run during the Central upgrade process, **before** Central starts serving
traffic. They have exclusive database access but directly extend upgrade downtime.

Use startup migrations for data transformations on tables with fewer than ~100k rows.
For larger tables, use [background migrations](../central/backgroundmigrations/BACKGROUND_MIGRATIONS.md).

## Getting started

### 1. Bootstrap the migration

```bash
DESCRIPTION="add_foo_column_to_bar" make bootstrap_migration
```

This generates:

| File | Purpose |
|---|---|
| `migrator/migrations/m_{N}_to_m_{N+1}_{description}/migration.go` | Registration wrapper (do not edit) |
| `migrator/migrations/m_{N}_to_m_{N+1}_{description}/migration_impl.go` | Your migration logic (edit this) |
| `migrator/migrations/m_{N}_to_m_{N+1}_{description}/migration_test.go` | Your tests (edit this) |
| `pkg/migrations/internal/seq_num.go` | `CurrentDBVersionSeqNum` incremented |
| `migrator/runner/all.go` | Blank import added for auto-registration |

### 2. Write the migration

Edit `migration_impl.go`. The generated file contains TODOs guiding you through the implementation.

Your migration function receives a `*types.Databases`
(`github.com/stackrox/rox/migrator/types`):

```go
type Databases struct {
    GormDB     *gorm.DB      // For GORM operations (NOT part of outer transaction)
    PostgresDB postgres.DB   // For raw SQL (participates in outer transaction) — pkg/postgres.DB
    DBCtx      context.Context
}
```

### 3. Write tests

Edit `migration_test.go`. Tests require a running Postgres instance on port 5432:

```bash
docker run --rm --env POSTGRES_USER="$USER" --env POSTGRES_HOST_AUTH_METHOD=trust --publish 5432:5432 docker.io/library/postgres:15
```

Run the test:

```bash
go test -v -tags sql_integration ./migrator/migrations/m_{N}_to_m_{N+1}_*/
```

## Data access patterns

### Raw SQL (preferred)

Raw SQL provides the best isolation from future code changes and participates in the
outer migration transaction. Use it for straightforward operations.

```go
func migrate(database *types.Databases) error {
    ctx := database.DBCtx
    conn, err := database.PostgresDB.Acquire(ctx)
    if err != nil {
        return fmt.Errorf("acquiring connection: %w", err)
    }
    defer conn.Release()

    _, err = conn.Exec(ctx, `
        UPDATE my_table
        SET new_column = other_table.value
        FROM other_table
        WHERE my_table.fk = other_table.id
          AND my_table.new_column IS NULL`)
    return err
}
```

### GORM

Use GORM when you need object-oriented access or when working with schema changes.
GORM operations do **not** participate in the outer transaction.

**Important:** Always narrow your SELECT to specific columns. A `SELECT *` will break if a
subsequent migration modifies the table structure, because GORM caches prepared statements.

```go
// Define a trimmed model with only the columns you need.
type myRow struct {
    ID         string `gorm:"column:id;type:varchar;primaryKey"`
    Serialized []byte `gorm:"column:serialized;type:bytea"`
}

func migrate(database *types.Databases) error {
    db := database.GormDB.WithContext(database.DBCtx).Table("my_table")
    query := database.GormDB.WithContext(database.DBCtx).Table("my_table").Select("id, serialized")

    rows, err := query.Rows()
    if err != nil {
        return err
    }
    defer rows.Close()
    // ... process rows ...
}
```

### Frozen schemas

A migration must **never** import schemas from `pkg/postgres/schema` — those evolve with the
latest release and will break the migration when the schema changes in a future version.

If your migration needs to create or modify a table schema, freeze the schema inside the
migration package:

```bash
./tools/generate-helpers/pg-schema-migration-helper --type=storage.MyType --search-category MY_CATEGORY
```

Copy the output into a `schema/` subdirectory within your migration package. Remove any
conversion functions you don't need.

Then apply the frozen schema:

```go
pgutils.CreateTableFromModel(database.DBCtx, database.GormDB, schema.CreateTableMyTableStmt)
```

## Common scenarios

Startup migrations are for operations on **small, bounded tables** where the execution time
is predictable and short. If you need to backfill data or create indexes on large or
high-cardinality tables, use a
[background migration](../central/backgroundmigrations/BACKGROUND_MIGRATIONS.md) instead.

### Scenario 1: Fix or transform config data on a small table

Update values in a small table (e.g., cluster config, admission controller settings).
Use GORM with a frozen schema when you need to deserialize/re-serialize protobuf blobs.

See `m_211_to_m_212_admission_control_config` for a real example:

```go
func migrate(database *types.Databases) error {
    ctx := sac.WithAllAccess(context.Background())
    db := database.GormDB
    pgutils.CreateTableFromModel(ctx, db, schema.CreateTableClustersStmt)

    return fixAdmissionControllerConfig(ctx, db)
}
```

Key patterns:
- Apply the frozen schema first (`CreateTableFromModel`) to ensure the table has the
  expected structure
- Use GORM's `FindInBatches` or `Rows()` to iterate, with explicit column selection
- Re-serialize the protobuf blob after modifying fields

### Scenario 2: Drop a table or delete rows

Remove deprecated tables or obsolete data. These are simple SQL operations.

```go
func migrate(database *types.Databases) error {
    _, err := database.PostgresDB.Exec(database.DBCtx,
        "DROP TABLE IF EXISTS deprecated_table")
    return err
}
```

Or delete specific rows:

```go
func migrate(database *types.Databases) error {
    _, err := database.PostgresDB.Exec(database.DBCtx,
        "DELETE FROM risks WHERE subject_type = $1",
        storage.RiskSubjectType_IMAGE_COMPONENT)
    return err
}
```

See `m_216_to_m_217_remove_compliance_benchmark_table` and
`m_222_to_m_223_remove_component_risk_records` for real examples.

### Scenario 3: Index type conversion (legacy)

Use the `indexhelper` package for converting index types (e.g., HASH to BTREE).
Note: for **new** index creation, use proto tags (`sql:"index"` or `sql:"background-index"`)
and the code generator instead. This scenario is only for legacy index type conversions.

```go
import "github.com/stackrox/rox/migrator/migrations/indexhelper"

func migrate(database *types.Databases) error {
    return indexhelper.MigrateIndex(
        database.DBCtx,
        database.PostgresDB,
        "my_table",           // table name
        "my_table_col_idx",   // current index name
        "my_column",          // indexed column
        "my_table_col_tmp",   // temporary index name during migration
    )
}
```

The helper creates a new BTREE index with a temporary name, drops the old HASH index,
then renames the temporary index to the original name.

### What does NOT belong in a startup migration

| Operation | Why not | Use instead |
|---|---|---|
| Backfill on high-cardinality table | Can cause hours of downtime (m_212: 8h on largest tenant) | Background migration |
| `CREATE INDEX` on large table | Holds exclusive lock, blocks all writes | Background migration with `CONCURRENTLY` |
| Pure SQL JOIN backfill on unbounded table | Single `UPDATE ... FROM` locks affected rows for the full duration | Background migration with batching |
| Schema-only changes (add column, add table) | Unnecessary — GORM AutoMigrate handles this | No migration needed |

## Writing tests

Tests use `testify/suite` and require the `sql_integration` build tag.

```go
//go:build sql_integration

package m999tom1000

import (
    "context"
    "testing"

    pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
    "github.com/stackrox/rox/migrator/migrations/m_999_to_m_1000_example/schema"
    "github.com/stackrox/rox/migrator/types"
    "github.com/stackrox/rox/pkg/postgres/pgutils"
    "github.com/stackrox/rox/pkg/sac"
    "github.com/stretchr/testify/suite"
)

type migrationTestSuite struct {
    suite.Suite
    db  *pghelper.TestPostgres
    ctx context.Context
}

func TestMigration(t *testing.T) {
    suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupSuite() {
    s.ctx = sac.WithAllAccess(context.Background())
    s.db = pghelper.ForT(s.T(), false)
    // Create table using frozen schema
    pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), schema.CreateTableMyTableStmt)
}

func (s *migrationTestSuite) TearDownSuite() {
    s.db.Teardown(s.T())
}

func (s *migrationTestSuite) TestMigration() {
    // 1. Insert pre-migration data
    s.insertTestData()

    // 2. Run migration
    dbs := &types.Databases{
        GormDB:     s.db.GetGormDB(),
        PostgresDB: s.db.DB,
        DBCtx:      s.ctx,
    }
    s.Require().NoError(migration.Run(dbs))

    // 3. Verify post-migration state
    s.verifyResults()

    // 4. Verify idempotency: running again should be a no-op
    s.Require().NoError(migration.Run(dbs))
    s.verifyResults()
}

func (s *migrationTestSuite) TestBackwardsCompatibility() {
    // Verify that queries from the PREVIOUS release still work
    // against the post-migration schema.
    // Example: if old code does SELECT col1, col2 FROM my_table,
    // ensure those columns still exist and return valid data.
}
```

### What to test

- **Happy path**: data is correctly transformed
- **Already-migrated rows**: running twice produces the same result (idempotency)
- **Edge cases**: NULL values, empty strings, missing foreign keys
- **Backwards compatibility**: old-version SQL queries work against the new schema

## Local testing on a cluster

1. Create a PR with your migration to build a CI image
2. Checkout the commit **before** the migration: `make clean image`
3. Deploy: `export STORAGE=pvc && teardown && ./deploy/k8s/deploy-local.sh`
4. Port-forward: `./scripts/k8s/local-port-forward.sh`
5. Create test data via Central UI or REST API
6. Checkout your migration commit
7. Update Central image: `kubectl -n stackrox set image deploy/central central=stackrox/main:$(make tag)`
8. Check logs for:
   ```
   Migrator: Info: Found DB at version N, which is less than what we expect (N+1). Running migrations...
   Migrator: Info: Successfully updated DB from version N to N+1
   ```
9. Re-run port-forward and verify results

## Breaking changes

When a migration makes a schema change that is incompatible with a previous release:

1. Write the migration normally
2. Update `MinimumSupportedDBVersionSeqNum` in `pkg/migrations/internal/fallback_seq_num.go`
   to the `CurrentDBVersionSeqNum` of the first release that can tolerate the change
3. Update the associated version string

The migrator will reject upgrades from versions below this threshold, preventing users
from upgrading into an incompatible state.

## Checklist

- [ ] Migration is idempotent (safe to re-run)
- [ ] Uses frozen schemas, not `pkg/postgres/schema`
- [ ] GORM queries use explicit column selection (no `SELECT *`)
- [ ] Table size is bounded and small (< 100k rows) — use a background migration otherwise
- [ ] Tests cover happy path, edge cases, and idempotency
- [ ] Backwards compatibility test verifies old queries still work
- [ ] No feature flag dependencies in migration code
- [ ] `CurrentDBVersionSeqNum` is incremented (done by bootstrap tool)
- [ ] Migration is registered in `migrator/runner/all.go` (done by bootstrap tool)
