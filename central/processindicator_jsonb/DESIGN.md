# `--jsonb` Store Generator Option

## Problem

The existing postgres stores serialize the full proto as a binary vtproto blob
into a `serialized bytea` column. This is fast but opaque — the data can't be
queried, inspected, or debugged directly in SQL. Meanwhile, the `--no-serialized`
option goes to the other extreme: every proto field becomes a DB column, which
enables full SQL queryability but introduces child tables, complex scanners, and
slower writes.

A middle ground is needed: human-readable, SQL-queryable data without the
architectural complexity of full column decomposition.

## Solution

The `--jsonb` flag stores the same single-column blob as `jsonb` instead of
`bytea`, using `protojson` marshaling. This preserves the simple single-column
read/write pattern of existing stores while making the data human-readable and
SQL-queryable.

### What changes

| | bytea (default) | jsonb (new) |
|---|---|---|
| Column type | `serialized bytea` | `serialized jsonb` |
| Marshal | `obj.MarshalVT()` | `protojson.Marshal(obj)` |
| Unmarshal | `UnmarshalVTUnsafe(data)` | `protojson.Unmarshal(data, msg)` |
| Column count | unchanged | unchanged |
| Store type | `genericStore` | `NoSerializedStore` (reused) |
| Search framework | unchanged | unchanged |
| Child tables | unchanged | unchanged |

### What doesn't change

- Column count (indexed columns + 1 serialized blob)
- The search framework's GET path (`SELECT {table}.serialized`)
- GROUP BY logic (still groups on `{table}.serialized` since `NoSerialized=false`)
- Child table handling
- COUNT, SEARCH, DELETE, SELECT query types

## Architecture

### Key Reuse

- **`NoSerializedStore`** — reused as the store type since it doesn't require
  `UnmarshalVTUnsafe`. The jsonb store passes `scanRow`/`scanRows` functions that
  scan a single `[]byte` column and call `protojson.Unmarshal`.
- **`RunGetQueryForSchemaWithScanner`** — reused for scanner-based reads.
- **Existing `insertValues` template** — unchanged, still iterates DBColumnFields.
- **Existing `copyObject` template** — only the marshal call changes.

### Files Modified

| File | Changes |
|------|---------|
| `pkg/postgres/walker/schema.go` | `Jsonb bool` on Schema, `getSerializedField()` returns jsonb SQL type |
| `pkg/postgres/walker/walker.go` | `WithJsonb()` WalkOption |
| `tools/generate-helpers/pg-table-bindings/main.go` | `--jsonb` CLI flag, template data |
| `tools/generate-helpers/pg-table-bindings/funcs.go` | `canScanDirect()` helper for direct proto field scanning |
| `tools/generate-helpers/pg-table-bindings/store.go.tpl` | Jsonb marshal/unmarshal, scanRow/scanRows, Store type alias |
| `tools/generate-helpers/pg-table-bindings/schema.go.tpl` | Pass `WithJsonb()` to walker |
| `tools/generate-helpers/pg-table-bindings/list.go` | Register ProcessIndicatorJsonb |
| `pkg/search/options.go` | Register new search field labels |

### Files Created

| File | Purpose |
|------|---------|
| `proto/storage/process_indicator_jsonb.proto` | Copy of ProcessIndicator as ProcessIndicatorJsonb |
| `central/processindicator_jsonb/store/postgres/` | Generated store, tests, and benchmarks |
| `pkg/postgres/schema/process_indicator_jsonbs.go` | Generated schema (jsonb column) |

## Performance Analysis

All benchmarks run on Apple M3 Pro with local PostgreSQL 15, 3s benchtime.

### Go-Side: Writes

| Operation | Serialized (bytea) | Jsonb | NoSerialized |
|---|---|---|---|
| **Upsert x 1** | 132 us / 7 KB / 153 allocs | 147 us (+11%) / 10 KB / 189 allocs | 163 us (+23%) / 9 KB / 251 allocs |
| **Upsert x 10** | 1.06 ms / 69 KB / 1,110 allocs | 1.26 ms (+19%) / 104 KB / 1,461 allocs | 1.87 ms (+76%) / 81 KB / 866 allocs |
| **Upsert x 100** | 5.3 ms / 584 KB / 10K allocs | 7.7 ms (+45%) / 1,065 KB / 14K allocs | 16.9 ms (+219%) / 724 KB / 6K allocs |
| **Upsert x 500** | 10.3 ms / 2.2 MB / 50K allocs | 14.0 ms (+36%) / 5.9 MB / 71K allocs | 18.4 ms (+79%) / 3.8 MB / 30K allocs |

