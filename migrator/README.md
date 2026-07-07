# StackRox Database Migrations

## IMPORTANT: Idempotency and Backwards compatibility

All migrations must be backwards compatible to ensure safe rollback.
Migrations must be idempotent because they re-execute when rolling forward after a rollback.
Migrations are forward only, there are no reverse migrations.

**All migrations must be backwards compatible**. This means:
- No destructive schema changes (dropping columns, renaming columns)
- Data transformations must work with both old and new application code
- New columns should use the zero value or a new field name to avoid conflicts

GORM AutoMigrate enforces this by design: it will not remove unused columns or make
breaking data type changes, though it will perform updates on precision.
See [GORM Auto Migration](https://gorm.io/docs/migration.html#Auto-Migration).

When a previously-supported version can no longer tolerate the current schema, update
`MinimumSupportedDBVersionSeqNum` in `pkg/migrations/internal/fallback_seq_num.go`.
The migrator will reject upgrades from versions below this threshold.

## Do you need a migration?

**Most changes do NOT require a migration.**

```
Is the change schema-only (new column without backfill, new table)?
├── YES → No migration needed. GORM AutoMigrate and proto code generation handles it.
└── NO →
    Adding an index?
    ├── YES → No migration needed. Use proto tags instead.
    │         Is the table on the high-cardinality list below?
    │         ├── YES → Use sql:"background-index=btree" (created non-blocking after startup)
    │         └── NO  → Use sql:"index=btree" (created at startup)
    └── NO →
        Can the new column tolerate its zero value until normal operation populates it?
        ├── YES → No migration needed.
        └── NO →
            Setting a static column default (no backfill of existing rows)?
            ├── YES → STARTUP MIGRATION [1] with ALTER TABLE ... ALTER COLUMN ... SET DEFAULT
            └── NO → Existing data must be backfilled or transformed.
                      → BACKGROUND MIGRATION [2]. Central MUST tolerate partially
                        migrated data while the migration runs.
```

**All data backfills must be background migrations.** Central supports both Recreate and
RollingUpdate rollout strategies. With RollingUpdate, old Central pods continue running
during startup migrations, so any data backfill would be immediately inconsistent — old
pods keep writing rows without populating the new column. Background migrations avoid this
by waiting for the rollout to fully complete (all old pods terminated) before starting.

Central code must tolerate partially migrated state (some rows backfilled, some not) while
a background migration is in progress.

- **\[1\] Startup migration guide**: [STARTUP_MIGRATIONS.md](STARTUP_MIGRATIONS.md)
- **\[2\] Background migration guide**: [BACKGROUND_MIGRATIONS.md](../central/backgroundmigrations/BACKGROUND_MIGRATIONS.md)

## Migration types

| | Startup migration | Background migration |
|---|---|---|
| **Runs when** | Before Central starts (during upgrade) | After Central is running and rollout is complete |
| **Central availability** | Central is **down** during execution | Central is **live** and serving traffic |
| **Downtime impact** | Extends upgrade downtime (Recreate) or new pod startup time (RollingUpdate) | None |
| **Concurrency** | Exclusive access to the database | Must handle concurrent reads/writes from Central |
| **Transaction support** | Version update wrapped in transaction; migration code itself is not automatically wrapped | No automatic transaction wrapping |
| **When to use** | DDL/schema changes that GORM AutoMigrate cannot handle (e.g., `SET DEFAULT`, dropping tables, constraints) | All data backfills and transformations |
| **Sequence tracking** | `CurrentDBVersionSeqNum` in `pkg/migrations/internal/seq_num.go` | `CurrentBgMigrationSeqNum` in `central/backgroundmigrations/seq_num.go` |
| **Code location** | `migrator/migrations/m_{N}_to_m_{N+1}_*/` | `central/backgroundmigrations/migrations/*/` |
| **Rerun on rollback** | Yes -- all migrations between versions re-execute on roll-forward | Yes -- seq num resets on rollback, migrations re-run on next upgrade |
| **Skip support** | `ROX_SKIP_MIGRATIONS` (comma-separated seq nums) | `ROX_SKIP_BACKGROUND_MIGRATIONS` (comma-separated seq nums) |
| **Force rerun** | Rerun happens on rollback/rollforward | Rerun on rollback/rollforward `ROX_BACKGROUND_MIGRATION_OVERRIDE_SEQ_NUM` + `ROX_BACKGROUND_MIGRATION_OVERRIDE_TAG` for fallback on upgrade from older versions |
| **Detailed guide** | [STARTUP_MIGRATIONS.md](STARTUP_MIGRATIONS.md) | [BACKGROUND_MIGRATIONS.md](../central/backgroundmigrations/BACKGROUND_MIGRATIONS.md) |

### High-cardinality tables (always use background migrations)

These tables regularly exceed 100k rows in production environments:

| Table | Observed in |
|---|---|
| `alerts` | RHIT, AppSRE |
| `image_component_v2` | RHIT, IBM, Dogfood, AppSRE |
| `image_cves_v2` | RHIT, IBM, AppSRE |
| `images_layers` | RHIT, IBM |
| `listening_endpoints` | RHIT |
| `network_flows_v2` | RHIT, IBM, Dogfood, AppSRE, RHIT2 |
| `process_indicators` | RHIT, IBM, AppSRE, RHIT2 |

## How the systems interact

```
Central upgrade sequence:

  +----------------------------------------------------------+
  |                   STARTUP PHASE                          |
  |                                                          |
  |  1. Migrator acquires advisory lock                      |
  |  2. Run startup migrations sequentially                  |
  |  3. GORM AutoMigrate applies all registered schemas      |
  |  4. Apply startup indexes (CREATE INDEX CONCURRENTLY)    |
  |  5. Release advisory lock                                |
  |                                                          |
  |  Central is NOT serving traffic during this phase         |
  +----------------------------+-----------------------------+
                               |
                               v
  +----------------------------------------------------------+
  |              CENTRAL STARTS SERVING                      |
  |                                                          |
  |  6. Datastores initialize                                |
  |  7. gRPC/HTTP servers start                              |
  |  8. Central is ready and serving traffic                  |
  +----------------------------+-----------------------------+
                               |
                               v
  +----------------------------------------------------------+
  |         BACKGROUND PHASE (concurrent)                    |
  |                                                          |
  |  Background Migrations:                                  |
  |  9. Rollout checker waits for all old pods to terminate   |
  | 10. Acquire background migration advisory lock           |
  | 11. Run background migrations sequentially               |
  | 12. Each migration checkpoints progress on completion    |
  | 13. Reconcile background indexes (CREATE INDEX           |
  |     CONCURRENTLY for all background-index fields)        |
  |                                                          |
  |  Central IS serving traffic during this phase             |
  +----------------------------------------------------------+
```

The rollout checker (step 9) ensures that no old-version Central pods are still running
before background migrations begin. This prevents conflicts between old code writing data
in the pre-migration format while the migration is transforming it.

### Testing backwards compatibility

The `gke-upgrade-tests` start with a 4.1.3 deployment and upgrade to the current
release, executing all migrations and verifying rollback succeeds.

Beyond that, for any schema and/or data changes, engineers should:

1. Deploy the previous version
2. Upgrade to the current version
3. Exercise the change (populate any necessary data)
4. Rollback to the previous version
5. Verify Central is up and functioning
6. Roll forward to the current version
7. Verify migrations executed and functionality works

## Feature flags and migrations

**Do not use feature flags in migration code.**

Both startup and background migrations must produce the same result regardless of feature
flag state. If a feature needs a new field, the migration must populate it unconditionally.
The feature flag controls whether the application *uses* the field, not whether it *exists*.

If a feature cannot work without a migration, the data should be migrated prior to any work
on the feature. If a feature can work without migration at lesser cost, workaround code can
be written to work on the existing schema. Once the feature is GA, a migration can be written
and the workaround code replaced.

## Index Management

All indexes are managed through proto tags and the code generator, not GORM struct tags.
The generator produces complete `CREATE INDEX` SQL at codegen time; at runtime the SQL is
executed verbatim. There are two creation paths depending on table size.

### Adding a new index

1. Add the appropriate tag to the proto field:
   - **Small or new tables**: `sql:"index=btree"` -- created at startup using
     `CREATE INDEX CONCURRENTLY`
   - **High-cardinality tables**: `sql:"background-index=btree"` -- created after startup
     using `CREATE INDEX CONCURRENTLY`
2. Run `make proto-generated-srcs && make go-generated-srcs` to regenerate schema code
3. The generated `CreateStmts` will include `Indexes: []*postgres.IndexDefinition{...}` entries

### How it works

- **Startup indexes** (`sql:"index"`, `Background: false`): Created by
  `schema.ApplyAllStartupIndexes()` before Central serves traffic. All indexes use
  `CREATE INDEX CONCURRENTLY` to avoid locking the table.
- **Background indexes** (`sql:"background-index"`, `Background: true`): Created by the
  background migration runner using `CREATE INDEX CONCURRENTLY IF NOT EXISTS` after all
  numbered migrations complete. Runs after Central starts serving traffic. Use this for
  high-cardinality tables where index creation on large datasets is slow.
- **SAC filter index**: Automatically generated for tables with `Cluster ID` or `Namespace`
  search fields.

### Composite indexes

To create a composite index across multiple fields, give them the same index name:

```protobuf
string auth_provider_id = 1; // @gotags: sql:"index=name:groups_unique;type:btree;category:unique"
string key = 2;              // @gotags: sql:"index=name:groups_unique;type:btree;category:unique"
string value = 3;            // @gotags: sql:"index=name:groups_unique;type:btree;category:unique"
```

### High-cardinality table indexes

When adding **new** indexes to the tables listed in the
[high-cardinality table list](#high-cardinality-tables-always-use-background-migrations),
always use `background-index`.

Existing indexes on these tables that were previously created by GORM at startup are now
also managed by the code generator and applied outside of GORM.

## Writing migrations

- **Startup migration**: See [STARTUP_MIGRATIONS.md](STARTUP_MIGRATIONS.md) for the full
  guide including bootstrapping, data access patterns, frozen schemas, and testing.
- **Background migration**: See [BACKGROUND_MIGRATIONS.md](../central/backgroundmigrations/BACKGROUND_MIGRATIONS.md)
  for the full guide including concurrency handling, batching patterns, and testing.

## Migrator limitations

The migrator upgrades data from a previous datamodel to the current one. Each migration may
apply a different schema change, so the current datastore or schema code cannot be used.

Each migration is responsible for accessing the databases it needs and converting the data.
A migration must **never** import schemas from `pkg/postgres/schema` -- those evolve with the
latest release. Instead, freeze schemas inside the migration package. This is primarily
relevant for background migrations that deserialize data; startup migrations that only
run DDL typically don't need frozen schemas.

## History

1. Before release 3.73, the migrator targeted internal key-value stores (BoltDB and RocksDB).
2. Releases 3.73 and 3.74 introduced Postgres as Technical Preview with parallel migration
   paths (key-value and data-move migrations).
3. After 4.0, Postgres became the default data store.
4. Starting in 4.2, all migrations must be backwards compatible while previous releases are
   supported.
5. Starting in 4.11, background migrations were introduced for high-cardinality tables to
   eliminate upgrade downtime for large data transformations.
