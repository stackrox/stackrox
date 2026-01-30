# Option 2: Percentile-Based Distribution - Detailed Examples

## Configuration Format

```yaml
loadgen:
  vmCount: 200

  packages:
    type: percentile
    buckets:
      - percent: 70
        value: 500
      - percent: 20
        value: 1000
      - percent: 8
        value: 1500
      - percent: 2
        value: 2000

  reportInterval:
    type: percentile
    buckets:
      - percent: 50
        value: 30s
      - percent: 30
        value: 60s
      - percent: 20
        value: 300s
```

## What This Means

### Package Distribution (200 VMs)

| Bucket | Percent | Value | VMs | Description |
|--------|---------|-------|-----|-------------|
| 1 | 70% | 500 packages | 140 VMs | Typical small VMs |
| 2 | 20% | 1000 packages | 40 VMs | Medium VMs |
| 3 | 8% | 1500 packages | 16 VMs | Large VMs |
| 4 | 2% | 2000 packages | 4 VMs | Outliers (P99) |

**Result:**
- 140 VMs with exactly 500 packages
- 40 VMs with exactly 1000 packages
- 16 VMs with exactly 1500 packages
- 4 VMs with exactly 2000 packages

**Statistics:**
- Mean: 710 packages
- P50 (median): 500 packages (70th percentile falls in first bucket)
- P95: 1500 packages
- P99: 2000 packages

### Report Interval Distribution (200 VMs)

| Bucket | Percent | Value | VMs | Description |
|--------|---------|-------|-----|-------------|
| 1 | 50% | 30s | 100 VMs | Frequent reporters |
| 2 | 30% | 60s | 60 VMs | Normal reporters |
| 3 | 20% | 300s | 40 VMs | Infrequent reporters |

**Result:**
- 100 VMs report every 30 seconds
- 60 VMs report every 60 seconds
- 40 VMs report every 300 seconds

**Statistics:**
- Mean: ~84 seconds
- P50: 30s
- P95: 300s

## How It Works (Implementation)

### Step 1: Parse Configuration

```go
type PercentileBucket struct {
    Percent int
    Value   interface{} // int for packages, time.Duration for intervals
}

type PercentileDistribution struct {
    Type    string             // "percentile"
    Buckets []PercentileBucket
}

// Validation: buckets must sum to 100%
func (d *PercentileDistribution) Validate() error {
    sum := 0
    for _, bucket := range d.Buckets {
        sum += bucket.Percent
    }
    if sum != 100 {
        return fmt.Errorf("buckets must sum to 100%%, got %d%%", sum)
    }
    return nil
}
```

### Step 2: Assign Values to VMs

```go
// For 200 VMs with the example config:
func assignPackageCounts(vmCount int, dist PercentileDistribution) []int {
    counts := make([]int, vmCount)
    vmIndex := 0

    for _, bucket := range dist.Buckets {
        // Calculate how many VMs get this value
        numVMs := (vmCount * bucket.Percent) / 100

        // Assign the value to those VMs
        for i := 0; i < numVMs; i++ {
            counts[vmIndex] = bucket.Value.(int)
            vmIndex++
        }
    }

    // Handle remainder VMs (rounding errors)
    for vmIndex < vmCount {
        // Assign to last bucket
        counts[vmIndex] = dist.Buckets[len(dist.Buckets)-1].Value.(int)
        vmIndex++
    }

    return counts
}
```

### Step 3: Result for 200 VMs

```go
packageCounts := assignPackageCounts(200, packageDistribution)
// Result:
// packageCounts[0..139] = 500   (140 VMs, 70%)
// packageCounts[140..179] = 1000 (40 VMs, 20%)
// packageCounts[180..195] = 1500 (16 VMs, 8%)
// packageCounts[196..199] = 2000 (4 VMs, 2%)
```

## Comparison: Option 1 vs Option 2

### Same Scenario: 200 VMs

**Option 1 (Normal Distribution):**
```yaml
packages:
  mean: 700
  stddev: 350
  min: 100
  max: 2000
```

Result: Continuous distribution
- VM #1: 682 packages
- VM #2: 1023 packages
- VM #3: 451 packages
- VM #4: 889 packages
- VM #5: 1456 packages
- ...every VM has a different value (sampled from bell curve)

**Option 2 (Percentile):**
```yaml
packages:
  type: percentile
  buckets:
    - {percent: 70, value: 500}
    - {percent: 20, value: 1000}
    - {percent: 8, value: 1500}
    - {percent: 2, value: 2000}
```

Result: Discrete buckets
- VMs #1-140: exactly 500 packages
- VMs #141-180: exactly 1000 packages
- VMs #181-196: exactly 1500 packages
- VMs #197-200: exactly 2000 packages
- ...only 4 unique values across all VMs

## Real-World Examples

### Example 1: Match GAPS Document Recommendations Exactly

From GAPS doc section "Package Count Distribution (P50/P95/P99)":

```yaml
loadgen:
  vmCount: 1000

  packages:
    type: percentile
    buckets:
      - percent: 70    # P70: typical VMs
        value: 500
      - percent: 20    # P90: medium VMs
        value: 1000
      - percent: 8     # P98: high-package VMs
        value: 1500
      - percent: 2     # P100: extreme outliers
        value: 2000

  # Result for 1000 VMs:
  # - 700 VMs with 500 packages (P50 = 500)
  # - 200 VMs with 1000 packages
  # - 80 VMs with 1500 packages (P95 ≈ 1500)
  # - 20 VMs with 2000 packages (P99 = 2000)
```

