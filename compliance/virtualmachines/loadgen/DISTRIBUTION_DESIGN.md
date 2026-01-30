# Load Generator Distribution Design

## Problem Statement

Current loadgen uses uniform values:
- All VMs have exactly the same package count (e.g., 700)
- All VMs report at the same interval (e.g., 60s ± 5% jitter)

This doesn't match real-world scenarios where:
- Package counts vary significantly (P50: 500, P95: 1500, P99: 2500)
- Report intervals can be mixed (some VMs at 30s, some at 300s)
- Traffic can be bursty (VMs synchronized, not uniformly distributed)

## Proposed Solution

Introduce statistical distributions for package counts and report intervals.

### Option 1: Normal Distribution (Recommended for simplicity)

```yaml
loadgen:
  vmCount: 192

  # Package count distribution
  packages:
    mean: 700          # Target average packages per VM
    stddev: 300        # Standard deviation (controls spread)
    min: 100           # Floor (prevent unrealistic low values)
    max: 3000          # Ceiling (prevent unrealistic high values)

  # Report interval distribution
  reportInterval:
    mean: 60s          # Target average report interval
    stddev: 20s        # Standard deviation
    min: 30s           # Floor
    max: 300s          # Ceiling

  # Optional: backward compatibility
  # If 'packages' is an int instead of object, use uniform distribution
  # packages: 700      # All VMs get exactly 700 packages (old behavior)
```

**Pros:**
- Simple to understand (mean + spread)
- Mathematically sound (normal distribution is common in nature)
- Achieves target mean accurately
- Easy to configure and validate

**Cons:**
- Doesn't match exact percentile requirements from GAPS doc
- Need to tune stddev to get desired P95/P99 values

### Option 2: Percentile-Based Distribution

```yaml
loadgen:
  vmCount: 192

  # Package count distribution (matches GAPS recommendations exactly)
  packages:
    type: percentile
    buckets:
      - percent: 70    # 70% of VMs
        value: 500
      - percent: 20    # 20% of VMs
        value: 1000
      - percent: 8     # 8% of VMs
        value: 1500
      - percent: 2     # 2% of VMs
        value: 2000

  # Report interval distribution
  reportInterval:
    type: percentile
    buckets:
      - percent: 50    # 50% of VMs at 30s
        value: 30s
      - percent: 30    # 30% of VMs at 60s
        value: 60s
      - percent: 20    # 20% of VMs at 300s
        value: 300s
```

**Pros:**
- Exact control over distribution shape
- Matches GAPS document recommendations precisely
- Can model any distribution shape

**Cons:**
- More complex configuration
- User must specify all buckets (percentages must sum to 100)
- Less intuitive than mean/stddev

### Option 3: Hybrid (Flexible)

Support both approaches with type detection:

```yaml
# Simple mode (normal distribution)
packages:
  mean: 700
  stddev: 300
  min: 100
  max: 3000

# Or precise mode (percentile-based)
packages:
  type: percentile
  buckets:
    - {percent: 70, value: 500}
    - {percent: 20, value: 1000}
    - {percent: 10, value: 1500}
```

## Implementation Plan

### Phase 1: Normal Distribution (Recommended first step)

1. Extend config types to support distribution parameters
2. Add distribution sampling logic in `payload.go` and `simulator.go`
3. Maintain backward compatibility (detect if old config format is used)
4. Pre-compute distributions at startup for all VMs

### Phase 2: Percentile-Based (Optional enhancement)

1. Add percentile distribution type
2. Extend config parser to support both types

## Implementation Details

### Package Count Distribution

Currently: One generator with fixed package count
```go
generator := vmindexreport.NewGeneratorWithSeed(cfg.numPackages, 0)
```

Proposed: Multiple generators for different package counts
```go
// Sample package count for each VM from distribution
packageCounts := samplePackageDistribution(cfg.packages, vmCount, rng)

// Create generators (reuse same generator for VMs with same package count)
generators := make(map[int]*vmindexreport.Generator)
for _, count := range packageCounts {
    if _, exists := generators[count]; !exists {
        generators[count] = vmindexreport.NewGeneratorWithSeed(count, someSeed)
    }
}
```

