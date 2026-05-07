# Store Generator Options: `--no-serialized` and `--jsonb`

## Motivation

The existing postgres stores serialize the full proto as a binary vtproto blob
into a `serialized bytea` column. This is fast and battle-tested, but it creates
several problems:

- **Opaque data** — the serialized column is binary gibberish in `psql`, making
  debugging and ad-hoc analysis difficult
- **Migration burden** — the most common migration pattern in this project is
  promoting a blob field to an indexed column, which requires deserializing
  every row to backfill. This is expensive and error-prone
- **Memory cost** — marshaling and unmarshaling the full proto on every
  read/write consumes CPU and heap, especially at scale
- **No SQL filtering** — blob contents cannot be queried, indexed, or filtered
  at the SQL level

For some objects, keeping a serialized blob makes sense. For others — where
fields are frequently promoted to columns, or where SQL-level inspection is
valuable — alternatives are needed.

## Overview of the Three Approaches

| | **bytea** (default) | **`--no-serialized`** | **`--jsonb`** |
|---|---|---|---|
| Serialized column | `bytea` blob | None — all fields are columns | `jsonb` blob |
| Marshal | `obj.MarshalVT()` | n/a | `protojson.Marshal(obj)` |
| Unmarshal | `UnmarshalVTUnsafe(data)` | Scanner-based field reconstruction | `protojson.Unmarshal(data, msg)` |
| Store type | `genericStore` | `NoSerializedStore` | `NoSerializedStore` (reused) |
| Child tables | Unchanged | Separate queries for repeated fields | Unchanged |
| Search framework | Unchanged | GET path selects individual columns | Unchanged |
| Schema evolution | Blob absorbs changes | Every field change = migration | Blob absorbs changes |

## `--no-serialized` Design

The `--no-serialized` flag eliminates the serialized blob entirely. Every proto
field becomes a database column, making all data immediately queryable and
indexable without migration.

### Generator usage

```go
//go:generate pg-table-bindings-wrapper --type=storage.ProcessIndicatorNoSerialized --no-serialized --search-category 85 --schema-directory=pkg/postgres/schema
```

### Writes

- **Bulk INSERT**: Uses PostgreSQL's `unnest()` function for efficient batch
  inserts. All scalar columns are passed as arrays and unnested in a single
  statement
- **UPDATE fallback**: Columns that cannot be unnested (string arrays, maps)
  fall back to per-row UPDATE statements after the bulk insert
- No marshal step — field values are passed directly as SQL parameters

### Reads

- **Scanner-based**: Generated `scanRow`/`scanRows` functions reconstruct the
  proto from individual column values
- **Direct proto field scanning** (optimization): Where possible, `pgx.Scan`
  writes directly into proto struct fields, eliminating intermediate variables
  and a second copy step
- **Search framework GET path**: Selects individual columns instead of a
  single serialized blob

### Store type

Uses `NoSerializedStore` from `pkg/search/postgres/no_serialized_store.go`,
which accepts scanner functions instead of requiring `UnmarshalVTUnsafe`.
Supports `FetchChildren` opt-in for child table reconstruction.

### Files modified

| File | Changes |
|------|---------|
| `pkg/postgres/walker/schema.go` | `NoSerialized bool` on Schema, skip serialized field generation |
| `pkg/postgres/walker/walker.go` | `WithNoSerialized()` WalkOption |
| `tools/generate-helpers/pg-table-bindings/main.go` | `--no-serialized` CLI flag, template data |
| `tools/generate-helpers/pg-table-bindings/funcs.go` | `canScanDirect()` helper for direct proto field scanning |
| `tools/generate-helpers/pg-table-bindings/store.go.tpl` | Scanner generation, NoSerializedStore construction, write templates |
| `tools/generate-helpers/pg-table-bindings/schema.go.tpl` | Pass `WithNoSerialized()` to walker |
| `pkg/search/postgres/common.go` | `RunGetQueryForSchemaWithScanner` and related functions |
| `pkg/search/postgres/no_serialized_store.go` | `NoSerializedStore` type |

## `--jsonb` Design

The `--jsonb` flag stores the same single-column blob as `jsonb` instead of
`bytea`, using `protojson` marshaling. This preserves the simple single-column
read/write pattern of existing stores while making the data human-readable and
SQL-queryable.

### Generator usage

```go
//go:generate pg-table-bindings-wrapper --type=storage.ProcessIndicatorJsonb --jsonb --search-category 86 --schema-directory=pkg/postgres/schema
```

### What changes from bytea

| | bytea (default) | jsonb |
|---|---|---|
| Column type | `serialized bytea` | `serialized jsonb` |
| Marshal | `obj.MarshalVT()` | `protojson.Marshal(obj)` |
| Unmarshal | `UnmarshalVTUnsafe(data)` | `protojson.Unmarshal(data, msg)` |
| Column count | unchanged | unchanged |
| Store type | `genericStore` | `NoSerializedStore` (reused) |
| Search framework | unchanged | unchanged |
| Child tables | unchanged | unchanged |

