---
name: proto-writer
description: Guide the creation or modification of StackRox protobuf definitions and Postgres store generation. Use when creating new storage types, adding fields to existing protos, setting up gen.go files, or making architectural decisions about data modeling (embedded vs FK relationships, search scope, API/storage separation).
disable-model-invocation: true
---

# Proto Writer

Interactive skill for creating and modifying StackRox protobuf definitions with correct tags, gen.go files, and store generation.

## Tag and Flag Reference

Read `proto/TAGS.md` for the complete reference on all proto `@gotags:` annotations (sql, search, policy, hash, sensorhash, scrub, validate, crYaml), gen.go generator flags, scenarios, and type mappings. The sections below guide the interactive workflow.

## Step 1: Determine the Change Type

Ask the user which scenario applies:

### A) Adding or modifying fields on an existing proto

- Identify the proto file and the gen.go file that generates its store.
- The gen.go file may NOT need changes -- only execution. See the decision below.
- **gen.go changes needed when:** adding a new FK relationship (`--references`), adding a new search category (`--search-category`), changing search scope (`--search-scope`), or changing store behavior.
- **gen.go changes NOT needed (but execution IS) when:** adding/removing `search:` tags, `sql:` index tags, non-SQL tags (hash, scrub, policy), or new fields without FK/search-category changes.

### B) Creating a new storage proto + store

- Needs a new proto file in `proto/storage/`, a new gen.go, and registration in several files.
- Follow all steps below.

### C) Creating a new no-serialized store

- Same as (B) but uses `--no-serialized` flag — all proto fields become individual DB columns, no serialized bytea blob.
- Generates a `NoSerializedStore[T]` with column-based scan/insert and optional child fetch control.
- `sql:"-"` is **disallowed** (walker panics). Use `sql:"strategy(bytea)"` for repeated messages you want in the parent table.
- Follow all steps below, with the no-serialized architectural decision in Step 3c.

### D) Adding a v2 API for a storage proto

- Storage protos (`proto/storage/`) are written first. v2 API protos (`proto/api/v2/`) are created afterward as a separate layer.
- v2 API protos MUST have separate message bodies from storage protos — even if the fields look similar.
- A `convert.go` bridges them. See `central/reports/service/v2/convert.go` for the pattern.
- When updating storage protos, ask: "Should I update the v2 API proto and converter too?"

## Fast Path: Simple Field Addition

If the user is adding a plain field to an existing proto that already has a gen.go (no FK, no new search category, no secrets, no policy criteria), skip directly to **Step 6**. Only the proto field and possibly a `search:` or `sql:"index"` tag are needed — the gen.go doesn't need changes, just re-execution.

## Step 2: Tag Decision Tree

When adding a field, walk through these questions with the user. Refer to `proto/TAGS.md` for full syntax of each tag.

1. **Is this the primary key?**
   - Yes -> `sql:"pk,id,type(uuid)"` (one per table, except singletons)

2. **Should this field be searchable?**
   - Yes -> `search:"Display Name"` and register a `FieldLabel` in `pkg/search/options.go`
   - Should it be hidden from UI? -> add `,hidden`

3. **Does this field reference another table?**
   - Yes -> `sql:"fk(TypeName:field)"` and add `--references=storage.TypeName` to gen.go
   - Does it need a real SQL FK constraint? If not -> add `,no-fk-constraint`
   - Should it allow NULL? -> add `,allow-null`
   - Should it cascade delete? (default yes) If restrict -> add `,restrict-delete` (use sparingly)

4. **Should this field be indexed?**
   - Yes -> `sql:"index=btree"` (default), `brin` for time-series, `gin` for arrays. `hash` indexes have been observed to be expensive; prefer btree unless the user explicitly requests a hash index.
   - Part of a composite unique index? -> `sql:"index=name:my_index;category:unique"` (same name on all fields)

5. **Is this a repeated message field? Which storage strategy?**
   - Default (no tag) -> creates a child table with parent FK + `idx` column
   - `sql:"strategy(bytea)"` -> serializes as bytea in the parent table (no child table, no SQL-level access to individual elements)
   - Child table is best for queryable/joinable data. Bytea is best for opaque data that doesn't need SQL access.

