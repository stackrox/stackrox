# Bundle Helpers

Go-based tools for managing operator bundle manifests.

## Tools

### fix-spec-descriptor-order
Sorts specDescriptors in CRDs and resolves field dependencies.

### patch-csv
Patches ClusterServiceVersion files with version updates, image replacements,
and related images configuration.

## Usage

```bash
go run ./main.go patch-csv [flags] < input.yaml > output.yaml
go run ./main.go fix-spec-descriptor-order < input.yaml > output.yaml
```

## Development

### Running tests:
```bash
cd operator/bundle_helpers
go test ./...
```

## Architecture

- `cmd/` - CLI command implementations
- `pkg/csv/` - CSV patching logic, version handling
- `pkg/descriptor/` - Descriptor sorting and field resolution
- `pkg/rewrite/` - String replacement utilities
- `pkg/values/` - Values handling