### Example 2: Bursty Traffic Pattern

From GAPS doc section "Real-World Workload Patterns":

```yaml
loadgen:
  vmCount: 200

  reportInterval:
    type: percentile
    buckets:
      - percent: 50    # Half of VMs synchronized at :00
        value: 60s
        # Add synchronization flag (future enhancement)
        synchronized: true
        offset: 0s     # All report at :00

      - percent: 30    # 30% synchronized at :30
        value: 60s
        synchronized: true
        offset: 30s    # All report at :30

      - percent: 20    # 20% random (existing jitter behavior)
        value: 60s
        synchronized: false

  # Result:
  # - 100 VMs all report at :00, :60, :120, ... (traffic spike!)
  # - 60 VMs all report at :30, :90, :150, ... (traffic spike!)
  # - 40 VMs report randomly distributed (smooth load)
```

### Example 3: Simple Bimodal Distribution

Two types of VMs: small and large

```yaml
packages:
  type: percentile
  buckets:
    - percent: 80
      value: 400     # 80% are small VMs
    - percent: 20
      value: 2000    # 20% are large VMs

# With 100 VMs:
# - 80 VMs with 400 packages
# - 20 VMs with 2000 packages
# Mean: 720 packages (misleading - actual distribution is bimodal!)
# P50: 400, P95: 2000, P99: 2000
```

### Example 4: Multi-Cluster Heterogeneous

Different clusters have different workload profiles:

**Cluster A (Development):**
```yaml
packages:
  type: percentile
  buckets:
    - percent: 90
      value: 300     # Mostly minimal VMs
    - percent: 10
      value: 600

reportInterval:
  type: percentile
  buckets:
    - percent: 100
      value: 300s    # Infrequent reporting
```

**Cluster B (Production):**
```yaml
packages:
  type: percentile
  buckets:
    - percent: 50
      value: 800
    - percent: 30
      value: 1200
    - percent: 15
      value: 1800
    - percent: 5
      value: 2500

reportInterval:
  type: percentile
  buckets:
    - percent: 80
      value: 60s     # Frequent reporting
    - percent: 20
      value: 30s     # Very frequent
```

## Advantages of Option 2

1. **Exact Control:** You specify exactly what percentages get what values
2. **Matches GAPS Doc:** Can implement recommendations literally
3. **Predictable:** No randomness (deterministic assignment)
4. **Easy to Reason About:** "70% of VMs have 500 packages" is clear
5. **Testing Specific Scenarios:** Can test "what if 10% of VMs have 10x packages?"

## Disadvantages of Option 2

1. **More Configuration:** Must specify all buckets explicitly
2. **Percentages Must Sum to 100:** Validation overhead
3. **Discrete Values Only:** Can't get smooth continuous distribution
4. **Less Natural:** Real world has continuous variation, not discrete buckets
5. **Harder to Tweak:** Want slightly more variation? Must change all buckets

## When to Use Option 2

Use percentile-based when you:
- Want to test specific scenarios from GAPS doc exactly
- Need precise control over P95/P99 values
- Are modeling known workload profiles (from customer data)
- Want deterministic, reproducible distributions
- Need to create "adversarial" test cases (e.g., "what if 5% have 100x packages?")

**Don't use** if you just want "realistic variation" - Option 1 is simpler.

## Implementation Notes

### Assignment Strategy

Current design: sequential assignment
```go
// VMs 0-139: 500 packages
// VMs 140-179: 1000 packages
// etc.
```

**Alternative:** Random assignment
```go
// Shuffle VMs randomly before assigning
// This prevents "all small VMs report first" patterns

shuffle(vmIndices)
for _, vmIdx := range vmIndices {
    vmConfigs[vmIdx].packages = getNextBucketValue()
}
```

Random assignment is more realistic (VMs aren't ordered by size in production).

### Rounding Errors

With 200 VMs and 70% bucket:
- Expected: 200 * 0.70 = 140 VMs exactly
- Works perfectly

With 200 VMs and 8% bucket:
- Expected: 200 * 0.08 = 16 VMs exactly
- Works perfectly

With 197 VMs and 8% bucket:
- Expected: 197 * 0.08 = 15.76 VMs
- Round down: 15 VMs or round up: 16 VMs?
- Remainder VMs need assignment strategy

**Solution:** Assign remainders to later buckets or distribute proportionally.

## Hybrid Example

Could combine both approaches in one config:

```yaml
packages:
  type: percentile
  buckets:
    - percent: 70
      distribution:
        type: normal
        mean: 500
        stddev: 50    # 70% of VMs around 500 ± variation
    - percent: 20
      distribution:
        type: normal
        mean: 1000
        stddev: 100
    - percent: 10
      value: 2000     # Exact value (no variation)

# This gives:
# - 70% of VMs: normal distribution centered at 500
# - 20% of VMs: normal distribution centered at 1000
# - 10% of VMs: exactly 2000
```

This is probably overkill though!

## My Take

**Option 2 is powerful but overkill for most cases.**

Use Option 1 (normal distribution) unless you need:
1. Exact percentile matching from GAPS doc
2. Testing specific adversarial scenarios
3. Modeling known customer workload profiles

For "make the load more realistic", Option 1 is simpler and achieves the goal.
