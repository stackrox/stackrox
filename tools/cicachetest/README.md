# cicachetest

Fixture module for `.github/workflows/cache-go-dependencies-regression.yaml`.
It exists solely to reproduce, on every PR to the shared `cache-go-dependencies`
action and on a weekly schedule, the Go module-index `dirHash` staleness bug
that caused the golang-jwt v4->v5 incident (`could not import ... (open : no
such file or directory)`), plus to positively verify the GOCACHE build cache
and Go test cache are actually being hit after a restore, not just failing to
error.

**This module ships no product functionality and must never be imported by
product code.** It is an isolated Go module (own `go.mod`) specifically so it
never participates in the main module's `./...` build graph — `make style`,
`golangci-lint`, and `go build ./...` from the repo root never see it.

## Layout

- `variantfoo/`, `variantbar/` — two packages with equal-length names and
  identical `Value() int` functions. Never edited by the workflow.
- `consumer/` — imports `variantfoo` on disk. The workflow's `verify-restore`
  job sed-swaps this to `variantbar` (an equal-length identifier swap, so the
  file's byte size and mtime stay collision-compatible) to simulate a
  same-size import change restored against a stale cache.
- `testcache/` — a test that reads `testdata/fixture.txt` at runtime, used to
  verify Go's test-result cache (a separate `(size, mode, mtime)`-keyed
  mechanism from the module index) is hit across the same save/restore cycle.

See `cache-go-dependencies-regression.yaml` for how these are wired together.
