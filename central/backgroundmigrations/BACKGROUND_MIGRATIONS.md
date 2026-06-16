# Background Migrations Guide

Background migrations run **after** Central is fully started and serving traffic. They are
designed for operations on high-cardinality tables where the processing time would cause
unacceptable upgrade downtime if run as a startup migration.

For an overview of both migration types and when to choose which, see
[README.md](../../migrator/README.md).

## When to use background migrations

- Backfilling columns on tables with > 100k rows (see the
  [high-cardinality table list](../../migrator/README.md#high-cardinality-tables-always-use-background-migrations))
- Creating ad-hoc indexes on large tables (partial indexes, expression indexes — standard
  indexes are handled by proto tags and the background migration runner's index reconciliation)
- Any data transformation that would take minutes or hours on production data

**Do NOT use background migrations** for:
- Schema-only changes (GORM AutoMigrate handles these)
- Standard index creation (use `sql:"index"` or `sql:"background-index"` proto tags instead)
- Setting column defaults (use a startup migration with `ALTER COLUMN SET DEFAULT`)
- Small table backfills (< 100k rows — use a startup migration instead)

## How it works

```
Central starts and begins serving traffic
        │
        ▼
Rollout checker polls K8s API
  "Are all old-version pods terminated?"
        │
   NO ──┤──► Wait 60s, retry
        │
   YES ─┘
        │
        ▼
Acquire advisory lock (only one Central runs migrations)
        │
        ▼
Read current seq num from background_migration_versions table
        │
        ▼
For each migration from current to target:
  ├── Run migration.Run(ctx, db)
  ├── Update seq num in DB (checkpoint)
  └── Update Prometheus metric
        │
        ▼
Reconcile background indexes (CREATE INDEX CONCURRENTLY, ROX_BACKGROUND_INDEX_TIMEOUT per statement, default 2h)
        │
        ▼
Set bg_migration_complete metric to 1
```

The rollout checker ensures old-version Central pods are fully terminated before
migrations begin. This prevents the old code from writing data in the pre-migration
format while the migration is transforming it.

If the migration fails, the runner retries after 60 seconds. If Central shuts down
mid-migration, the next startup resumes from the last checkpoint.

## Getting started

### 1. Create the migration package

Unlike startup migrations, there is no `make bootstrap_migration` tool for background
migrations. Create the package manually.

Create a new directory under `central/backgroundmigrations/migrations/`. Follow the
naming convention `m_{N}_to_m_{N+1}_{description}/`.

```
central/backgroundmigrations/migrations/m_000_to_m_001_backfill_foo/
└── migration.go
```

### 2. Register the migration

In `migration.go`, register via `init()`:

```go
package m000tom001

import (
    "context"

    "github.com/stackrox/rox/central/backgroundmigrations/migrations"
    "github.com/stackrox/rox/central/backgroundmigrations/types"
    "github.com/stackrox/rox/pkg/logging"
    "github.com/stackrox/rox/pkg/postgres"
)

var log = logging.LoggerForModule()

func init() {
    migrations.MustRegister(types.BackgroundMigration{
        StartingSeqNum:     0,
        VersionAfterSeqNum: 1,
        Description:        "Backfill foo column on process_indicators",
        Run:                run,
    })
}

func run(ctx context.Context, db postgres.DB) error {
    // Your migration logic here
    return nil
}
```

### 3. Import in runner/all.go

Add a blank import in `central/backgroundmigrations/runner/all.go`:

```go
import (
    _ "github.com/stackrox/rox/central/backgroundmigrations/migrations/m_000_to_m_001_backfill_foo"
)
```

### 4. Bump the sequence number

Check the current value in `central/backgroundmigrations/seq_num.go` and increment it.
Your migration's `StartingSeqNum` must equal the current value, and
`VersionAfterSeqNum` must be `StartingSeqNum + 1`.

```go
const CurrentBgMigrationSeqNum = 1  // was 0
```

## The contract

Your `Run` function must satisfy these requirements:

1. **Idempotent**: Safe to re-run after a crash, rollback, or manual override. Previously
   migrated rows must produce the same result when processed again.

2. **Context-aware**: Check `ctx.Done()` between units of work (e.g., between batches) for
   graceful shutdown. When cancelled, return `ctx.Err()`.

3. **Conflict-free with Central**: Central is live and processing events. Your migration
   must not conflict with concurrent reads and writes. See
   [Concurrency with Central](#concurrency-with-central) below.

4. **Resumable**: Use WHERE clauses that filter out already-processed rows so that restarts
   don't repeat work unnecessarily.

## Concurrency with Central

This is the most critical aspect of background migrations. Central is actively receiving
sensor data, API calls, and processing events while your migration runs.

### Design principle: partition work between migration and application

The safest pattern is to ensure the migration and Central operate on disjoint sets of rows:

- **Migration processes**: Existing rows where the new column is NULL (not yet backfilled)
- **Central processes**: New rows arriving after the migration started (application code
  already populates the new column)

This means:
1. The application code (datastore, store) must be updated to populate the new column on
   INSERT and UPDATE **before** the background migration ships.
2. The migration's WHERE clause filters to `new_column IS NULL`, so it only touches rows
   that pre-date the application code change.

### When disjoint partitioning isn't possible

If the migration must update rows that Central also writes to, use row-level locking:

```go
rows, err := conn.Query(ctx, `
    SELECT id, serialized FROM my_table
    WHERE id > $1 AND new_column IS NULL
    ORDER BY id
    LIMIT $2
    FOR UPDATE SKIP LOCKED`, lastID, batchSize)
```

`FOR UPDATE SKIP LOCKED` acquires row locks but skips rows currently locked by Central,
preventing deadlocks while ensuring forward progress.

### What the application code must handle

While the migration is in progress, some rows have the new column populated and some don't.
The application code must tolerate both states:

```go
// In your datastore/query code, handle the zero value gracefully:
if row.NewColumn != "" {
    // Use the backfilled value
} else {
    // Fall back to parsing from serialized blob or using a default
}
```

Once the migration completes (check via `background_migration_complete` Prometheus metric
or by verifying all rows are populated), the fallback code can be removed in a subsequent
release.

## Example 1: Backfill a column from a serialized proto

This example adds a `signal_hostname` column to `process_indicators` by extracting it from
the serialized protobuf blob. The table has millions of rows in production.

### Prerequisites

1. The proto field already exists in `storage.ProcessIndicator`
2. The column is already added to the GORM schema (GORM AutoMigrate creates it on startup)
3. The application datastore code already populates `signal_hostname` on new inserts

### Migration implementation

```go
package m000tom001

import (
    "context"
    "fmt"

    "github.com/jackc/pgx/v5"
    "github.com/stackrox/rox/central/backgroundmigrations/migrations"
    "github.com/stackrox/rox/central/backgroundmigrations/types"
    "github.com/stackrox/rox/generated/storage"
    "github.com/stackrox/rox/pkg/logging"
    "github.com/stackrox/rox/pkg/postgres"
)

var (
    log       = logging.LoggerForModule()
    batchSize = 5000
)

func init() {
    migrations.MustRegister(types.BackgroundMigration{
        StartingSeqNum:     0,
        VersionAfterSeqNum: 1,
        Description:        "Backfill signal_hostname on process_indicators from serialized proto",
        Run:                run,
    })
}

func run(ctx context.Context, db postgres.DB) error {
    conn, err := db.Acquire(ctx)
    if err != nil {
        return fmt.Errorf("acquiring connection: %w", err)
    }
    defer conn.Release()

    totalUpdated := 0
    lastID := ""

    for {
        // Check for graceful shutdown between batches.
        if ctx.Err() != nil {
            log.Infof("shutdown requested after %d rows, will resume on next start", totalUpdated)
            return ctx.Err()
        }

        updated, newLastID, err := migrateBatch(ctx, conn, lastID)
        if err != nil {
            return fmt.Errorf("batch after id %s: %w", lastID, err)
        }
        if newLastID == "" {
            break
        }

        lastID = newLastID
        totalUpdated += updated

        if totalUpdated%50000 == 0 {
            log.Infof("progress: %d rows backfilled", totalUpdated)
        }
    }

    log.Infof("backfill complete: %d rows updated", totalUpdated)
    return nil
}

func migrateBatch(ctx context.Context, conn *postgres.Conn, lastID string) (int, string, error) {
    // Only select rows where the new column hasn't been populated yet.
    // This makes the migration:
    //   - Idempotent (re-runs skip already-processed rows)
    //   - Conflict-free (new rows inserted by Central already have this column set)
    //   - Resumable (restarts continue from where they left off)
    rows, err := conn.Query(ctx, `
        SELECT id, serialized
        FROM process_indicators
        WHERE id > $1 AND signal_hostname IS NULL
        ORDER BY id
        LIMIT $2`, lastID, batchSize)
    if err != nil {
        return 0, "", fmt.Errorf("querying batch: %w", err)
    }
    defer rows.Close()

    batch := &pgx.Batch{}
    var rowLastID string
    count := 0

    for rows.Next() {
        var id string
        var serialized []byte
        if err := rows.Scan(&id, &serialized); err != nil {
            return 0, "", fmt.Errorf("scanning row: %w", err)
        }

        indicator := &storage.ProcessIndicator{}
        if err := indicator.UnmarshalVT(serialized); err != nil {
            log.Warnf("skipping row %s: unmarshal failed: %v", id, err)
            rowLastID = id
            count++
            continue
        }

        hostname := indicator.GetSignal().GetHostname()
        batch.Queue(
            "UPDATE process_indicators SET signal_hostname = $1 WHERE id = $2 AND signal_hostname IS NULL",
            hostname, id,
        )

        rowLastID = id
        count++
    }

    if err := rows.Err(); err != nil {
        return 0, "", fmt.Errorf("iterating rows: %w", err)
    }

    if batch.Len() == 0 {
        return count, rowLastID, nil
    }

    // Execute all updates in one round-trip.
    results := conn.SendBatch(ctx, batch)
    for i := 0; i < batch.Len(); i++ {
        if _, err := results.Exec(); err != nil {
            _ = results.Close()
            return 0, "", fmt.Errorf("executing update %d: %w", i, err)
        }
    }
    if err := results.Close(); err != nil {
        return 0, "", fmt.Errorf("closing batch: %w", err)
    }

    return count, rowLastID, nil
}
```

### Key design decisions

| Decision | Rationale |
|---|---|
| `WHERE signal_hostname IS NULL` | Only processes rows that haven't been backfilled. New rows from Central already have this set. Makes the migration idempotent and conflict-free. |
| `WHERE ... AND signal_hostname IS NULL` in UPDATE | Double-check in the UPDATE prevents overwriting a value that Central set between the SELECT and UPDATE. |
| `ctx.Err()` check per batch | Allows graceful shutdown. The next startup resumes from the last processed ID. |
| `pgx.Batch` for writes | Sends all updates in a single round-trip, reducing network overhead. |
| Keyset pagination (`id > $1 ORDER BY id`) | Consistent forward progress. No offset drift from concurrent inserts/deletes. |
| `log.Warnf` on unmarshal failure | Don't fail the entire migration for a single corrupt row. Log and continue. |

## Example 2: Batched SQL JOIN backfill

When the value you need exists in another table (no deserialization required), but the
target table is too large for a single `UPDATE ... FROM` (which locks all affected rows
for the full statement duration).

```go
func run(ctx context.Context, db postgres.DB) error {
    conn, err := db.Acquire(ctx)
    if err != nil {
        return fmt.Errorf("acquiring connection: %w", err)
    }
    defer conn.Release()

    totalUpdated := 0
    lastID := ""

    for {
        if ctx.Err() != nil {
            log.Infof("shutdown requested after %d rows, will resume on next start", totalUpdated)
            return ctx.Err()
        }

        updated, newLastID, err := backfillBatch(ctx, conn, lastID)
        if err != nil {
            return fmt.Errorf("batch after id %s: %w", lastID, err)
        }
        if newLastID == "" {
            break
        }

        lastID = newLastID
        totalUpdated += updated
    }

    log.Infof("backfill complete: %d rows updated", totalUpdated)
    return nil
}

func backfillBatch(ctx context.Context, conn *postgres.Conn, lastID string) (int, string, error) {
    // Batched UPDATE ... FROM with LIMIT via a CTE.
    // Only touches rows where the column hasn't been set yet (idempotent, conflict-free).
    tag, err := conn.Exec(ctx, `
        WITH batch AS (
            SELECT a.id
            FROM alerts a
            WHERE a.id > $1 AND a.deployment_type IS NULL
            ORDER BY a.id
            LIMIT $2
        )
        UPDATE alerts a
        SET deployment_type = d.type
        FROM batch
        JOIN deployments d ON a.deployment_id = d.id
        WHERE a.id = batch.id`, lastID, batchSize)
    if err != nil {
        return 0, "", fmt.Errorf("updating batch: %w", err)
    }

    if tag.RowsAffected() == 0 {
        return 0, "", nil
    }

    // Get the last ID processed for keyset pagination.
    var newLastID string
    err = conn.QueryRow(ctx, `
        SELECT id FROM alerts
        WHERE id > $1 AND deployment_type IS NOT NULL
        ORDER BY id DESC
        LIMIT 1`, lastID).Scan(&newLastID)
    if err != nil {
        return 0, "", fmt.Errorf("getting last id: %w", err)
    }

    return int(tag.RowsAffected()), newLastID, nil
}
```

This avoids holding locks on the entire table — each batch locks only its subset of rows
and commits before the next batch begins.

## Example 3: Create an index concurrently (one-off)

> **Note:** Standard indexes should be added via proto tags (`sql:"index=btree"` or
> `sql:"background-index=btree"`) and the code generator. The background migration runner
> handles `CREATE INDEX CONCURRENTLY` automatically for `background-index` fields after all
> numbered migrations complete. This example is only for ad-hoc indexes not covered by the
> generator (e.g., partial indexes, expression indexes, or indexes on hand-written schemas).

Creating an index on a large table with a regular `CREATE INDEX` acquires an exclusive lock,
blocking all writes for the duration. `CREATE INDEX CONCURRENTLY` avoids this by building the
index without holding an exclusive lock.

```go
package m001tom002

import (
    "context"
    "fmt"

    "github.com/stackrox/rox/central/backgroundmigrations/migrations"
    "github.com/stackrox/rox/central/backgroundmigrations/types"
    "github.com/stackrox/rox/pkg/logging"
    "github.com/stackrox/rox/pkg/postgres"
)

var log = logging.LoggerForModule()

func init() {
    migrations.MustRegister(types.BackgroundMigration{
        StartingSeqNum:     1,
        VersionAfterSeqNum: 2,
        Description:        "Create index on process_indicators.signal_hostname",
        Run:                run,
    })
}

func run(ctx context.Context, db postgres.DB) error {
    conn, err := db.Acquire(ctx)
    if err != nil {
        return fmt.Errorf("acquiring connection: %w", err)
    }
    defer conn.Release()

    // CREATE INDEX CONCURRENTLY:
    // - Does NOT hold an exclusive lock (Central continues reading/writing normally)
    // - Cannot run inside a transaction (Postgres requirement)
    // - IF NOT EXISTS makes it idempotent (safe to re-run after failure or rollback)
    //
    // If a previous attempt failed partway, Postgres may leave an INVALID index behind.
    // The IF NOT EXISTS clause will see the invalid index and skip creation, which is wrong.
    // So we first drop any invalid index with the same name, then create it.
    log.Info("dropping any invalid index from a previous failed attempt")
    _, err = conn.Exec(ctx, `
        DROP INDEX CONCURRENTLY IF EXISTS processindicators_signal_hostname`)
    if err != nil {
        return fmt.Errorf("dropping invalid index: %w", err)
    }

    log.Info("creating index processindicators_signal_hostname concurrently")
    _, err = conn.Exec(ctx, `
        CREATE INDEX CONCURRENTLY IF NOT EXISTS processindicators_signal_hostname
        ON process_indicators USING btree (signal_hostname)`)
    if err != nil {
        return fmt.Errorf("creating index: %w", err)
    }

    log.Info("index creation complete")
    return nil
}
```

### Key points

- **`CREATE INDEX CONCURRENTLY`** builds the index without blocking writes. It takes longer
  than a regular `CREATE INDEX` but doesn't cause downtime.
- **Cannot run inside a transaction.** The background migration runner does not wrap
  migrations in transactions, so this works out of the box.
- **Handles failed previous attempts.** A failed `CREATE INDEX CONCURRENTLY` leaves an
  INVALID index. We drop it first, then recreate. The `IF NOT EXISTS` on the CREATE handles
  the case where the index already exists and is valid.
- **No `ctx.Done()` check needed.** Index creation is a single Postgres statement. If the
  context is cancelled, Postgres cancels the statement. On the next run, the DROP + CREATE
  pattern handles cleanup.

## Operations

### Skipping a migration

To skip specific background migrations (e.g., during an emergency), set the environment
variable on the Central deployment:

```bash
ROX_SKIP_BACKGROUND_MIGRATIONS=0,1  # comma-separated seq nums to skip
```

The runner will log that the migration was skipped and advance the seq num past it.
Remove the env var after the situation is resolved.

### Forcing a re-run (override)

To force background migrations to re-run from a specific sequence number:

```bash
ROX_BACKGROUND_MIGRATION_OVERRIDE_SEQ_NUM=0     # start from this seq num
ROX_BACKGROUND_MIGRATION_OVERRIDE_TAG=rerun-v1  # unique tag for this override
```

Both env vars must be set. The tag is persisted to the DB to prevent re-applying the same
override on subsequent pod restarts. To trigger another re-run, change the tag value.

This is useful when:
- A migration completed but produced incorrect results that were fixed in a code update
- A rollback occurred and you need to re-run migrations from a specific point
- Debugging a migration in a test environment

### Monitoring

Two Prometheus metrics track background migration progress:

| Metric | Description |
|---|---|
| `rox_central_background_migration_seq_num` | Current completed seq num |
| `rox_central_background_migration_complete` | `1` when all migrations are done, `0` otherwise |

Use the `_complete` metric to gate features or alerts that depend on migration completion.

### Rollback behavior

When Central is rolled back to a previous version:

1. The old binary has a lower `CurrentBgMigrationSeqNum`
2. The runner detects `dbSeqNum > targetSeqNum` (rollback)
3. The runner resets the DB seq num to the target
4. On the next roll-forward, migrations re-run from the reset point

This is why idempotency is critical — migrations will be re-executed after rollback + upgrade.

## Feature flag

Background migrations are gated by the feature flag `ROX_BACKGROUND_MIGRATION` (enabled by
default), defined in `pkg/features/list.go`. When disabled, no background migrations run.
The flag is checked in `central/main.go` before starting the runner.

## Testing

Background migrations should be tested with the same patterns as startup migrations:

```go
//go:build sql_integration

package m000tom001

import (
    "context"
    "fmt"
    "testing"
    "time"

    pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
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
    // Create table schema
    s.Require().NoError(s.createSchema())
}

func (s *migrationTestSuite) TearDownSuite() {
    s.db.Teardown(s.T())
}

func (s *migrationTestSuite) TestBackfillNewRows() {
    // Insert rows WITHOUT the new column populated (pre-migration state)
    s.insertRow("id-1", "hostname-1", nil)  // signal_hostname is NULL
    s.insertRow("id-2", "hostname-2", nil)

    // Run migration
    s.Require().NoError(run(s.ctx, s.db.DB))

    // Verify backfill
    s.Equal("hostname-1", s.getHostname("id-1"))
    s.Equal("hostname-2", s.getHostname("id-2"))
}

func (s *migrationTestSuite) TestIdempotency() {
    s.insertRow("id-3", "hostname-3", nil)

    // Run twice
    s.Require().NoError(run(s.ctx, s.db.DB))
    s.Require().NoError(run(s.ctx, s.db.DB))

    s.Equal("hostname-3", s.getHostname("id-3"))
}

func (s *migrationTestSuite) TestSkipsAlreadyPopulatedRows() {
    // Simulate a row that Central already populated
    existing := "already-set"
    s.insertRow("id-4", "hostname-4", &existing)

    s.Require().NoError(run(s.ctx, s.db.DB))

    // Should not be overwritten
    s.Equal("already-set", s.getHostname("id-4"))
}

func (s *migrationTestSuite) TestGracefulShutdown() {
    // Insert enough rows to span multiple batches
    for i := 0; i < 100; i++ {
        s.insertRow(fmt.Sprintf("id-%03d", i), fmt.Sprintf("host-%d", i), nil)
    }

    // Run with a context that cancels after a short time
    ctx, cancel := context.WithTimeout(s.ctx, 1*time.Millisecond)
    defer cancel()

    batchSize = 10
    err := run(ctx, s.db.DB)
    // Should return context error, not a migration error
    s.ErrorIs(err, context.DeadlineExceeded)

    // Run again with full context — should complete remaining rows
    s.Require().NoError(run(s.ctx, s.db.DB))

    // All rows should be populated
    for i := 0; i < 100; i++ {
        s.NotEmpty(s.getHostname(fmt.Sprintf("id-%03d", i)))
    }
}
```

### What to test

- **Backfill correctness**: new column gets the right value from the serialized blob
- **Idempotency**: running twice produces the same result
- **Conflict-free with existing data**: rows already populated by Central are not overwritten
- **Graceful shutdown**: cancellation mid-migration doesn't corrupt data; resumption completes
- **Batch boundaries**: test with small batch sizes to exercise pagination logic

## Checklist

- [ ] Migration is idempotent (safe to re-run after rollback or crash)
- [ ] `ctx.Done()` is checked between batches for graceful shutdown
- [ ] WHERE clause filters already-processed rows (`new_column IS NULL`)
- [ ] UPDATE statement includes a guard condition to avoid overwriting concurrent Central writes
- [ ] Application code already populates the new column on INSERT/UPDATE
- [ ] Application code tolerates partial migration state (some rows backfilled, some not)
- [ ] Registered in `central/backgroundmigrations/runner/all.go`
- [ ] `CurrentBgMigrationSeqNum` incremented in `central/backgroundmigrations/seq_num.go`
- [ ] Tests cover correctness, idempotency, existing data, and graceful shutdown
- [ ] No feature flag dependencies in migration code
