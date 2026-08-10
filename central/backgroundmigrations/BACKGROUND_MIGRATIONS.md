# Background Migrations Guide

Background migrations run **after** Central is fully started and serving traffic.
They are the **only** path for all data backfills and transformations. They wait for the
rollout to fully complete (all old-version pods terminated) before starting, which is the
primary mechanism that prevents conflicts with old Central code. Use a startup migration
only for DDL/schema changes that GORM AutoMigrate cannot handle.

For an overview of both migration types and the decision flowchart, see
[README.md](../../migrator/README.md).

## When to use background migrations

- All data backfills and transformations, regardless of table size. Central must tolerate partially migrated state (some rows backfilled, some not) while the migration runs
- Creating indexes not covered by the code generator (e.g., partial indexes, expression
  indexes). Standard non-unique indexes are handled by proto tags (`sql:"index"`) and
  the background migration runner's index reconciliation. Unique indexes are created at
  startup via GORM — see [Index Management](../../migrator/README.md#index-management).

**Do NOT use background migrations** for:
- Schema-only changes (GORM AutoMigrate handles these)
- Standard index creation (use `sql:"index"` proto tags instead).
  Unique indexes are always created at startup via GORM.
- Setting static column defaults, use a startup migration with `ALTER COLUMN SET DEFAULT` because PSQL optimizes for that use case

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
  └── Update seq num in DB (checkpoint)
        │
        ▼
Reconcile indexes (CREATE INDEX CONCURRENTLY)
```

The rollout checker ensures old-version Central pods are fully terminated before
migrations begin. This prevents the old code from writing data in the pre-migration
format while the migration is transforming it.

If the migration fails, the runner retries after 60 seconds. If Central shuts down
mid-migration, the next startup resumes from the last checkpoint.

## Getting started

### 1. Create the migration package

**The part of creating the migration manually is going to be replaced with [ROX-35101](https://redhat.atlassian.net/browse/ROX-35101)**

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
   migrated rows must be processed again on re-run and must produce a consistent result with the serialized field.
2. **Context-aware**: Check `ctx.Done()` between units of work (e.g., between batches) for
   graceful shutdown. When cancelled, return `ctx.Err()`.
3. **Conflict-free with Central**: Central is live and processing events. Your migration
   must not conflict with concurrent reads and writes. See
   [Concurrency with Central](#concurrency-with-central) below.
4. **Efficient on re-run**: On crash, rollback, or retry the migration starts from the
   beginning. Iterate by primary key and only write rows that actually need a change --
   already-migrated rows are read but not written, making re-runs fast. Avoid filtering
   by `WHERE new_column IS NULL` as the sole pagination mechanism unless there is an
   index on that column; without an index every batch scan is expensive.

## Concurrency with Central

Background migrations wait for the rollout to fully complete before starting — at that
point, only new Central code is running. However, the new Central is actively receiving
sensor data, API calls, and processing events while the migration runs. You cannot
control what events arrive, so the migration must account for concurrent writes from
the new Central.

### Use `SELECT ... FOR UPDATE SKIP LOCKED`

The standard pattern is to iterate by primary key in batches, locking rows for update
while skipping any rows currently locked by Central. See
[Example 1](#example-1-backfill-a-column-from-a-serialized-proto) for a complete
implementation.

```go
rows, err := conn.Query(ctx, `
    SELECT id, serialized FROM my_table
    WHERE id > $1
    ORDER BY id
    LIMIT $2
    FOR UPDATE SKIP LOCKED`, lastID, batchSize)
```

- **`FOR UPDATE`** acquires row-level locks, preventing Central from modifying those
  rows while the migration processes them.
- **`SKIP LOCKED`** skips rows currently locked by the new Central rather than blocking.
  Since the rollout is complete at this point, any locked row is being written by the new
  Central version which already populates the new column consistently.
- **Primary key iteration** (`id > $1 ORDER BY id`) uses the primary key index for
  efficient pagination. The full table scan happens once; writes are far more expensive
  than the read pass, so this is less of a bottleneck.

Within each batch, only update rows that actually need a change (e.g., where the target
column is still NULL or differs from the expected value). This avoids unnecessary writes
and accelerates migration progress on re-runs.

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

The application code must also populate the new column on INSERT and UPDATE so that
new rows written by Central are already in the post-migration format.

When using stores this is handled by the code generation automatically.

## Example 1: Backfill a column from a serialized proto

This example adds a `signal_hostname` column to `process_indicators` by extracting it from
the serialized protobuf blob. The table has millions of rows in production.

### Prerequisites

1. The proto field already exists in `storage.ProcessIndicator`
2. The column is already added to the GORM schema (GORM AutoMigrate creates it on startup)
3. The application datastore code already populates `signal_hostname` on new inserts and updates

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

type indicatorRow struct {
    id               string
    serialized       []byte
    signalHostname   *string
}

func run(ctx context.Context, db postgres.DB) error {
    totalUpdated := 0
    lastID := ""

    for {
        if ctx.Err() != nil {
            log.Infof("shutdown requested after %d rows, will restart on next start", totalUpdated)
            return ctx.Err()
        }

        rowsUpdated, newLastID, err := migrateBatch(ctx, db, lastID)
        if err != nil {
            return fmt.Errorf("batch after id %s: %w", lastID, err)
        }
        if newLastID == "" {
            break
        }

        lastID = newLastID
        totalUpdated += rowsUpdated
    }

    log.Infof("backfill complete: %d rows updated", totalUpdated)
    return nil
}

// migrateBatch runs one batch inside an explicit transaction so that the
// FOR UPDATE SKIP LOCKED row locks are held through the UPDATE statements.
func migrateBatch(ctx context.Context, db postgres.DB, lastID string) (int, string, error) {
    conn, err := db.Acquire(ctx)
    if err != nil {
        return 0, "", fmt.Errorf("acquiring connection: %w", err)
    }
    defer conn.Release()

    tx, err := conn.Begin(ctx)
    if err != nil {
        return 0, "", fmt.Errorf("begin tx: %w", err)
    }
    defer func() { _ = tx.Rollback(ctx) }()

    rows, err := fetchBatch(ctx, tx, lastID)
    if err != nil {
        return 0, "", err
    }
    if len(rows) == 0 {
        return 0, "", nil
    }

    batch, err := buildBatchUpdates(rows)
    if err != nil {
        return 0, "", err
    }

    if batch.Len() > 0 {
        if err := sendBatch(ctx, tx, batch); err != nil {
            return 0, "", err
        }
    }

    if err := tx.Commit(ctx); err != nil {
        return 0, "", fmt.Errorf("commit: %w", err)
    }

    return batch.Len(), rows[len(rows)-1].id, nil
}

func fetchBatch(ctx context.Context, tx pgx.Tx, lastID string) ([]indicatorRow, error) {
    rows, err := tx.Query(ctx, `
        SELECT id, serialized, signal_hostname
        FROM process_indicators
        WHERE id > $1
        ORDER BY id
        LIMIT $2
        FOR UPDATE SKIP LOCKED`, lastID, batchSize)
    if err != nil {
        return nil, fmt.Errorf("querying batch: %w", err)
    }
    defer rows.Close()

    result := make([]indicatorRow, 0, batchSize)
    for rows.Next() {
        var r indicatorRow
        if err := rows.Scan(&r.id, &r.serialized, &r.signalHostname); err != nil {
            return nil, fmt.Errorf("scanning row: %w", err)
        }
        result = append(result, r)
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("iterating rows: %w", err)
    }
    return result, nil
}

func buildBatchUpdates(rows []indicatorRow) (*pgx.Batch, error) {
    batch := &pgx.Batch{}
    for _, r := range rows {
        indicator := &storage.ProcessIndicator{}
        if err := indicator.UnmarshalVT(r.serialized); err != nil {
            return nil, fmt.Errorf("unmarshal row %s: %w", r.id, err)
        }

        hostname := indicator.GetSignal().GetHostname()

        // Only write if the column value differs from what the serialized
        // blob says it should be. The serialized blob is the source of truth
        // and may have changed (e.g. after a rollback), so we cannot skip
        // rows just because the column is non-NULL.
        if r.signalHostname != nil && *r.signalHostname == hostname {
            continue
        }

        batch.Queue(
            "UPDATE process_indicators SET signal_hostname = $1 WHERE id = $2",
            hostname, r.id,
        )
    }
    return batch, nil
}

func sendBatch(ctx context.Context, tx pgx.Tx, batch *pgx.Batch) error {
    results := tx.SendBatch(ctx, batch)
    for i := 0; i < batch.Len(); i++ {
        if _, err := results.Exec(); err != nil {
            _ = results.Close()
            return fmt.Errorf("batch exec statement %d: %w", i, err)
        }
    }
    return results.Close()
}
```

### Key design decisions

| Decision | Rationale |
|---|---|
| Explicit transaction per batch | Ensures `FOR UPDATE` locks are held through the UPDATE statements. |
| `FOR UPDATE SKIP LOCKED` | Locks rows to prevent conflicts with Central. Skips rows currently locked by Central. |
| Primary key iteration (`id > $1 ORDER BY id`) | Uses the primary key index for efficient pagination. |
| Skip rows where column already matches serialized blob | Avoids unnecessary writes. |
| `ctx.Err()` check per batch | Allows graceful shutdown. |
| `pgx.Batch` for writes | Sends all updates in a single round-trip, reducing network overhead. |

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
    // Iterate by primary key, join to get the value, only update rows that need it.
    // FOR UPDATE SKIP LOCKED on the CTE prevents conflicts with Central.
    tag, err := conn.Exec(ctx, `
        WITH batch AS (
            SELECT a.id
            FROM alerts a
            WHERE a.id > $1
            ORDER BY a.id
            LIMIT $2
            FOR UPDATE SKIP LOCKED
        )
        UPDATE alerts a
        SET deployment_type = d.type
        FROM batch
        JOIN deployments d ON a.deployment_id = d.id
        WHERE a.id = batch.id
          AND (a.deployment_type IS NULL OR a.deployment_type != d.type)`,
        lastID, batchSize)
    if err != nil {
        return 0, "", fmt.Errorf("updating batch: %w", err)
    }

    // Get the last ID in the batch for keyset pagination.
    var newLastID string
    err = conn.QueryRow(ctx, `
        SELECT id FROM alerts
        WHERE id > $1
        ORDER BY id
        LIMIT 1 OFFSET $2 - 1`, lastID, batchSize).Scan(&newLastID)
    if err != nil {
        // No more rows to process.
        return int(tag.RowsAffected()), "", nil
    }

    return int(tag.RowsAffected()), newLastID, nil
}
```

This avoids holding locks on the entire table — each batch locks only its subset of rows
and commits before the next batch begins.

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
- A rollback to a version unaware of background migration occured, since that version does not automatically reset the background migrations sequence number
- Debugging a migration in a test environment

### Monitoring

Two Prometheus metrics track background migration progress:

| Metric | Description |
|---|---|
| `rox_central_background_migration_seq_num` | Current completed seq num |
| `rox_central_background_migration_complete` | `1` when all migrations are done, `0` otherwise |

### Rollback behavior

When Central is rolled back to a previous version:

1. The old binary has a lower `CurrentBgMigrationSeqNum`
2. The runner detects `dbSeqNum > targetSeqNum` (rollback)
3. The runner resets the background DB seq num to the target
4. On the next roll-forward, background migrations re-run from the reset point

This is why idempotency is critical — migrations will be re-executed after rollback + upgrade.

## Feature flag

Background migrations are gated by the feature flag `ROX_BACKGROUND_MIGRATION` (enabled by
default), defined in `pkg/features/list.go`. When disabled, no background migrations run.
The flag is checked in `central/main.go` before starting the runner.

## Testing

Use the same test patterns as startup migrations (see
[STARTUP_MIGRATIONS.md](../../migrator/STARTUP_MIGRATIONS.md#testing)). Tests require
the `//go:build sql_integration` tag and a running PostgreSQL instance.

