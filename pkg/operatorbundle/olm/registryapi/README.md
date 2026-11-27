# registryapi

Vendored gRPC client stubs for the OLM operator catalog index (operator-registry's
`api.Registry` service).

These files are copied verbatim from
`github.com/operator-framework/operator-registry@v1.74.0/pkg/api`:

- `registry.pb.go`
- `registry_grpc.pb.go`
- `registry.proto` (source proto, for reference; not compiled)

Only the `package` declaration was changed (`api` → `registryapi`).

## Why copied instead of a module dependency

Importing `operator-registry/pkg/api` directly drags in its model-conversion helpers
(`api_to_model.go` / `model_to_api.go`), which depend on `github.com/operator-framework/api`.
That module requires `k8s.io/* v0.37.x`, which under MVS would bump this repo's pinned
`k8s.io/* v0.36.5` across the entire codebase.

The two generated files here import only `google.golang.org/grpc` and
`google.golang.org/protobuf` (already repo dependencies) plus the standard library, so copying
them avoids the k8s version bump entirely. Re-copy from the upstream tag if the proto changes.
