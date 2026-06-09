# Plan: Validate and regenerate code in roxctl-pac worktree

## Context

All source changes for `roxctl policy-config reconcile` were implemented in a previous worktree and copied here. The proto changes, Central service changes, search options, and roxctl command code are all present but:

1. **Proto-generated code is stale** — `proto/storage/policy.proto` was modified (added `config_scope` field) but generated Go code needs regeneration
2. **Postgres schema/store are hand-edited** — `pkg/postgres/schema/policies.go` and `central/policy/store/postgres/store.go` show manual edits that should be regenerated from the proto
3. **E2e test is in wrong location** — `roxctl/policy_config_reconcile_test.go` (package `tests`) should be at `tests/policy_config_reconcile_test.go`

## Steps

### 1. Regenerate proto-generated sources
```
make proto-generated-srcs
```
This regenerates `generated/storage/policy.pb.go` etc. from the modified proto.

### 2. Regenerate postgres schema and store
```
PATH="tools/generate-helpers:$PATH" go generate ./central/policy/store/postgres/...
```
This runs `pg-table-bindings` to regenerate the GORM model (`pkg/postgres/schema/policies.go`) and the store (`central/policy/store/postgres/store.go`) with the new `config_scope` column.

### 3. Move e2e test to correct location
Move `roxctl/policy_config_reconcile_test.go` → `tests/policy_config_reconcile_test.go`

### 4. Verify compilation
- `go build ./roxctl/...`
- `go build ./central/policy/...`
- `go test -tags test_e2e -c -o /dev/null ./tests/`

### 5. Run unit tests
- `go test ./roxctl/policyconfig/reconcile/...`

### 6. Run linter
- `make golangci-lint` (or targeted lint on changed packages)
