# Guide to Proto Tags and Postgres Stores

This document is the comprehensive reference for StackRox proto `@gotags:` annotations, `gen.go` generator flags, and common store generation scenarios. It covers all tag types used in `proto/storage/` and how they translate into generated SQL schemas, search indexes, and runtime behavior.

## Table of Contents

- [High-Level Overview](#high-level-overview)
- [Steps to Generate Postgres Store Layer](#steps-to-generate-postgres-store-layer)
- [Proto Tag Reference](#proto-tag-reference)
  - [SQL Tags](#sql-tags-sql)
  - [Search Tags](#search-tags-search)
  - [Policy Tags](#policy-tags-policy)
  - [Hash Tags](#hash-tags-hash)
  - [Sensor Hash Tags](#sensor-hash-tags-sensorhash)
  - [Scrub Tags](#scrub-tags-scrub)
  - [Validate Tags](#validate-tags-validate)
  - [crYaml Tags](#cryaml-tags-cryaml)
  - [Combining Multiple Tags](#combining-multiple-tags)
- [Generator Flags (gen.go)](#generator-flags-gengo)
- [Scenarios](#scenarios)
  - [Simple Table](#simple-table)
  - [Table with Unique Constraints](#table-with-unique-constraints)
  - [Table with Searchable Fields](#table-with-searchable-fields)
  - [Table with Foreign Keys](#table-with-foreign-keys)
  - [Nested Proto Messages](#nested-proto-messages)
  - [Excluding Proto Fields from SQL Table](#excluding-proto-fields-from-sql-table)
  - [Restricting Searching to Specific Tables](#restricting-searching-to-specific-tables)
  - [Generating Read-Only Store](#generating-read-only-store)
  - [No-Serialized Store](#no-serialized-store)
- [Proto Type to SQL Type Mapping](#proto-type-to-sql-type-mapping)
- [Notable Code References](#notable-code-references)

---

## High-Level Overview

The Postgres store layer primarily comprises two components:

- **Table Schema** -- defines the logical structure of a proto in SQL and Go.
- **Store** -- provides an interface to the underlying SQL table which stores the data.

A column is added in the generated table for each proto field which is tagged as a constraint or searchable. By default, for all top-level proto messages, the `serialized` column is added to the table. `serialized` records all bytes of the proto object.

When `--no-serialized` is used, **all** proto fields become individual DB columns and no `serialized` bytea blob is stored. This eliminates data duplication but requires every field to be scannable. See [No-Serialized Store](#no-serialized-store) for details.

---

## Steps to Generate Postgres Store Layer

Given your dev environment is already set up as described in the [README](https://github.com/stackrox/stackrox/blob/master/README.md#development):

1. Add required [proto tags](#proto-tag-reference) to the proto definition.
2. Execute `make proto-generated-srcs`.
3. Create a directory `<resource>/datastore/[internal/]store/postgres` (recommended package structure). For example, `image/datastore/store/postgres/`.
4. Add the Postgres store generate instruction `//go:generate pg-table-bindings-wrapper` with appropriate [flags](#generator-flags-gengo) to a file called `gen.go` (recommended filename) in the directory created above.
5. Execute `gogen -run pg-table-bindings <path_to_gen.go>`. This generates:
   - The SQL and Go schema at `pkg/postgres/schema/<table_name>.go`.
   - The table schema is registered (with the table name as the key) in `pkg/postgres/schema/all.go`.
   - If the table is associated with a search category, it is also registered (with the search category as key) in `pkg/search/postgres/mapping/mapping.go`.
   - The Postgres store at `<resource>/datastore/store/postgres/store.go` (unless `--schema-only`).

> **Note:** Not all proto changes require modifying the gen.go file. Adding a search tag, an index, or a new field without new FK/search-category changes only requires *executing* the existing gen.go, not changing it.

---

## Proto Tag Reference

Tags are added to proto fields using the `@gotags:` comment syntax:
```protobuf
string id = 1; // @gotags: search:"Alert ID" sql:"pk,type(uuid)"
```

### SQL Tags (`sql:""`)

Controls how proto fields map to SQL columns in the generated schema.

**Parsed by:** `pkg/postgres/walker/walker.go` (`getPostgresOptions()`)

| Option | Syntax | Description |
|--------|--------|-------------|
| Exclude | `sql:"-"` | Do not add the field and all its sub-fields as columns. Does not nullify the field in the serialized object. |
| Primary key | `sql:"pk"` | Adds PRIMARY KEY constraint. Required on all tables except singletons. Only one per table. |
| ID marker | `sql:"id"` | Marks field as the logical ID column. In practice redundant since single-PK tables (the only kind allowed) auto-infer the ID from the PK field. Commonly seen alongside `pk` but has no effect. |
| Unique | `sql:"unique"` | Adds UNIQUE constraint on the column. |
| Foreign key | `sql:"fk(TypeName:field)"` | References `field` in proto `TypeName`. Adds FK constraint with CASCADE delete. |
| No FK constraint | `sql:"no-fk-constraint"` | Used with `fk`. Creates search graph edge only, no SQL constraint. Useful for cross-table searching without FK overhead. |
| Restrict delete | `sql:"restrict-delete"` | Used with `fk`. Uses RESTRICT instead of CASCADE. **Use sparingly** -- can cause stuck records on rollback. |
| Allow null | `sql:"allow-null"` | FK relationship can be NULL. |
| Directional | `sql:"directional"` | Used with `fk`. Creates one-way edge in search graph (no reverse traversal). |
| Column type | `sql:"type(X)"` | Override inferred SQL type. Common values: `uuid`, `timestamptz`, `cidr`. |
| Index (default) | `sql:"index"` | Creates B-tree index on the column. |
| Index (typed) | `sql:"index=btree"` | Explicit B-tree index. Also supports `hash`, `gin`, `brin`. |
| Named index | `sql:"index=name:idx_name"` | Composite index. All fields sharing the same `name` become columns in one multi-column index. Add `;category:unique` to make it a unique constraint. |
| Ignore child PKs | `sql:"ignore_pk"` | Ignore PK tags from sub-fields of an embedded message. |
| Ignore child uniques | `sql:"ignore_unique"` | Ignore UNIQUE tags from sub-fields of an embedded message. |
| Ignore child FKs | `sql:"ignore-fks"` | Ignore FK tags from sub-fields. |
| Ignore child indexes | `sql:"ignore-index"` | Ignore index tags from sub-fields. |
| Ignore search labels | `sql:"ignore_labels(Label1,Label2)"` | Skip specific search labels from embedded message fields. |
| Repeated strategy | `sql:"strategy(bytea)"` | For repeated message fields: store as a serialized bytea blob in the parent table instead of creating a child table. Valid values: `bytea`, `child_table` (default). See [No-Serialized Store](#no-serialized-store). |

> **Warning:** Multiple primary keys are highly discouraged. Use composite primary keys via `postgres.IDFromPks(...)` (`pkg/search/postgres/utils.go`) instead.

> **Warning:** `sql:"-"` is incompatible with `--no-serialized`. Without a serialized blob, excluded fields would be silently lost on read. The walker panics if it encounters `sql:"-"` in a no-serialized schema. Use `sql:"strategy(bytea)"` for repeated messages you want to keep in the parent table.

**Examples:**
```protobuf
// Primary key with UUID type
string id = 1; // @gotags: sql:"pk,id,type(uuid)"

// Foreign key with search support, no SQL constraint, indexed
string deployment_id = 8; // @gotags: search:"Deployment ID" sql:"fk(Deployment:id),no-fk-constraint,index=btree,type(uuid)"

// BRIN index for time-series data
google.protobuf.Timestamp last_seen_timestamp = 2; // @gotags: sql:"index=brin"

// Named composite unique index (all fields sharing the name form one index)
string auth_provider_id = 1; // @gotags: sql:"index=category:unique;name:groups_unique_indicator"
string key = 2;              // @gotags: sql:"index=category:unique;name:groups_unique_indicator"
string value = 3;            // @gotags: sql:"index=category:unique;name:groups_unique_indicator"

// Embedded entity with ignored PK and search labels
Policy policy = 2; // @gotags: sql:"ignore_pk,ignore_unique,ignore_labels(Lifecycle Stage)"

// Repeated message stored as bytea in parent table (no child table created)
repeated Annotation annotations = 16; // @gotags: sql:"strategy(bytea)"
```

---

### Search Tags (`search:""`)

Controls field indexing and searchability. Fields tagged with `search` get a corresponding column in the generated table.

**Parsed by:** `pkg/search/walker.go` (`getSearchField()`)

| Syntax | Description |
|--------|-------------|
| `search:"Field Name"` | Searchable and visible in UI autocomplete. |
| `search:"Field Name,hidden"` | Searchable but hidden from UI autocomplete. |
| `search:"-"` | Exclude this field and all children from search. |

> **Important:** Each `search:"Field Name"` requires a matching `FieldLabel` constant registered in `pkg/search/options.go`. The label string must match exactly.

**Examples:**
```protobuf
// Searchable, visible in UI
string name = 2; // @gotags: search:"Deployment"

// Searchable but hidden
string cluster_id = 3; // @gotags: search:"Cluster ID,hidden" sql:"type(uuid)"

// Hidden field for sorting
string SORT_name = 16; // @gotags: search:"SORT_Policy,hidden"
```

---

### Policy Tags (`policy:""`)

Maps proto fields to boolean policy evaluation criteria. When a field has both `policy` and `search` tags, the `policy` tag takes precedence for policy field mapping.

**Parsed by:** `pkg/booleanpolicy/evaluator/pathutil/augmented_obj_meta.go` (`parsePolicyTag()`)

| Syntax | Description |
|--------|-------------|
| `policy:"Display Name"` | Available as policy criteria with the given display name. |
| `policy:",ignore"` | Exclude from policy evaluation (note leading comma). |
| `policy:",prefer-parent"` | If field exists in both child and parent, prefer the parent's value. |

**Examples:**
```protobuf
// Policy criteria field
bool allow_privilege_escalation = 7; // @gotags: policy:"Allow Privilege Escalation"

// Excluded from policy evaluation
float risk_score = 29; // @gotags: search:"Deployment Risk Score,hidden" policy:",ignore"

// Prefer parent value
string image_name = 5; // @gotags: policy:",prefer-parent"
```

---

### Hash Tags (`hash:""`)

Controls which fields are included when computing a hash for deduplication in Central. Uses the `hashstructure` library.

**Used by:** `pkg/sensor/hash/hasher.go` and other hash computation code

| Syntax | Description |
|--------|-------------|
| `hash:"ignore"` | Exclude from hash computation (e.g., timestamps, computed scores, the hash field itself). |
| `hash:"set"` | Treat repeated field as a set (order-independent hashing). |

**Examples:**
```protobuf
// Excluded from hash -- changes to these don't indicate a meaningful update
uint64 hash = 12;                          // @gotags: hash:"ignore"
google.protobuf.Timestamp last_updated = 9; // @gotags: hash:"ignore"

// Order-independent hashing for repeated fields
repeated Note notes = 10; // @gotags: hash:"set"
```

---

### Sensor Hash Tags (`sensorhash:""`)

Controls which fields are included when Sensor hashes events for deduplication before sending to Central. Semantically identical to `hash` but uses a separate tag name so the two can be configured independently.

**Parsed by:** `pkg/sensor/hash/hasher.go` (uses `hashstructure` with `TagName: "sensorhash"`)

| Syntax | Description |
|--------|-------------|
| `sensorhash:"ignore"` | Exclude from sensor hash (e.g., server-assigned IDs, timestamps set by Central). |
| `sensorhash:"set"` | Treat repeated field as a set. |

**Examples:**
```protobuf
// Server-assigned fields excluded from sensor-side hashing
string id = 1;                              // @gotags: sensorhash:"ignore" sql:"pk,type(uuid)"
google.protobuf.Timestamp time = 7;         // @gotags: sensorhash:"ignore"
map<string, string> annotations = 14;       // @gotags: sensorhash:"ignore"
```

---

### Scrub Tags (`scrub:""`)

Marks fields containing sensitive data for redaction in API responses. When an object is scrubbed, tagged fields are replaced with `"******"`.

**Parsed by:** `pkg/secrets/scrub.go` (`ScrubSecretsFromStructWithReplacement()`)

**Scrubbing** (API responses — `ScrubSecretsFromStructWithReplacement`):

| Syntax | Description |
|--------|-------------|
| `scrub:"always"` | Replaces field value with `"******"` in API responses (passwords, API keys, tokens). |
| `scrub:"map-values"` | Replaces values of hardcoded sensitive keys (currently only `client_secret`) with `"******"` in `map<string,string>` fields. |

**Reconciliation** (updates — `ReconcileScrubbedStructWithExisting`):

These tags do NOT scrub anything. They control what happens when a user submits an update to an object that had scrubbed fields.

| Syntax | Description |
|--------|-------------|
| `scrub:"always"` | On update, the user's value (empty or `"******"`) is replaced with the existing stored credential. Prevents credential loss on update. |
| `scrub:"dependent"` | On update, if this field differs from the stored value while credential fields (`scrub:"always"`) remain empty or scrubbed (`"******"`), the reconciler rejects the update with "credentials required". Prevents credential exfiltration via endpoint/username redirection. |
| `scrub:"disableDependentIfTrue"` | If this bool field is `true`, `dependent` reconciliation is skipped. Used when authentication is disabled (e.g., `allow_unauthenticated_smtp`), so there are no credentials to protect. |

**Examples:**
```protobuf
// Always scrub credentials
string secret_access_key = 4; // @gotags: scrub:"always"
string password = 2;          // @gotags: scrub:"always"

// Dependent on whether credentials are present
string endpoint = 7;          // @gotags: scrub:"dependent" validate:"nolocalendpoint"

// Disable dependent scrubbing when true (e.g., no auth needed)
bool allow_unauthenticated_smtp = 9; // @gotags: scrub:"disableDependentIfTrue"

// Scrub specific keys in map values
map<string, string> config = 6; // @gotags: scrub:"map-values"
```

---

### Validate Tags (`validate:""`)

Marks fields for input validation rules.

**Parsed by:** `pkg/endpoints/validate.go` (`ValidateEndpoints()`)

| Syntax | Description |
|--------|-------------|
| `validate:"nolocalendpoint"` | Rejects localhost, loopback addresses (`127.0.0.1`), and cloud metadata service URLs (`169.254.169.254`, `metadata.google.internal`). |

**Examples:**
```protobuf
string endpoint = 6; // @gotags: scrub:"dependent" validate:"nolocalendpoint"
string url = 1;       // @gotags: scrub:"dependent" validate:"nolocalendpoint"
```

---

### crYaml Tags (`crYaml:""`)

Controls YAML serialization for the config-as-code feature. **Not related to Postgres store generation.** Only used on `Policy` and `Scope` protos.

**Parsed by:** `tools/generate-helpers/config-as-code-helper/`

| Syntax | Description |
|--------|-------------|
| `crYaml:"-"` | Exclude from config-as-code YAML. |
| `crYaml:"fieldName"` | Override YAML field name. |
| `crYaml:",omitempty"` | Omit field if empty. |
| `crYaml:",stringer"` | Serialize enum fields as human-readable string names instead of numeric values (e.g., `MEDIUM_SEVERITY` instead of `2`). |
| `crYaml:",timestamp"` | Serialize `google.protobuf.Timestamp` fields as string instead of the proto struct. |

Options can be combined: `crYaml:"fieldName,omitempty,stringer"`

**Examples:**
```protobuf
string id = 1;   // @gotags: sql:"pk" crYaml:"-"
string name = 2; // @gotags: crYaml:"policyName"
string cluster = 1; // @gotags: crYaml:",omitempty"
repeated LifecycleStage lifecycle_stages = 9; // @gotags: crYaml:"lifecycleStages,stringer"
google.protobuf.Timestamp expiration = 6;     // @gotags: crYaml:",timestamp,omitempty"
```

---

### Combining Multiple Tags

Multiple tag types can be combined on a single field, space-separated:

```protobuf
// Search + SQL + sensor hash
string id = 1; // @gotags: search:"Alert ID" sensorhash:"ignore" sql:"pk,type(uuid)"

// Scrub + validate
string endpoint = 7; // @gotags: scrub:"dependent" validate:"nolocalendpoint"

// Search + policy + SQL
string deployment_id = 2; // @gotags: search:"Deployment ID,hidden" policy:",prefer-parent" sql:"index,fk(Deployment:id),no-fk-constraint,type(uuid)"

// Search + SQL + crYaml
string name = 2; // @gotags: search:"Policy" sql:"unique" crYaml:"policyName"

// Search + hash + SQL
float risk_score = 29; // @gotags: search:"Deployment Risk Score,hidden" policy:",ignore" sql:"index=btree" hash:"ignore"
```

---

## Generator Flags (gen.go)

The `gen.go` file contains a `//go:generate` directive that invokes the `pg-table-bindings-wrapper` with configuration flags. The wrapper auto-injects `--schema-directory`.

**File structure:**
```go
package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TypeName [flags...]
```

### When gen.go Changes are Needed

- Creating a new store
- Adding a new FK relationship (`--references`)
- Adding a new search category (`--search-category`)
- Changing search scope (`--search-scope`)
- Changing store behavior (singleton, cached, read-only)

### When gen.go Changes are NOT Needed (but Execution IS)

- Adding/removing `search:` tags on fields
- Adding/removing `sql:` index tags
- Adding/removing non-SQL tags (hash, scrub, policy, etc.)
- Adding new fields without FK or search category changes

### Flag Reference

**Source:** `tools/generate-helpers/pg-table-bindings/main.go`

| Flag | Description | Default | Required |
|------|-------------|---------|----------|
| `--type` | The Go name of the proto type (e.g., `storage.Deployment`) | | yes |
| `--table` | Custom table name | pluralized lower `snake_case` of type | no |
| `--registered-type` | The proto registry name when it differs from the Go struct name (e.g., proto `K8sRole` vs Go `K8SRole`). Used only for proto registry lookup. | same as `--type` | no |
| `--search-category` | Search category enum name to index under. Required if any `search:` tags exist. | | no |
| `--references` | Additional FK references, comma-separated: `<[table_name:]type>` | | no |
| `--search-scope` | Restrict search joins to these categories, comma-separated | all connected | no |
| `--no-serialized` | All proto fields become individual DB columns with no serialized bytea blob. Generates a `NoSerializedStore[T]` with column-based scan/insert. `sql:"-"` is disallowed. | `false` | no |
| `--schema-only` | Generate only the schema, not store and index | `false` | no |
| `--read-only-store` | Generate a read-only store (no write methods) | `false` | no |
| `--singleton` | Single-record store (no PK required) | `false` | no |
| `--cached-store` | Mirror store in memory cache. **Low-cardinality only.** | `false` | no |
| `--for-sac` | SAC-optimized methods. Only with `--cached-store`. | `false` | no |
| `--no-copy-from` | Disable Postgres COPY FROM optimization | `false` | no |
| `--get-all-func` | Generate GetAll() function | `false` | no |
| `--default-sort` | Default sort field (e.g., `search.DeploymentPriority.String()`) | | no |
| `--reverse-default-sort` | Reverse the default sort direction | `false` | no |
| `--transform-sort-options` | Sort option transform map (e.g., `DeploymentsSchema.OptionsMap`) | | no |
| `--feature-flag` | Gate schema registration behind a feature flag | | no |
| `--cycle` | Handle self-referential FK cycles. Nils out the self-referencing field in generated tests to avoid FK violations. Pass the Go field name (e.g., `EmbeddedCollections`). | | no |
| `--generate-data-model-helpers` | Generate CreateTableAndNewStore/Destroy helpers. Only used for the generator's own test protos — real stores get their tables created via `ApplyAllSchemas` in `pgtest.ForT()`. | `false` | no |

### Gotchas

- **`fk(...)` without `--references`**: The generator **panics**. Every `fk(TypeName:field)` in proto requires its `storage.TypeName` in `--references`.
- **`--references` without `fk(...)`**: Silently ignored. The referenced schema is loaded but never used since no field references it.
- **Referenced schema must exist first**: Generate the referenced table's schema before the referencing table's schema.

### Examples

**Simple store:**
```go
//go:generate pg-table-bindings-wrapper --type=storage.Secret --search-category SECRETS --default-sort search.CreatedTime.String()
```

**Schema-only (hand-written store):**
```go
//go:generate pg-table-bindings-wrapper --type=storage.ImageV2 --table=images_v2 --search-category IMAGES_V2 --schema-only --search-scope IMAGE_VULNERABILITIES_V2,IMAGE_COMPONENTS_V2,IMAGES_V2,DEPLOYMENTS,NAMESPACES,CLUSTERS --feature-flag FlattenImageData
```

**Singleton:**
```go
//go:generate pg-table-bindings-wrapper --type=storage.Config --singleton
```

**Cached store with SAC:**
```go
//go:generate pg-table-bindings-wrapper --type=storage.Cluster --cached-store --for-sac --search-category CLUSTERS --no-copy-from --default-sort search.Cluster.String()
```

**No-serialized store (all fields as columns):**
```go
//go:generate pg-table-bindings-wrapper --type=storage.MyResource --no-serialized --search-category MY_RESOURCES
```

**Read-only edge table with references:**
```go
//go:generate pg-table-bindings-wrapper --type=storage.NodeComponentCVEEdge --table=node_components_cves_edges --search-category NODE_COMPONENT_CVE_EDGE --references=node_components:storage.NodeComponent,node_cves:storage.NodeCVE --read-only-store --search-scope NODE_VULNERABILITIES,NODE_COMPONENT_CVE_EDGE,NODE_COMPONENTS,NODE_COMPONENT_EDGE,NODES,CLUSTERS
```

**Many-to-many edge table:**
```go
//go:generate pg-table-bindings-wrapper --type=storage.PolicyCategoryEdge --table=policy_category_edges --search-category POLICY_CATEGORY_EDGE --references=policies:storage.Policy,policy_categories:storage.PolicyCategory --search-scope POLICY_CATEGORY_EDGE,POLICY_CATEGORIES
```

---

## Scenarios

### Simple Table

To generate a table with a primary key, set the `sql` tag to `pk`. A primary key is required on all tables, except for singleton tables. When there is a conflict between incoming and stored rows, we default to updating the stored row.

> **Warning:** Adding multiple primary keys is highly discouraged and support was discontinued. Instead, create a composite primary key using `postgres.IDFromPks(...)` (`pkg/search/postgres/utils.go`).

**Proto:**
```protobuf
message Namespace {
  string id = 1; // @gotags: sql:"pk,id,type(uuid)"
  string name = 2;
}
```

**gen.go:**
```go
//go:generate pg-table-bindings-wrapper --type=storage.Namespace
```

**Generated SQL table:**
```sql
create table if not exists namespaces (
  Id varchar,
  serialized bytea,
  PRIMARY KEY(Id)
)
```

By default, the table name is the pluralized proto message name. To use a custom table name, specify it with `--table`:

```go
//go:generate pg-table-bindings-wrapper --type=storage.Namespace --table=kube_namespaces
```

### Table with Unique Constraints

**Proto:**
```protobuf
message Policy {
  string id = 1; // @gotags: sql:"pk,id,type(uuid)"
  string name = 2; // @gotags: sql:"unique"
}
```

**Generated SQL table:**
```sql
create table if not exists policies (
  Id varchar,
  Name varchar UNIQUE,
  serialized bytea,
  PRIMARY KEY(Id)
)
```

### Table with Searchable Fields

Labeling a proto field as searchable adds a corresponding column to the table.

**Proto:**
```protobuf
message Namespace {
  string id = 1; // @gotags: sql:"pk,id,type(uuid)"
  string name = 2; // @gotags: search:"Namespace Name"
}
```

**Generated SQL table:**
```sql
create table if not exists namespaces (
  Id varchar,
  Name varchar,
  serialized bytea,
  PRIMARY KEY(Id)
)
```

### Table with Foreign Keys

Use foreign key constraints when referential integrity is needed.

> **Important:**
> 1. The schema of the referenced table must exist first.
> 2. Use foreign keys sensibly as they have overhead for referential integrity checks.
> 3. Use great caution with `restrict-delete`. If a rollback is needed, records could get stuck. Preferred alternatives: use cascade delete with code-level existence checks, or use `no-fk-constraint` to create search-only references.

#### One-to-One Relationship

```protobuf
message NamespaceSummary {
  string id = 1; // @gotags: sql:"fk(Namespace:id),type(uuid)"
  int32 num_deployments = 2;
}
```

```go
//go:generate pg-table-bindings-wrapper --type=storage.NamespaceSummary --references=storage.Namespace
```

**Generated SQL table:**
```sql
create table if not exists namespace_summaries (
  Id varchar,
  serialized bytea,
  PRIMARY KEY(Id),
  CONSTRAINT fk_parent_table_0 FOREIGN KEY (Id) REFERENCES namespaces(Id) ON DELETE CASCADE
)
```

#### One-to-Many Relationship

```protobuf
message Deployment {
  string id = 1; // @gotags: sql:"pk,id,type(uuid)"
  string namespace_id = 3; // @gotags: sql:"fk(Namespace:id),type(uuid)"
}
```

```go
//go:generate pg-table-bindings-wrapper --type=storage.Deployment --references=storage.Namespace
```

#### Many-to-Many Relationship

Use a connector/edge proto message:

```protobuf
message PolicyCategoryEdge {
  string id = 1; // @gotags: sql:"pk,id,type(uuid)"
  string policy_id = 2; // @gotags: sql:"fk(Policy:id),type(uuid)"
  string category_id = 3; // @gotags: sql:"fk(PolicyCategory:id),type(uuid)"
}
```

```go
//go:generate pg-table-bindings-wrapper --type=storage.PolicyCategoryEdge --references=storage.Policy,storage.PolicyCategory
```

### Nested Proto Messages

**1-to-1 embedded:** Fields are flattened into the parent's table with a prefix.

```protobuf
message Namespace {
  string id = 1; // @gotags: sql:"pk,id,type(uuid)"
  NamespaceSummary summary = 3;
}

message NamespaceSummary {
  int32 num_deployments = 2; // @gotags: search:"Deployment Count"
}
```

**Generated SQL table:**
```sql
create table if not exists namespaces (
  Id varchar,
  NamespaceSummary_NumDeployments int,
  serialized bytea,
  PRIMARY KEY(Id)
)
```

**n-to-1 embedded (repeated):** A separate child table with foreign key is created automatically.

```protobuf
message Namespace {
  string id = 1; // @gotags: sql:"pk,id,type(uuid)"
  repeated RiskFactor risk_factors = 3;
}

message RiskFactor {
  string message = 1;
  int32 score = 3; // @gotags: search:"Risk Score"
}
```

**Generated SQL tables:**
```sql
-- serialized column holds the complete Namespace including all risk factors.
create table if not exists namespaces (
  Id varchar,
  serialized bytea,
  PRIMARY KEY(Id)
)

-- idx column tracks position in the repeated field.
create table if not exists namespaces_risk_factors (
  Namespace_Id varchar,
  idx int,
  Score int,
  PRIMARY KEY(Namespace_Id, idx),
  CONSTRAINT fk_parent_table_0 FOREIGN KEY (Namespace_Id) REFERENCES namespaces(Id) ON DELETE CASCADE
)
```

> **Note:** The parent's `serialized` column stores the full proto including all repeated children. This means data is duplicated between the parent's serialized blob and the child table columns. This is intentional -- the child table columns enable SQL queries/joins while the serialized blob enables efficient full-object retrieval.

### Excluding Proto Fields from SQL Table

#### Exclude All Fields

Use `sql:"-"` to exclude a field and its sub-fields from table columns:

```protobuf
message Namespace {
  string id = 1; // @gotags: sql:"pk,id,type(uuid)"
  NamespaceSummary summary = 3; // @gotags: sql:"-"
}
```

#### Exclude Constraints

Use `ignore_pk`, `ignore_unique`, or `ignore-fks` to exclude specific constraints from embedded sub-fields:

```protobuf
message Namespace {
  string id = 1; // @gotags: sql:"pk,id,type(uuid)"
  NamespaceSummary summary = 3; // @gotags: sql:"ignore_pk"
}
```

### Restricting Searching to Specific Tables

The search framework uses BFS to find connected tables that can resolve query fields. By setting `--search-scope`, you restrict which connected tables participate in joins.

This is important when connected tables have overlapping search field names, which can cause non-deterministic query behavior.

> **Caveat:** `--search-scope` operates at the table level, not the field level. If two tables within the scope can both resolve the same search field, BFS still picks non-deterministically. Keep this risk in mind when designing schemas with overlapping search fields across connected tables.

> **Example of ambiguous BFS join paths:** Suppose table A connects to table B and table C, and both B and C have a field tagged `search:"Cluster ID"`. A query on A filtering by `Cluster ID` could join through either B or C — BFS picks whichever it finds first, which may not be the correct one for the intended business logic. `--search-scope` can exclude B or C entirely, but if both are needed in scope for other fields, the ambiguity remains. Overlapping search field names across connected tables are sometimes necessary, but verify that the BFS join path produces correct query results.

### Generating Read-Only Store

Use `--read-only-store` for tables whose data is derived from other tables:

```go
//go:generate pg-table-bindings-wrapper --type=storage.ImageComponentEdge --read-only-store
```

When data in a table is derived from another, write records from the primary object's store to avoid races. For embedded 1-to-n relationships, the generator handles transactional upserts. For synthetically attached foreign keys, this must be handled manually.

### No-Serialized Store

Use `--no-serialized` to generate a store where every proto field is stored as an individual database column. No `serialized` bytea blob is written. The generated store type is `NoSerializedStore[T]`.

> **Important:** `sql:"-"` is disallowed in no-serialized schemas — without a blob, excluded fields would be lost. The walker panics if it encounters this.

**Proto:**
```protobuf
message MyResource {
  string id = 1; // @gotags: search:"Resource ID" sql:"pk,type(uuid)"
  string name = 2; // @gotags: search:"Resource Name"
  google.protobuf.Timestamp created_at = 3; // @gotags: sql:"type(timestamptz)"

  // Repeated message — child table (default strategy)
  message Label {
    string key = 1;
    string value = 2;
  }
  repeated Label labels = 4;

  // Repeated message — bytea in parent table (no child table)
  message Annotation {
    string key = 1;
    string value = 2;
  }
  repeated Annotation annotations = 5; // @gotags: sql:"strategy(bytea)"
}
```

**gen.go:**
```go
//go:generate pg-table-bindings-wrapper --type=storage.MyResource --no-serialized --search-category MY_RESOURCES
```

**Generated SQL tables:**
```sql
create table if not exists my_resources (
  Id uuid,
  Name varchar,
  CreatedAt timestamptz,
  Annotations bytea,
  PRIMARY KEY(Id)
)

create table if not exists my_resources_labels (
  MyResources_Id uuid,
  idx int,
  Key varchar,
  Value varchar,
  PRIMARY KEY(MyResources_Id, idx),
  CONSTRAINT fk_parent_table_0 FOREIGN KEY (MyResources_Id) REFERENCES my_resources(Id) ON DELETE CASCADE
)
```

Repeated message fields default to a child table. Use `sql:"strategy(bytea)"` to store them as a blob in the parent table instead (useful when the data doesn't need SQL-level access).

No-serialized stores support `GetWithOptions(ctx, id, pgSearch.WithoutChildren())` and `WalkByQueryWithOptions` to skip child table fetches for performance. `strategy(bytea)` fields are in the parent table and are always returned regardless.

---

## Proto Type to SQL Type Mapping

Default mappings (overridable with `sql:"type(X)"`):

| Proto Type | SQL Type | Go Model Type |
|-----------|----------|---------------|
| `string` | `varchar` | `string` |
| `bool` | `bool` | `bool` |
| `bytes` | `bytea` | `[]byte` |
| `float` / `double` | `numeric` | `float32` / `float64` |
| `int32` | `integer` | `int32` |
| `uint32` / `int64` | `bigint` | `uint32` / `int64` |
| `uint64` | `numeric` | `uint64` |
| `enum` | `integer` | `int32` |
| `google.protobuf.Timestamp` | `timestamp` | `*time.Time` |
| `repeated string` | `text[]` | `*pq.StringArray` |
| `repeated int32` / `repeated enum` | `int[]` | `*pq.Int32Array` |
| `map<string, string>` | `jsonb` | `map[string]string` |
| `repeated Message` (with `strategy(bytea)`) | `bytea` | `[]byte` (length-prefixed proto messages) |

Common type overrides:
- `type(uuid)` -- forces UUID type for string fields storing UUIDs
- `type(timestamptz)` -- timestamp with timezone (vs default `timestamp`)
- `type(cidr)` -- CIDR type for IP ranges

**Source:** `pkg/postgres/types.go`

---

## Notable Code References

- `tools/generate-helpers/pg-table-bindings/` -- code generator (main.go, templates, flag definitions)
- `tools/generate-helpers/pg-table-bindings/list.go` -- type-to-resource mapping registry (must be updated for new types)
- `pkg/postgres/walker/walker.go` -- SQL tag parsing and schema walking
- `pkg/search/walker.go` -- search tag parsing
- `pkg/postgres/schema/` -- generated schema files
- `pkg/search/options.go` -- search field label registration
- `pkg/search/postgres/mapping/` -- search category to table mapping
- `pkg/search/postgres/joins.go` -- BFS join resolver
- `pkg/booleanpolicy/evaluator/pathutil/augmented_obj_meta.go` -- policy tag parsing
- `pkg/secrets/scrub.go` -- scrub tag implementation
- `pkg/endpoints/validate.go` -- validate tag implementation
- `pkg/sensor/hash/hasher.go` -- hash/sensorhash tag usage
- `pkg/search/postgres/no_serialized_store.go` -- NoSerializedStore implementation, FetchOption, WithChildren/WithoutChildren
- `pkg/postgres/pgutils/messagebytes.go` -- strategy(bytea) marshal/unmarshal helpers