6. **Should this field be a policy criteria?**
   - Yes -> `policy:"Display Name"`

7. **Is this a sensitive credential?**
   - Yes -> `scrub:"always"` for secrets, `scrub:"dependent"` for related fields like endpoints

8. **Should this field be excluded from hash computation?**
   - Yes (timestamps, computed scores, the hash field itself) -> `hash:"ignore"` or `sensorhash:"ignore"`

9. **Is this an endpoint URL that must not be localhost?**
   - Yes -> `validate:"nolocalendpoint"`

## Step 3: Architectural Decisions

When creating new protos with relationships, present these tradeoffs and ASK the user to choose.

### 3a. Relationship Modeling: Embedded vs FK

**Option 1: Embedded repeated field** (child in parent proto)
```protobuf
message Deployment {
  repeated Container containers = 11;
}
```
- Auto-generates a child table (`deployments_containers`) with parent FK + `idx` column.
- Parent's `serialized` column stores the full proto INCLUDING all children (data duplication).
- Each repeated message field = 1 child table. Nested repeated fields cascade further.
- Good when: children always accessed with parent, moderate cardinality, no independent lifecycle.
- Examples: `Deployment.containers`, `ImageV2.layers`.

**Option 2: Separate protos with FK** (child references parent)
```protobuf
message VirtualMachineScanV2 {
  string vm_v2_id = 2; // @gotags: sql:"fk(VirtualMachineV2:id),type(uuid),index=btree"
}
```
- Each proto has its own store, datastore, gen.go, search category.
- No data duplication in serialized blobs.
- Good when: children are top-level entities, queried independently, high cardinality.
- Examples: `VirtualMachineV2` -> `VirtualMachineScanV2` -> `VirtualMachineComponentV2`.

**Ask the user:**
> "Your proto has a relationship with `{child_type}`. Should it be an embedded repeated field (child table auto-generated, data duplicated in parent's serialized blob) or a separate top-level proto with its own store? Consider:
> 1. Does `{child_type}` have an independent lifecycle?
> 2. Will it be queried independently from the parent?
> 3. How many child rows per parent?
> 4. How many child tables will be generated?"

### 3b. Search Scope and BFS Join Concerns

Without `--search-scope`, the search framework BFS-traverses ALL connected schemas to resolve query fields.

**Potential issues:**
- If two connected schemas have the same search field name, BFS picks non-deterministically.
- Multiple paths between schemas can cause BFS to choose the wrong join path even with `--search-scope` restrictions, because scope operates at the table level, not field level (ROX-17252).
- Duplicate search fields are sometimes intentional (e.g., Container in Deployment has Image-related search tags that also exist in Image proto).

**Ask the user:**
> "Are there overlapping search fields with connected tables? Overlapping names are sometimes necessary for business logic, but be aware that `--search-scope` can only exclude entire tables — it cannot resolve ambiguity between tables that are both in scope. If overlapping names are intentional, verify that the BFS join path produces correct results for your queries."

### 3c. Serialized vs No-Serialized Store

