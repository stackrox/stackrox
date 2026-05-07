# FastJSON: Reflection-Free JSON Marshal/Unmarshal for JSONB Stores

**Date:** 2026-04-03
**Platform:** Apple M3 Pro, PostgreSQL 15, Go 1.24, 12 cores
**Branch:** `dashrews/experiment-non-serialized`

## Problem

The JSONB store option uses `protojson.Marshal`/`protojson.Unmarshal` from
`google.golang.org/protobuf/encoding/protojson` to serialize proto messages
into PostgreSQL JSONB columns. Benchmarks showed this was the primary
bottleneck on reads:

- **GetMany x500**: 3,780 µs with protojson vs 803 µs with vtprotobuf binary (4.7x slower)
- **Per-row allocations**: ~89 allocations vs ~11 for vtprotobuf binary (8x more)

The root cause is protojson's use of `protoreflect` — it walks the proto
message descriptor at runtime to discover field names, types, and presence,
boxing each value in a `protoreflect.Value` interface along the way. This
reflection overhead dominates the actual JSON encoding work.

## Solution

Replace `protojson` with generated, reflection-free `MarshalFastJSON()`/
`UnmarshalFastJSON()` methods that use direct struct field access.

### Runtime Library (`pkg/fastjson/`)

**Writer** (`writer.go`): Append-based JSON builder. Field values are written
directly to a `[]byte` buffer using `strconv.AppendUint`, `strconv.AppendInt`,
etc. No intermediate representations, no interface boxing. Comma management is
handled by a `first` flag toggled by `FieldName()` and `ArrayElem()`.

**Reader** (`reader.go`): Zero-copy byte-scanning JSON tokenizer. Scans through
the input `[]byte` directly without `encoding/json.Decoder` (an earlier
prototype using `encoding/json.Decoder` was actually 2.7x *slower* than
protojson due to per-token allocations and `json.RawMessage` copies). The
scanner handles string unescaping, numeric parsing, null detection, and
value skipping (for unknown fields) with minimal allocations.

**Well-known type delegation** (`wellknown.go`): For `google.protobuf.Timestamp`
and other well-known types, the generated code delegates to `protojson.Marshal`/
`protojson.Unmarshal` rather than reimplementing their complex encoding rules
(Timestamp → RFC 3339, Duration → `"1.5s"`, etc.). This keeps the compliance
risk near zero for the hardest parts of the proto3 JSON spec.

### Generated Code (`generated/storage/process_indicator_jsonb_fastjson.pb.go`)

Hand-written for the evaluation (a `protoc-gen-go-fastjson` plugin would
generate this automatically). For each message type, generates:

- `MarshalFastJSON() ([]byte, error)` — public entry point, allocates writer
- `marshalFastJSON(w *fastjson.Writer) error` — internal, writes to shared writer
- `UnmarshalFastJSON(data []byte) error` — public entry point

#### Marshal pattern

```go
func (m *ProcessIndicatorJsonb) marshalFastJSON(w *fastjson.Writer) error {
    w.BeginObject()
    if m.Id != "" {
        w.FieldName("id")     // writes ,"id": with comma management
        w.String(m.Id)        // JSON-escaped string
    }
    if m.Signal != nil {
        w.FieldName("signal")
        if err := m.Signal.marshalFastJSON(w); err != nil {  // recursive
            return err
        }
    }
    if m.ContainerStartTime != nil {
        w.FieldName("containerStartTime")
        if err := fastjson.MarshalTimestamp(w, m.ContainerStartTime); err != nil {  // delegate
            return err
        }
    }
    w.EndObject()
    return nil
}
```

#### Unmarshal pattern

```go
func (m *ProcessIndicatorJsonb) UnmarshalFastJSON(data []byte) error {
    r := fastjson.NewReader(data)
    return r.ReadObject(func(key string) error {
        switch key {
        case "id":
            v, err := r.ReadString()
            if err != nil { return err }
            m.Id = v
        case "deploymentId", "deployment_id":    // accept both per proto3 spec
            v, err := r.ReadString()
            if err != nil { return err }
            m.DeploymentId = v
        case "signal":
            raw, isNull, err := r.ReadNullOrRaw()
            if err != nil { return err }
            if isNull { m.Signal = nil; return nil }
            m.Signal = &ProcessSignalJsonb{}
            return m.Signal.UnmarshalFastJSON(raw)
        default:
            return r.SkipValue()    // unknown fields silently skipped
        }
        return nil
    })
}
```

## Proto3 JSON Spec Compliance

The approach uses **delegate + verify** rather than full reimplementation:

| Spec Rule | How We Handle It |
|-----------|-----------------|
| Field names as lowerCamelCase | Generated from `field.Desc.JSONName()` at codegen time |
| int64/uint64 as quoted strings | `w.Int64(v)` writes `"123"`, reader accepts both quoted and bare |
| Enum values as string names | `v.String()` (generated proto method) |
| Omit default-valued fields | Zero-check before each field (`m.Id != ""`, `m.Pid != 0`) |
| Accept both camelCase and snake_case | Both listed in unmarshal `switch` cases |
| Null = field unset | `ReadNullOrRaw()` checks for null before parsing |
| Unknown fields skipped | `default: r.SkipValue()` |
| Well-known types | Delegated to protojson (Timestamp, Duration, etc.) |