### Report Interval Distribution

Currently: Fixed interval with ±5% jitter
```go
jitter := time.Duration(float64(cfg.reportInterval) * (rng.Float64()*0.1 - 0.05))
nextInterval := cfg.reportInterval + jitter
```

Proposed: Per-VM interval sampled from distribution
```go
type vmConfig struct {
    cid            uint32
    packageCount   int
    reportInterval time.Duration
}

// Sample report interval for each VM from distribution
vmConfigs := make([]vmConfig, vmCount)
for i := 0; i < vmCount; i++ {
    vmConfigs[i] = vmConfig{
        cid:            startCID + uint32(i),
        packageCount:   sampleFromDistribution(cfg.packages),
        reportInterval: sampleFromDistribution(cfg.reportInterval),
    }
}

// In simulateVM:
func simulateVM(ctx context.Context, vmCfg vmConfig, ...) {
    // Use vmCfg.reportInterval instead of cfg.reportInterval
}
```

## Example Configurations

### Realistic Production Load (based on GAPS doc)

```yaml
loadgen:
  vmCount: 200

  packages:
    mean: 800          # Average VM has 800 packages
    stddev: 400        # Wide variance
    min: 200           # Min realistic
    max: 2500          # P99 ~ 2500

  reportInterval:
    mean: 90s          # Average 90s
    stddev: 60s        # Some VMs much faster/slower
    min: 30s
    max: 300s
```

### Conservative (tight distribution, predictable)

```yaml
loadgen:
  vmCount: 200

  packages:
    mean: 700
    stddev: 100        # Tight clustering around mean
    min: 500
    max: 1000

  reportInterval:
    mean: 60s
    stddev: 10s        # All VMs report ~60s ± 20s
    min: 40s
    max: 80s
```

### Extreme Outliers (test P99 behavior)

```yaml
loadgen:
  vmCount: 200

  packages:
    mean: 700
    stddev: 600        # Very wide spread
    min: 100
    max: 3000          # Some VMs with 3000 packages

  reportInterval:
    mean: 60s
    stddev: 90s        # Huge variance
    min: 10s
    max: 300s
```

## Backward Compatibility

Detect old config format and use uniform distribution:

```yaml
# Old format (still supported)
loadgen:
  vmCount: 192
  numPackages: 700        # Integer = uniform distribution
  reportInterval: 60s     # Duration = uniform distribution

# New format
loadgen:
  vmCount: 192
  packages:               # Object = distribution
    mean: 700
    stddev: 300
  reportInterval:         # Object = distribution
    mean: 60s
    stddev: 20s
```

## Testing & Validation

After implementation, verify:

1. **Mean Accuracy**: Actual mean package count matches configured mean (±5%)
2. **Distribution Shape**: Plot histogram of package counts, verify normal-ish shape
3. **Min/Max Enforcement**: No VM has packages < min or > max
4. **Backward Compatibility**: Old config files still work
5. **Reproducibility**: Same config + seed = same distribution

## Metrics & Observability

Add new Prometheus metrics:

```
# Package count distribution
vm_package_count_p50
vm_package_count_p95
vm_package_count_p99
vm_package_count_mean

# Report interval distribution
vm_report_interval_p50
vm_report_interval_p95
vm_report_interval_p99
vm_report_interval_mean
```

Log distribution stats at startup:

```
INFO: Package distribution: mean=702 stddev=298 min=150 max=2890 p50=690 p95=1456 p99=2103
INFO: Report interval distribution: mean=61.2s stddev=19.8s min=32s max=287s p50=59s p95=95s p99=128s
```

## Recommendation

Start with **Option 1 (Normal Distribution)** because:
1. Simple to implement and configure
2. Mathematically sound
3. Achieves realistic load patterns
4. User suggested "mean + random distribution" which aligns perfectly
5. Can add percentile-based later if needed

The user can tune `stddev` to achieve desired P95/P99 values:
- stddev = 0 → uniform (all VMs identical)
- stddev = mean/3 → moderate spread (P95 ≈ mean + 1.65*stddev)
- stddev = mean/2 → wide spread (P99 can be 2-3x mean)
