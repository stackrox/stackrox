# Migration Guide: Distribution-Based Load Generation

This guide documents the new distribution-based configuration format and backward compatibility support.

## Configuration Formats

The load generator now supports **both** legacy scalar format and new distribution format for `packages` and `reportInterval`.

### Legacy Format (Still Supported)

```yaml
loadgen:
  numPackages: 700
  reportInterval: 60s
```

When using legacy scalar values, they are automatically converted to distributions with:
- `mean` = scalar value
- `stddev` = 0 (no variability)
- `min` = `max` = scalar value

### New Distribution Format (Recommended)

```yaml
loadgen:
  packages:
    mean: 700
    stddev: 300
    min: 100
    max: 3000
  reportInterval:
    mean: 60s
    stddev: 20s
    min: 30s
    max: 300s
```

### Mixed Format

You can use legacy format for one field and distribution format for the other:

```yaml
loadgen:
  numPackages: 700  # Legacy scalar
  reportInterval:   # Distribution format
    mean: 60s
    stddev: 20s
    min: 30s
    max: 300s
```

## Migration Steps

**No migration required!** Legacy scalar format is still fully supported. However, we recommend migrating to the distribution format to take advantage of per-VM variability for more realistic load testing.

### Duration Format Requirements

When using the **distribution format**, all duration fields in the `reportInterval` distribution **must be strings**:
- ✅ Correct: `mean: "60s"`, `stddev: "20s"`, `min: "30s"`, `max: "300s"`
- ❌ Incorrect: `mean: 60`, `stddev: 20` (numbers are not accepted)

When using the **legacy scalar format**, `reportInterval` must be a duration string:
- ✅ Correct: `reportInterval: "60s"`
- ❌ Incorrect: `reportInterval: 60` (numbers are not accepted)

Duration parsing errors will cause the load generator to exit immediately.

## Recommended Migration (Optional)

While legacy format is still supported, we recommend migrating to distribution format for more realistic load testing:

1. **Update your configuration file** to use distribution maps for `packages` and `reportInterval`:
   ```yaml
   loadgen:
     vmCount: 1000
     packages:
       mean: 700
       stddev: 300
       min: 100
       max: 3000
     reportInterval:
       mean: 60s
       stddev: 20s
       min: 30s
       max: 300s
     # ... other fields unchanged
   ```

2. **Choose appropriate distribution parameters**:
   - For `packages`: Set `mean` to your desired average package count, `stddev` for variability, and `min`/`max` to bound the distribution
   - For `reportInterval`: Set `mean` to your desired average interval, `stddev` for variability, and `min`/`max` to bound the distribution

3. **Test your configuration** before deploying to production:
   ```bash
   # Validate config parsing
   ./vsock-loadgen --config your-config.yaml
   ```

## Example Configurations

### Small Scale (100 VMs, Low Variability)
```yaml
loadgen:
  vmCount: 100
  packages:
    mean: 500
    stddev: 100
    min: 200
    max: 1000
  reportInterval:
    mean: 30s
    stddev: 5s
    min: 20s
    max: 60s
  statsInterval: 30s
  port: 818
  metricsPort: 9090
  requestTimeout: 10s
```

### Large Scale (10,000 VMs, High Variability)
```yaml
loadgen:
  vmCount: 10000
  packages:
    mean: 700
    stddev: 300
    min: 100
    max: 3000
  reportInterval:
    mean: 60s
    stddev: 20s
    min: 30s
    max: 300s
  statsInterval: 30s
  port: 818
  metricsPort: 9090
  requestTimeout: 10s
```

## Behavior Changes

### Deterministic Seeding

The load generator now supports deterministic VM configuration assignment via the `LOADGEN_SEED` environment variable:

```bash
export LOADGEN_SEED=12345
./vsock-loadgen --config config.yaml
```

If no seed is provided, the current time (nanoseconds) is used, ensuring different distributions on each run.

### Per-VM Variability

Each VM now gets its own package count and report interval sampled from the configured distributions. This provides more realistic load patterns compared to the previous uniform configuration.

### Metrics

New Prometheus metrics are available:
- `vsock_loadgen_packages_mean` - Mean package count across all VMs
- `vsock_loadgen_packages_p50` - P50 (median) package count
- `vsock_loadgen_packages_p95` - P95 package count
- `vsock_loadgen_packages_p99` - P99 package count
- `vsock_loadgen_interval_mean_seconds` - Mean report interval in seconds
- `vsock_loadgen_interval_p50_seconds` - P50 report interval in seconds
- `vsock_loadgen_interval_p95_seconds` - P95 report interval in seconds
- `vsock_loadgen_interval_p99_seconds` - P99 report interval in seconds

## Validation Rules

The load generator enforces the following validation rules:

1. **Packages distribution:**
   - `stddev >= 0`
   - `min >= 0` (values will be clamped to >= 1)
   - `max > min`
   - Warning if `mean` is outside `[min, max]`

2. **Report interval distribution:**
   - `stddev >= 0`
   - `min >= 0` (values will be clamped to >= 1s)
   - `max > min`
   - `min >= 1s` (enforced)
   - Warning if `mean` is outside `[min, max]`

3. **Duration parsing:**
   - All duration fields must be valid duration strings
   - Parsing errors cause immediate exit

## Troubleshooting

### Error: "packages must be a map with mean, stddev, min, max"
- **Cause**: `packages` field is present but not a valid map
- **Solution**: Either use legacy `numPackages` scalar or provide a valid `packages` map

### Error: "reportInterval must be a duration string or a map"
- **Cause**: `reportInterval` field has invalid type
- **Solution**: Use either legacy scalar format (`reportInterval: "60s"`) or distribution map format

### Error: "reportInterval.mean must be a duration string"
- **Cause**: Duration fields are specified as numbers instead of strings
- **Solution**: Use string format: `mean: "60s"` instead of `mean: 60`

### Error: "packages.max must be > packages.min"
- **Cause**: Invalid distribution bounds
- **Solution**: Ensure `max > min` for both distributions

## Questions?

If you encounter issues during migration, please check:
1. All required distribution fields are present
2. Duration fields are strings (e.g., `"60s"`, not `60`)
3. Distribution bounds are valid (`max > min`, `min >= 0`)
4. Interval minimum is at least 1 second