**Verification**: 11 oracle tests round-trip messages through both fastjson and
protojson in all combinations, asserting `proto.Equal` on the results. This
catches any encoding deviation automatically.

## Why Not encoding/json.Decoder?

An earlier prototype used `encoding/json.Decoder` with `Token()` for the reader.
Results were significantly worse than protojson:

| Reader Implementation | Unmarshal ns/op | Allocs/op |
|-----------------------|----------------|-----------|
| protojson | 5,831 | 116 |
| encoding/json.Decoder | 15,911 | 404 |
| Zero-copy byte scanner | 2,437 | 65 |

`encoding/json.Decoder` allocates a new `interface{}` for each token, creates
`json.RawMessage` copies for `Decode(&raw)`, and rebuilds internal state on each
call. The zero-copy byte scanner reads directly from the input slice, only
allocating for string values that need unescaping and for the final Go struct fields.

## Benchmark Results

### Pure Marshal/Unmarshal (no database)

| Benchmark | protojson | fastjson | Speedup | Alloc Reduction |
|-----------|-----------|----------|---------|-----------------|
| Marshal | 3,173 ns / 42 allocs | 1,261 ns / 11 allocs | **2.5x faster** | **3.8x fewer** |
| Unmarshal | 5,831 ns / 116 allocs | 2,437 ns / 65 allocs | **2.4x faster** | **1.8x fewer** |
| Round-trip | 9,258 ns / 158 allocs | 3,864 ns / 76 allocs | **2.4x faster** | **2.1x fewer** |

### End-to-End Database Benchmarks

#### Single Operations

| Operation | Serialized (bytea) | JSONB (fastjson) | NoSerialized |
|-----------|-------------------|---------------------|--------------|
| Upsert | 134 µs / 153 allocs | 138 µs / 156 allocs | 120 µs / 160 allocs |
| Get | 47 µs / 98 allocs | 50 µs / 132 allocs | 62 µs / 161 allocs |
| Count | 74 µs / 52 allocs | 81 µs / 52 allocs | 69 µs / 52 allocs |

#### GetMany (batch reads)

| Batch Size | Serialized (bytea) | JSONB (fastjson) | JSONB (old protojson) | NoSerialized |
|-----------|-------------------|---------------------|-----------------------|--------------|
| 10 | 57 µs / 207 allocs | 103 µs / 555 allocs | ~190 µs | 80 µs / 458 allocs |
| 100 | 199 µs / 1.2K allocs | 440 µs / 4.7K allocs | ~930 µs | 270 µs / 3.3K allocs |
| 500 | 803 µs / 5.6K allocs | 1,868 µs / 23K allocs | ~3,780 µs | 1,088 µs / 16K allocs |

#### UpsertMany (batch writes)

| Batch Size | Serialized (bytea) | JSONB (fastjson) | NoSerialized |
|-----------|-------------------|---------------------|--------------|
| 10 | 519 µs / 1.1K allocs | 546 µs / 1.1K allocs | 411 µs / 556 allocs |
| 100 | 1,916 µs / 10K allocs | 3,026 µs / 11K allocs | 1,498 µs / 3.8K allocs |
| 500 | 7,608 µs / 50K allocs | 12,691 µs / 55K allocs | 8,599 µs / 18K allocs |

## Remaining Gap Analysis

JSONB with fastjson is now within 6% of bytea on single Get, but still 2.3x
slower on GetMany x500 (was 4.7x with protojson). The remaining gap is:

1. **Payload size**: JSON text is larger than protobuf binary (~663 bytes vs
   ~304 bytes per row for this message). More bytes transferred from Postgres.
2. **Parsing cost**: Even with zero-copy scanning, JSON parsing requires
   character-by-character field name matching. Protobuf binary uses varint
   field tags that decode in a single operation.
3. **Allocation count**: 46 allocs/row vs 11 for vtprotobuf. Most fastjson
   allocs come from Go string construction during unmarshal (each `ReadString`
   creates a Go string from the scanned bytes).

These are fundamental to the JSON-vs-binary format tradeoff and cannot be
eliminated without abandoning JSON encoding entirely.

## Future Work: protoc-gen-go-fastjson Plugin

For production use, the hand-written fastjson code should be generated by a
protoc plugin. The plugin would:

1. Use `google.golang.org/protobuf/compiler/protogen` (same framework as vtprotobuf)
2. Generate `_fastjson.pb.go` files alongside existing `_vtproto.pb.go` files
3. Handle all proto types: scalars, enums, oneofs (type switch), maps, nested messages
4. Delegate well-known types to protojson
5. Integrate into `make/protogen.mk` alongside the existing vtprotobuf rule

Estimated scope: ~500-800 lines of generator code. See the plan file at
`.claude/plans/glistening-cooking-fiddle.md` for the full plugin design.

## Files

| File | Purpose |
|------|---------|
| `pkg/fastjson/writer.go` | Append-based JSON writer with typed methods |
| `pkg/fastjson/reader.go` | Zero-copy byte-scanning JSON tokenizer |
| `pkg/fastjson/wellknown.go` | protojson delegation for Timestamp etc. |
| `generated/storage/process_indicator_jsonb_fastjson.pb.go` | Marshal/Unmarshal for ProcessIndicatorJsonb |
| `generated/storage/process_indicator_jsonb_fastjson_test.go` | Oracle tests + pure benchmarks |
| `central/processindicator_jsonb/store/postgres/store.go` | Store wired to use fastjson |
