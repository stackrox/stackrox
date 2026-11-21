# gocache-serve

A GOCACHEPROG cache program: Go's build cache fronted by an Azure Blob
Store, with a tmpfs local tier, disk spill, size-gated prefetch bundles,
per-job manifests, and replica accounts for read locality.

**Full design, measurements, and history:** `/Volumes/dev/stack/content-addressed-build-plan.md`
(the plan document for the content-addressed build system) — read the
appendices for the bug archive (OOM classes, marker semantics, races).

## Subcommands

- (no args) — GOCACHEPROG protocol mode (stdin/stdout)
- `-store KEY FILE` / `-fetch KEY FILE` — single-account blob put/get
- `-store-all KEY FILE` / `-fetch-any KEY FILE` — replica set with
  content-hash markers (KEY.sha256); fetch verifies before serving
- `-stub BUILDID OUTFILE` — tiny ELF buildID stub (bootstrap self-test)
- `-verify`, `-doctor`, `-metrics-report`, `-metrics-push`,
  `-hash-manifest`, `-docker-push/pull/prefetch` — diagnostics and CI helpers

## Key invariants (do not break)

1. **Markers hash LOGICAL content** — the .sha256 marker hashes the
   decompressed payload; blobs are gzip at rest.
2. **GET verifies** — fetched entries are checked against the manifest's
   outputID (the content hash); mismatch = miss + recompile (self-heal).
3. **Never drop uploads** — a lost entry permanently caps the hit rate;
   queues block rather than drop.
4. **Large bodies stream** — whole-buffer handling of multi-GB tars
   OOM-killed runners three times before the streaming paths existed.
5. **Memory limit** — GOMEMLIMIT (default 1500MiB, env-tunable) keeps
   the heap a good citizen next to the tmpfs tier (which is RAM too).

## Env knobs

GOCACHE_SERVE_{BACKEND,CONTAINER,CONCURRENCY,MANIFEST,MANIFEST_WRITE,
RUN_ID,GOMEMLIMIT,SPILL,PFB,PFB_MAX_ENTRY,PFB_BUNDLE_BYTES},
AZURE_TENANT_ID/CLIENT_ID/CLIENT_SECRET.
