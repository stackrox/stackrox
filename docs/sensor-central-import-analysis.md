# Sensor → Central Import Analysis

## Date: 2026-04-13

### sensor/kubernetes imports:

No central/ imports found (excluding pkg/).

```bash
$ go list -f '{{.Deps}}' ./sensor/kubernetes | tr ' ' '\n' | grep -E "^github.com/stackrox/rox/central/" | grep -v "/pkg/" | sort -u
(no output - clean)
```

### sensor/admission-control imports:

No central/ imports found (excluding pkg/).

```bash
$ go list -f '{{.Deps}}' ./sensor/admission-control | tr ' ' '\n' | grep -E "^github.com/stackrox/rox/central/" | grep -v "/pkg/" | sort -u
(no output - clean)
```

### Dependency chains:

N/A - no unwanted imports detected.

### Recommended actions:

✅ No action needed

Both sensor/kubernetes and sensor/admission-control have ZERO imports from central/* packages (excluding the shared pkg/* utilities). This is the desired state.

The sensor components are properly isolated from central-specific code and will not load central's heavy init() functions (compliance checks, GraphQL loaders, central metrics) when running in the busybox binary.

### Verification commands used:

```bash
# Check full dependency tree for central/* imports
go list -f '{{.Deps}}' ./sensor/kubernetes | tr ' ' '\n' | grep -E "^github.com/stackrox/rox/central/"
go list -f '{{.Deps}}' ./sensor/admission-control | tr ' ' '\n' | grep -E "^github.com/stackrox/rox/central/"

# Check direct imports
go list -f '{{.ImportPath}}: {{.Imports}}' ./sensor/kubernetes | tr ' ' '\n' | grep -E "central/"
go list -f '{{.ImportPath}}: {{.Imports}}' ./sensor/admission-control | tr ' ' '\n' | grep -E "central/"
```

All commands returned no results, confirming complete isolation.

### Impact on ROX-33958:

This clean import structure means sensor components will NOT trigger central's init() functions, which is critical for avoiding OOMKills in:
- admission-control (currently 6-7 restarts per replica under race detector)
- config-controller (currently 7 restarts under race detector)

The sensor/admission-control component already has its own metrics init() (migrated in Task 2.1), and sensor/kubernetes will have its own metrics init() (migrated in Task 4.1).