**Option 1: Serialized (default)**
- A `serialized` bytea column stores the full proto blob alongside indexed columns.
- Reads deserialize from the blob — fast single-row retrieval, no need to scan every column.
- Data is duplicated between the blob and indexed columns.
- `sql:"-"` can exclude fields from columns (they're still in the blob).
- Mature, battle-tested path used by all existing stores.

**Option 2: No-Serialized (`--no-serialized`)**
- All proto fields become individual DB columns. No serialized blob.
- No data duplication. Every field is directly queryable via SQL.
- Generated store is `NoSerializedStore[T]` with column-based scan/insert.
- Supports child fetch control: `WithChildren()` / `WithoutChildren()` functional options.
- `sql:"-"` is disallowed (walker panics). Use `sql:"strategy(bytea)"` for repeated messages that don't need SQL access.
- Best for new tables where you want full SQL-level access to all fields.

**Ask the user:**
> "Should this store use the default serialized blob or the no-serialized mode?
> - Serialized: proven, fast single-row reads, allows `sql:\"-\"` exclusions.
> - No-serialized: no data duplication, full SQL access to all fields, child fetch control."

### 3d. API Proto vs Storage Proto

For v2 APIs, storage and API protos MUST be separate messages even if they look similar.

**Pattern:**
- `proto/storage/report_configuration.proto` -> storage representation
- `proto/api/v2/report_service.proto` -> API representation
- `central/reports/service/v2/convert.go` -> converter between them

**Ask the user:**
> "Is this for a v2 API? I'll create separate API and storage protos with a converter."

## Step 4: gen.go Creation

### Creating a New gen.go

Create the file at `<resource>/datastore/[internal/]store/postgres/gen.go`:

```go
package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TypeName [flags...]
```

Select flags based on the decisions above. Common patterns:

**Standard searchable store:**
```go
//go:generate pg-table-bindings-wrapper --type=storage.MyResource --search-category MY_RESOURCES
```

**With FK references:**
```go
//go:generate pg-table-bindings-wrapper --type=storage.MyResource --search-category MY_RESOURCES --references=storage.OtherType
```

**Schema-only (hand-written store):**
```go
//go:generate pg-table-bindings-wrapper --type=storage.MyResource --schema-only --search-category MY_RESOURCES
```

**No-serialized store (all fields as columns):**
```go
//go:generate pg-table-bindings-wrapper --type=storage.MyResource --no-serialized --search-category MY_RESOURCES
```

**Singleton (config/settings):**
```go
//go:generate pg-table-bindings-wrapper --type=storage.MyConfig --singleton
```

See `proto/TAGS.md` for the complete flag reference with all options.

### Critical Gotcha

- Every `fk(TypeName:field)` tag in proto REQUIRES `storage.TypeName` in gen.go `--references`. Missing it causes a **panic** at generation time.
- The reverse (`--references` without a matching `fk(...)`) is silently ignored -- it does nothing.

## Step 5: Registration Checklist

For new types, these files may need updating:

1. **`tools/generate-helpers/pg-table-bindings/list.go`**
   - Add `storage.TypeName` -> `resources.ResourceHandle` mapping in the `typeRegistry`.
   - Keep entries in lexicographic order.
   - Required for ALL new types.

2. **`proto/api/v1/search_service.proto`**
   - Add a new `SearchCategory` enum value.
   - Required only if the store is searchable (`--search-category`).

3. **`pkg/search/options.go`**
   - Add `FieldLabel` constants for each new `search:` tag value.
   - The label string must match the tag value exactly.

4. **`pkg/sac/resources/list.go`**
   - Add a new resource type if none of the existing ones fit.
   - Required only for new SAC resource types.

## Step 6: Generate and Verify

```bash
# 1. Regenerate proto Go code (if proto files changed)
make proto-generated-srcs

# 2. Run Postgres bindings generator
gogen -run pg-table-bindings <path/to/gen.go>  # gogen is from https://github.com/stackrox/workflow

# 3. Verify generated output
# - pkg/postgres/schema/<table_name>.go  -- schema definition
# - <resource>/datastore/store/postgres/store.go  -- store interface (if not --schema-only)
# - pkg/search/postgres/mapping/mapping.go  -- search registration (if searchable)
# - pkg/postgres/schema/all.go  -- table registration
```

## Step 7: Wire Compatibility

All proto changes must be wire-compatible and schema changes must be backwards-compatible:

- **Never** remove or renumber existing proto fields. Use `reserved` and `[deprecated = true]`.
- **Never** remove SQL columns. Add columns, don't remove.
- New columns tolerate their zero value until normal operation populates them — a migration is not needed immediately if that's acceptable.
- Migrations are needed when existing data must be backfilled or transformed for correct behavior. They can often be added later in the feature development cycle rather than upfront. See `migrator/README.md`.
- When deprecating a field: `string old_field = 5 [deprecated = true];` and add `reserved` for the field number if removing entirely later.