### Go-Side: Reads

| Operation | Serialized (bytea) | Jsonb | NoSerialized |
|---|---|---|---|
| **Get x 1** | 47 us / 6 KB / 98 allocs | 55 us (+17%) / 8 KB / 175 allocs | 61 us (+30%) / 33 KB / 155 allocs |
| **GetMany x 10** | 55 us / 17 KB / 207 allocs | 121 us (+120%) / 34 KB / 985 allocs | 71 us (+29%) / 46 KB / 380 allocs |
| **GetMany x 100** | 180 us / 135 KB / 1.2K allocs | 765 us (+325%) / 300 KB / 9K allocs | 236 us (+31%) / 184 KB / 2.5K allocs |
| **GetMany x 500** | 1.72 ms / 684 KB / 5.6K allocs | 3.78 ms (+120%) / 1.5 MB / 45K allocs | 2.57 ms (+49%) / 827 KB / 12K allocs |

### Postgres-Side: Storage (5,000 rows)

| | Serialized (bytea) | Jsonb | NoSerialized |
|---|---|---|---|
| **Total on disk** | 3,688 KB | 5,136 KB (+39%) | 4,856 KB (+32%) |
| **Avg row size** | 512 B | 871 B (+70%) | 226 B (-56%) |
| **Avg blob column** | 304 B | 663 B (+118%) | n/a |

### Postgres-Side: Query Execution (EXPLAIN ANALYZE)

| Query Pattern | Serialized (bytea) | Jsonb | NoSerialized |
|---|---|---|---|
| **PK lookup (1 row)** | 0.007 ms / 3 buffers | 0.004 ms / 3 buffers | 0.007 ms / 5 buffers |
| **PK IN list (500 rows)** | 0.40 ms / 334 buffers | 0.51 ms / 556 buffers | 0.32 ms / 295 buffers |
| **Full table scan (5K rows)** | 0.51 ms / 334 buffers | 0.62 ms / 556 buffers | 0.30 ms / 295 buffers |

### Where the Time Goes (GetMany x 500)

| | Serialized | Jsonb | NoSerialized |
|---|---|---|---|
| **Postgres execution** | 0.40 ms | 0.51 ms | 0.32 ms |
| **Total wall-clock** | 1.72 ms | 3.78 ms | 2.57 ms |
| **Client-side** | 1.32 ms | 3.27 ms | 2.25 ms |
| **% time in client** | 77% | 86% | 88% |

The Postgres server is not the bottleneck for any approach. The dominant cost is
always client-side: marshaling, network transfer, and GC pressure from allocations.

## Tradeoff Summary

### Serialized (bytea) -- the current default

**Pros:**
- Fastest across the board -- vtproto's `unsafe.String` achieves near-zero-copy
  deserialization (98 allocs for GetSingle)
- Smallest blob (304 B avg), compact on disk and wire, fewest buffer page reads
- Battle-tested -- every production store uses this
- Schema evolution is free -- blob absorbs proto field changes with no migration
- Best batch read scaling -- GetMany/500 at 1.72 ms and 5.6K allocs

**Cons:**
- Opaque data -- binary gibberish in `psql`
- Making a field queryable requires a migration -- the most common migration
  pattern in this project is promoting a blob field to an indexed column, which
  requires deserializing every row to backfill
- No SQL-level filtering on blob contents

### Jsonb

**Pros:**
- Human-readable in SQL -- `SELECT serialized->>'deploymentId'` works in `psql`
- SQL-queryable -- `WHERE serialized @> '{"namespace":"stackrox"}'` with GIN
  index support
- Same architecture as bytea -- simplest possible change from the default
- Good single-row write performance -- only 11% slower than bytea
- Schema evolution is free -- same as bytea
- Postgres validates JSON on insert

**Cons:**
- Batch reads are the weakest point -- GetMany/500 is 120% slower than bytea
  (3.78 ms vs 1.72 ms), driven by `protojson.Unmarshal` doing per-field
  reflection: 89 allocs/row vs 11 for bytea