### Architecture

The `--jsonb` store reuses `NoSerializedStore` infrastructure because the
`genericStore` requires `UnmarshalVTUnsafe`, which parses binary protobuf and
would fail on JSON bytes. Instead, the jsonb store provides trivial scanners
that scan a single `[]byte` column and call `protojson.Unmarshal`:

```go
func scanRow(row pgx.Row) (*storeType, error) {
    var data []byte
    if err := row.Scan(&data); err != nil {
        return nil, err
    }
    msg := &storeType{}
    if err := protojson.Unmarshal(data, msg); err != nil {
        return nil, err
    }
    return msg, nil
}
```

The GET query `SELECT {table}.serialized` works for both `bytea` and `jsonb`
columns — `pgx` scans a `jsonb` column into `[]byte` just like `bytea`. No
search framework changes are needed.

### Files modified

| File | Changes |
|------|---------|
| `pkg/postgres/walker/schema.go` | `Jsonb bool` on Schema, `getSerializedField()` returns jsonb SQL type |
| `pkg/postgres/walker/walker.go` | `WithJsonb()` WalkOption |
| `tools/generate-helpers/pg-table-bindings/main.go` | `--jsonb` CLI flag, template data |
| `tools/generate-helpers/pg-table-bindings/store.go.tpl` | Jsonb marshal/unmarshal, scanRow/scanRows, Store type alias |
| `tools/generate-helpers/pg-table-bindings/schema.go.tpl` | Pass `WithJsonb()` to walker |
| `tools/generate-helpers/pg-table-bindings/list.go` | Register ProcessIndicatorJsonb |

## Benchmark Results

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

### Serialized (bytea) — the current default

**Pros:**
- Fastest across the board — vtproto's `unsafe.String` achieves near-zero-copy
  deserialization (98 allocs for GetSingle)
- Smallest blob (304 B avg), compact on disk and wire, fewest buffer page reads
- Battle-tested — every production store uses this
- Schema evolution is free — blob absorbs proto field changes with no migration
- Best batch read scaling — GetMany/500 at 1.72 ms and 5.6K allocs

**Cons:**
- Opaque data — binary gibberish in `psql`
- Making a field queryable requires a migration (add column + deserialize all
  rows to backfill)
- No SQL-level filtering on blob contents

### Jsonb

**Pros:**
- Human-readable in SQL — `SELECT serialized->>'deploymentId'` works in `psql`
- SQL-queryable — `WHERE serialized @> '{"namespace":"stackrox"}'` with GIN
  index support
- Same architecture as bytea — simplest possible change from the default
- Good single-row write performance — only 11% slower than bytea
- Schema evolution is free — same as bytea
- Postgres validates JSON on insert

**Cons:**
- Batch reads are the weakest point — GetMany/500 is 120% slower than bytea
  (3.78 ms vs 1.72 ms), driven by `protojson.Unmarshal` doing per-field
  reflection: 89 allocs/row vs 11 for bytea
- 2.2x larger blobs — 66% more heap space and buffer page reads
- Largest memory footprint on writes — 5.9 MB for UpsertMany/500 vs 2.2 MB
- 39% more total disk space
- Making a field queryable still requires a migration (same as bytea)

### NoSerialized — all columns, no blob

**Pros:**
- Every field is immediately a column — no migration to promote from a blob.
  Adding a proto field = `ALTER TABLE ADD COLUMN` with a default; no
  deserialize-and-backfill step
- Smallest rows (226 B avg), fewest buffer page reads (295 vs 334)
- Fastest Postgres-side scans (0.30 ms for 5K rows vs 0.51 ms for bytea)
- Competitive batch reads after direct-scan optimization — GetMany/500 at
  2.57 ms is only 49% slower than bytea, and 32% faster than jsonb
- Fewest allocations on writes (unnest-based bulk INSERT)

**Cons:**
- Slowest writes at scale — UpsertMany/100 takes 16.9 ms (3.2x bytea) due to
  per-row UPDATE fallback for non-unnestable columns (string arrays, maps)
- Every proto field change requires a migration (even non-queryable fields)
- Child tables add complexity (separate queries for repeated fields)
- Higher per-read memory for single objects (33 KB vs 6 KB for bytea)
- More generated code to maintain

### Migration Comparison

The most common migration in this project is making a field queryable. The
approaches differ in when and how that migration happens:

| | bytea / jsonb | NoSerialized |
|---|---|---|
| **Add a proto field (not queryable)** | No migration | Migration required |
| **Make a field queryable** | Migration required (add column + deserialize all rows to backfill) | Already done — field is already a column |
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

As part of the `--no-serialized` work, the read path was optimized to scan
directly into proto struct fields instead of using intermediate variables.

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
