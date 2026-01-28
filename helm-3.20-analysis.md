# Helm 3.20.0 Breaking Change Analysis

## Summary

Helm 3.20.0 introduces a behavioral change in how `null` values are handled during value merging, which breaks the StackRox legacy translation layer used in Helm charts. The compatibility layer assumes `null` values remove keys, but Helm 3.20.0 now preserves them explicitly.

## Root Cause

### The Breaking Change

Helm 3.20.0 merged [PR #12879](https://github.com/helm/helm/pull/12879) (merged Jan 23, 2025) which modified `pkg/chartutil/coalesce.go` to fix subchart value overrides. The change adds explicit `null` preservation:

```go
for key, val := range dst {
    if val == nil {
        src[key] = nil
    }
}
```

### Behavior Comparison

| Scenario | Helm 3.19.4 | Helm 3.20.0 |
|----------|-------------|-------------|
| `clusterName: null` in values | Key is **removed** | Key exists with `nil` value |
| `kindIs "invalid" $value` check | Returns `true` | Returns `false` |
| Legacy value translation | ✅ Works | ❌ Conflict error |

## Why StackRox Helm Tests Fail

### The Compatibility Layer Logic

File: `image/templates/helm/stackrox-secured-cluster/templates/_compatibility.tpl:42-44`

```go
{{ $currVal := index $values $k }}
{{ if kindIs "invalid" $currVal }}
  {{ $_ := set $values $k $v }}  // Set value from legacy config
{{ else }}
  // Error: Conflict between legacy and new config
{{ end }}
```

### Failing Test Scenario

Test: `pkg/helm/charts/tests/securedclusterservices/testdata/helmtest/legacy-settings.test.yaml`

1. **Suite setup:** `clusterName: "testcluster"`
2. **Test override:** `clusterName: null` (to clear for legacy mode)
3. **Legacy config:** `cluster.name: "legacy-cluster"`

**Expected (3.19.4):**
- `clusterName: null` removes the key
- `kindIs "invalid"` returns `true`
- Legacy layer sets `clusterName` from `cluster.name` ✅

**Actual (3.20.0):**
- `clusterName: null` preserves `nil` value
- `kindIs "invalid"` returns `false`
- Legacy layer detects conflict and fails ❌

### Error Message

```
FATAL ERROR:
Conflict between legacy configuration values cluster.name and explicitly set
configuration value clusterName, please unset legacy value
```

## Solution

Update the compatibility layer to treat `nil` values the same as missing values:

```diff
  {{ $currVal := index $values $k }}
- {{ if kindIs "invalid" $currVal }}
+ {{ if or (kindIs "invalid" $currVal) (eq $currVal nil) }}
    {{ $_ := set $values $k $v }}
  {{ else if and (kindIs "map" $v) (kindIs "map" $currVal) }}
    {{ include "srox._mergeCompat" (list $currVal $v $compatValuePath (append $path $k)) }}
  {{ else }}
    {{ include "srox.fail" (printf "Conflict...") }}
  {{ end }}
```

This aligns with Helm 3.20.0's semantic where `null` explicitly means "unset this value."

## Impact

All helmtest tests that use the legacy translation layer fail with Helm 3.20.0, including:
- `legacy-settings.test.yaml` (secured cluster)
- Any test that sets a value to `null` to enable legacy mode compatibility

## References

- [Helm PR #12879 - Override subcharts with null values](https://github.com/helm/helm/pull/12879)
- [Helm PR #18627 - StackRox Helm version bump attempt](https://github.com/stackrox/stackrox/pull/18627)
- [Helm Issue #11737 - Difference when treating nullified nodes](https://github.com/helm/helm/issues/11737)
- [Helm v3.20.0 Release](https://github.com/helm/helm/releases/tag/v3.20.0)