- 2.2x larger blobs -- 66% more heap space and buffer page reads
- Largest memory footprint on writes -- 5.9 MB for UpsertMany/500 vs 2.2 MB
- 39% more total disk space
- Making a field queryable still requires a migration (same as bytea)

### NoSerialized -- all columns, no blob

**Pros:**
- Every field is immediately a column -- no migration to promote from a blob.
  Adding a proto field = `ALTER TABLE ADD COLUMN` with a default; no
  deserialize-and-backfill step
- Smallest rows (226 B avg), fewest buffer page reads (295 vs 334)
- Fastest Postgres-side scans (0.30 ms for 5K rows vs 0.51 ms for bytea)
- Competitive batch reads after direct-scan optimization -- GetMany/500 at
  2.57 ms is only 49% slower than bytea, and 32% faster than jsonb
- Fewest allocations on writes (unnest-based bulk INSERT)

**Cons:**
- Slowest writes at scale -- UpsertMany/100 takes 16.9 ms (3.2x bytea) due to
  per-row UPDATE fallback for non-unnestable columns (string arrays, maps)
- Every proto field change requires a migration (even non-queryable fields)
- Child tables add complexity (separate queries for repeated fields)
- Higher per-read memory for single objects (33 KB vs 6 KB for bytea)
- More generated code to maintain

### Migration comparison

The most common migration in this project is making a field queryable. The
approaches differ in when and how that migration happens:

| | bytea / jsonb | NoSerialized |
|---|---|---|
| **Add a proto field (not queryable)** | No migration | Migration required |
| **Make a field queryable** | Migration required (add column + deserialize all rows to backfill) | Already done -- field is already a column |
| **Migration complexity** | Higher (must deserialize blob per row) | Lower (simple `ALTER TABLE ADD COLUMN`) |

### When to Use What

| Scenario | Best choice | Why |
|---|---|---|
| Production hot path, high throughput | **bytea** | Fastest reads and writes, lowest memory |
| Debugging, ad-hoc analysis, ops tooling | **jsonb** | Read data in `psql`, query with SQL |
| Tables where fields are frequently promoted | **NoSerialized** | Every field is already a column |
| SQL-native queries against all fields | **NoSerialized** | Every field is indexable |
| Protobuf schema changes frequently | **bytea or jsonb** | Blob absorbs changes |
| Data rarely read in bulk | **jsonb** | Write overhead is modest (11-36%) |
| Large batch reads (500+ rows) | **bytea** | Jsonb is 2.2x slower at this scale |

## Read-Path Optimization: Direct Proto Field Scanning

As part of this work, the NoSerialized read path was optimized to scan directly
into proto struct fields instead of using intermediate variables.

### Before

```go
// 20 intermediate variables declared
var col_Id string
var col_DeploymentId string
// ... 18 more ...

row.Scan(&col_Id, &col_DeploymentId, ...)  // pgx copies each string

// buildFromScan copies every value again into the proto
obj := &storeType{}
obj.Id = col_Id           // second copy
obj.DeploymentId = col_DeploymentId
```

### After

```go
obj := &storeType{}
obj.Signal = &storage.ProcessSignalNoSerialized{}

// Only 6 temp vars for fields needing type conversion (datetime, uuid)
var col_Signal_Time *time.Time
var col_ContainerStartTime *time.Time
// ... 4 more ...

row.Scan(
    &col_Id,              // uuid: needs conversion
    &obj.ContainerName,   // string: scan directly into proto
    &obj.Signal.Name,     // nested: scan directly
    &col_Signal_Time,     // datetime: needs conversion
    // ...
)

// Only convert the 6 fields that need it
obj.Signal.Time = protocompat.ConvertTimeToTimestampOrNil(col_Signal_Time)
```

### Impact (GetMany x 500)

| Metric | Before | After | Improvement |
|---|---|---|---|
| Allocations | 17,654 | 12,154 | -31% |
| Bytes | 919 KB | 827 KB | -10% |
| Wall-clock | 2.70 ms | 2.57 ms | -5% |

Fields that scan directly: strings, bools, uint32, int32, float, slices, maps.
Fields that still need temp vars: datetime/datetimetz (scanned as `*time.Time`,
converted to `*timestamppb.Timestamp`), enums (scanned as int32, cast to enum
type), UUID strings (scanned with pgx UUID codec).