### What to test

- **Backfill correctness**: new column gets the right value from the serialized blob
- **Idempotency**: running twice produces the same result
- **Conflict-free with existing data**: rows already populated by Central are not overwritten
- **Graceful shutdown**: cancellation mid-migration doesn't corrupt data; re-run completes
- **Batch boundaries**: test with small batch sizes to exercise pagination logic

### Performance testing on a long-running cluster

Background migration runtime is not a primary concern since they run after
Central is serving. However, they must not starve Central of CPU or memory, and
must not cause SQL conflicts or deadlocks with concurrent Central operations.

**For migrations touching [high-cardinality tables](../../migrator/README.md#high-cardinality-tables-always-use-background-migrations),
testing with a long running cluster or a similar real load environment is mandatory.**



For the testing process:

1. **Set up a long-running cluster under load**: Deploy Central with active
   sensor connections and realistic workload (events, API calls, policy
   evaluations). The cluster should be exercising the tables the migration
   touches.
2. **Populate target tables**: If the cluster's organic data is insufficient,
   generate synthetic rows to reach representative volumes (500k–1M rows) in
   the target table before triggering the migration.
3. **Monitor during migration**:
   - **Logs**: Watch for SQL conflicts, deadlocks, lock wait timeouts, or
     retry storms. Any `deadlock detected` or `could not serialize access`
     errors indicate the batch strategy needs adjustment.
   - **Metrics**: Track Central's memory RSS, heap usage, and CPU utilization.
     The migration must not cause memory spikes that could trigger OOM kills
     or CPU saturation that degrades API latency.
   - **Central responsiveness**: Verify that API response times and sensor
     connections remain healthy throughout the migration.
4. **Tune batch size**: Adjust `batchSize` to balance throughput against
   resource impact. Smaller batches reduce peak memory, shorten lock hold
   times, and lower the chance of conflicts with Central. Larger batches
   improve throughput. Find the sweet spot where Central remains responsive
   and no conflicts or deadlocks appear in the logs.

**The simplest way to set up such environment to run the [GHA](https://github.com/stackrox/stackrox/actions/workflows/create-clusters.yml) at a stable release (e.g 4.11.0), then update central to a PR image.**
Read this [section](https://redhat.atlassian.net/wiki/spaces/StackRox/pages/309338714/Long+Running+Clusters) of the release docs for more information.

## Checklist

- [ ] Migration is idempotent (safe to re-run after rollback or crash)
- [ ] `ctx.Done()` is checked between batches for graceful shutdown
- [ ] Iterates by primary key (not by `WHERE new_column IS NULL`) for efficient pagination
- [ ] Uses `FOR UPDATE SKIP LOCKED` to handle concurrent Central writes
- [ ] Only updates rows where the column value differs from the serialized source of truth
- [ ] Application code already populates the new column on INSERT/UPDATE
- [ ] Application code tolerates partial migration state (some rows backfilled, some not)
- [ ] Registered in `central/backgroundmigrations/runner/all.go`
- [ ] `CurrentBgMigrationSeqNum` incremented in `central/backgroundmigrations/seq_num.go`
- [ ] Tests cover correctness, idempotency, existing data, and graceful shutdown
- [ ] Long-running cluster test for high-cardinality tables (no deadlocks, stable memory/CPU)
- [ ] No feature flag dependencies in migration code
