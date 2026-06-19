# Startup Migrations Guide

Startup migrations run during the Central upgrade process, **before** a new Central version
starts serving traffic. With the Recreate rollout strategy they directly extend upgrade
downtime; with RollingUpdate they extend the startup time of the new pod while the old pod
continues serving traffic. They should be limited to **DDL/schema changes** that GORM
AutoMigrate cannot handle.

**Do not use startup migrations for data backfills.** 
Central supports both Recreate and RollingUpdate rollout strategies.
With RollingUpdate, old Central pods continue running during the startup
migration window, so any data backfill has the chance of being inconsistent, since old pods keep writing rows without populating the new column.
Use a [background migration](../central/backgroundmigrations/BACKGROUND_MIGRATIONS.md) for all data backfills; background migrations 
wait for the rollout to fully complete (all old pods terminated) before starting.

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
    GormDB     *gorm.DB      // For GORM operations
    PostgresDB postgres.DB   // For raw SQL — pkg/postgres.DB
    DBCtx      context.Context
}
```

**There is no outer transaction wrapping the migration.** 
The runner calls `migration.Run()` without a transaction; only the version update after each
migration is wrapped in a tx. If your migration needs transactional guarantees, open your own transaction.
For batch operations, use a transaction per batch to keep WAL size bounded and avoid losing all progress on error.

### 3. Write tests

Edit `migration_test.go`. Tests require a running Postgres instance on port 5432:

```bash
docker run --rm --env POSTGRES_USER="$USER" --env POSTGRES_HOST_AUTH_METHOD=trust --publish 5432:5432 docker.io/library/postgres:15
```

Run the test:

```bash
go test -v -tags sql_integration ./migrator/migrations/m_{N}_to_m_{N+1}_*/
```

## Writing DDL migrations

Use raw SQL for DDL statements. It provides the best isolation from future code changes.

```go
func migrate(database *types.Databases) error {
    ctx := database.DBCtx
    conn, err := database.PostgresDB.Acquire(ctx)
    if err != nil {
        return fmt.Errorf("acquiring connection: %w", err)
    }
    defer conn.Release()

    _, err = conn.Exec(ctx, `ALTER TABLE my_table ALTER COLUMN status SET DEFAULT 'active'`)
    return err
}
```

**Important:** Never import schemas from `pkg/postgres/schema` — those evolve with the
latest release and will break the migration when the schema changes in a future version.
If your migration needs to ensure a table/column exists before running DDL, use a frozen schema with `pgutils.CreateTableFromModel`.

## Common scenarios

Startup migrations are for **DDL/schema changes** that GORM AutoMigrate cannot handle.
For all data backfills and transformations, use a [background migration](../central/backgroundmigrations/BACKGROUND_MIGRATIONS.md) instead.

### Scenario 1: Set a column default

Set a DB-level default for new rows. PostgreSQL optimizes `SET DEFAULT` as a metadata-only
operation — no table scan.

```go
func migrate(database *types.Databases) error {
    _, err := database.PostgresDB.Exec(database.DBCtx,
        "ALTER TABLE my_table ALTER COLUMN status SET DEFAULT 'active'")
    return err
}
```

### Scenario 2: Drop a deprecated table

Remove a table that is no longer referenced by any supported version. Only do this after
the backwards compatibility window has passed (see [Breaking changes](#breaking-changes)).

```go
func migrate(database *types.Databases) error {
    _, err := database.PostgresDB.Exec(database.DBCtx,
        "DROP TABLE IF EXISTS deprecated_table")
    return err
}
```

### What does NOT belong in a startup migration

| Operation | Why not | Use instead |
|---|---|---|
| Data backfills (any table size) | With RollingUpdate, old pods keep writing rows without the new column — backfill is immediately inconsistent | Background migration |
| Serialized blob → column extraction | Old pods keep writing blobs without updating the extracted column | Background migration |
| Index type conversions | Can be done concurrently without downtime | Background migration or proto tags |
| `CREATE INDEX` on any table | Holds exclusive lock, blocks all writes | Proto tags (`sql:"index"` / `sql:"background-index"`) |
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

- [ ] Migration is DDL-only (no data backfills — use a background migration for those)
- [ ] DDL statements are idempotent (`IF EXISTS`, `IF NOT EXISTS`)
- [ ] Does not import `pkg/postgres/schema`
- [ ] Tests cover happy path, edge cases, and idempotency
- [ ] Backwards compatibility test verifies old queries still work
- [ ] No feature flag dependencies in migration code
- [ ] `CurrentDBVersionSeqNum` is incremented (done by bootstrap tool)
- [ ] Migration is registered in `migrator/runner/all.go` (done by bootstrap tool)
