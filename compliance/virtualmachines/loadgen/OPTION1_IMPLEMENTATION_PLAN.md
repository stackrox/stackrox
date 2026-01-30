# Option 1: Normal Distribution - Implementation Plan

## Overview

Implement normal (Gaussian) distribution for package counts and report intervals to create realistic load patterns. This is the simpler approach compared to percentile-based distributions.

## Goal

Transform from:
```yaml
numPackages: 700        # All VMs get exactly 700
reportInterval: 60s     # All VMs get exactly 60s
```

To:
```yaml
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

With backward compatibility (old format still works).

## Implementation Phases

### Phase 1: Config Schema & Parsing
### Phase 2: Distribution Sampling
### Phase 3: Payload Generation
### Phase 4: VM Simulation
### Phase 5: Observability
### Phase 6: Testing

---

## Phase 1: Config Schema & Parsing

### Task 1.1: Define New Config Types

**File:** `config.go`

Add new types for distribution configuration:

```go
// distributionConfig represents a statistical distribution.
// Supports both simple values (backward compatibility) and distribution parameters.
type distributionConfig struct {
    // Simple mode (backward compatibility)
    simpleValue interface{} // int or time.Duration

    // Distribution mode
    mean   float64
    stddev float64
    min    float64
    max    float64

    isDistribution bool // true if distribution mode, false if simple mode
}

// packageDistribution wraps distribution config for package counts
type packageDistribution struct {
    distributionConfig
}

// intervalDistribution wraps distribution config for time intervals
type intervalDistribution struct {
    distributionConfig
}
```

### Task 1.2: Update YAML Config Structure

**File:** `config.go`

Update `yamlConfig` to support both formats:

```go
type yamlConfig struct {
    Loadgen struct {
        VmCount     int    `yaml:"vmCount"`
        StatsInterval string `yaml:"statsInterval"`
        Port        uint   `yaml:"port"`
        MetricsPort int    `yaml:"metricsPort"`
        RequestTimeout string `yaml:"requestTimeout,omitempty"`
        Duration    string `yaml:"duration,omitempty"`

        // Backward compatible: simple values
        NumPackages    int    `yaml:"numPackages,omitempty"`
        ReportInterval string `yaml:"reportInterval,omitempty"`

        // New: distribution configs
        Packages       *yamlDistribution `yaml:"packages,omitempty"`
        ReportIntervalDist *yamlDistribution `yaml:"reportInterval,omitempty"`
    } `yaml:"loadgen"`
}

// yamlDistribution represents the YAML structure for distributions
type yamlDistribution struct {
    Mean   interface{} `yaml:"mean"`   // Can be int or string (for duration)
    Stddev interface{} `yaml:"stddev"` // Can be int or string (for duration)
    Min    interface{} `yaml:"min"`
    Max    interface{} `yaml:"max"`
}
```

### Task 1.3: Implement Custom YAML Unmarshaling

**File:** `config.go`

Handle both simple and distribution formats:

```go
func parsePackageConfig(yamlCfg *yamlConfig) packageDistribution {
    // Check new format first
    if yamlCfg.Loadgen.Packages != nil {
        return parsePackageDistribution(yamlCfg.Loadgen.Packages)
    }

    // Fall back to old format (backward compatibility)
    if yamlCfg.Loadgen.NumPackages > 0 {
        return packageDistribution{
            distributionConfig: distributionConfig{
                simpleValue:    yamlCfg.Loadgen.NumPackages,
                isDistribution: false,
            },
        }
    }

    // Default
    return packageDistribution{
        distributionConfig: distributionConfig{
            simpleValue:    700, // default
            isDistribution: false,
        },
    }
}

func parsePackageDistribution(yaml *yamlDistribution) packageDistribution {
    mean := parseIntOrDefault(yaml.Mean, 700)
    stddev := parseIntOrDefault(yaml.Stddev, 0)
    min := parseIntOrDefault(yaml.Min, 100)
    max := parseIntOrDefault(yaml.Max, 10000)

    return packageDistribution{
        distributionConfig: distributionConfig{
            mean:           float64(mean),
            stddev:         float64(stddev),
            min:            float64(min),
            max:            float64(max),
            isDistribution: true,
        },
    }
}

func parseIntervalConfig(yamlCfg *yamlConfig) intervalDistribution {
    // Similar logic for report intervals
    // ... (see parsePackageConfig for pattern)
}
```

### Task 1.4: Update Main Config Struct

**File:** `config.go`

Replace simple fields with distribution configs:

```go
type config struct {
    vmCount  int
    duration time.Duration

    // Old: numPackages int
    // New:
    packages packageDistribution

    // Old: reportInterval time.Duration
    // New:
    reportInterval intervalDistribution

    port           uint
    metricsPort    int
    statsInterval  time.Duration
    requestTimeout time.Duration
}
```

### Task 1.5: Validation

**File:** `config.go`

Add validation for distribution parameters:

```go
func validateDistribution(dist distributionConfig, name string) error {
    if !dist.isDistribution {
        return nil // Simple mode, no validation needed
    }

    if dist.stddev < 0 {
        return fmt.Errorf("%s: stddev must be >= 0, got %f", name, dist.stddev)
    }

    if dist.min < 0 {
        return fmt.Errorf("%s: min must be >= 0, got %f", name, dist.min)
    }

    if dist.max <= dist.min {
        return fmt.Errorf("%s: max (%f) must be > min (%f)", name, dist.max, dist.min)
    }

    if dist.mean < dist.min || dist.mean > dist.max {
        log.Warnf("%s: mean (%f) is outside [min, max] range [%f, %f]",
            name, dist.mean, dist.min, dist.max)
    }

    return nil
}

func validateConfig(cfg config) {
    if cfg.vmCount <= 0 {
        log.Error("vmCount must be > 0")
        os.Exit(1)
    }

    if err := validateDistribution(cfg.packages.distributionConfig, "packages"); err != nil {
        log.Error(err.Error())
        os.Exit(1)
    }

    if err := validateDistribution(cfg.reportInterval.distributionConfig, "reportInterval"); err != nil {
        log.Error(err.Error())
        os.Exit(1)
    }

    // ... existing validations
}
```

**Estimated Effort:** 4-6 hours
**Risk:** Medium (YAML unmarshaling can be tricky)

---

## Phase 2: Distribution Sampling

### Task 2.1: Normal Distribution Sampler

**File:** `distribution.go` (new file)

Implement Box-Muller transform for normal distribution:

```go
package main

import (
    "math"
    "math/rand"
)

// normalSample generates a sample from a normal distribution N(mean, stddev²)
// using the Box-Muller transform.
func normalSample(rng *rand.Rand, mean, stddev float64) float64 {
    // Box-Muller transform
    u1 := rng.Float64()
    u2 := rng.Float64()

    // Avoid log(0)
    if u1 < 1e-10 {
        u1 = 1e-10
    }

    z := math.Sqrt(-2.0*math.Log(u1)) * math.Cos(2.0*math.Pi*u2)
    return mean + stddev*z
}

// sampleFromDistribution samples a value from a distribution config.
// Clamps the result to [min, max] bounds.
func sampleFromDistribution(rng *rand.Rand, dist distributionConfig) float64 {
    if !dist.isDistribution {
        // Simple mode: return the simple value
        switch v := dist.simpleValue.(type) {
        case int:
            return float64(v)
        case float64:
            return v
        default:
            panic(fmt.Sprintf("unexpected simple value type: %T", v))
        }
    }

    // Distribution mode: sample from normal distribution
    if dist.stddev == 0 {
        // No variation, return mean
        return dist.mean
    }

    value := normalSample(rng, dist.mean, dist.stddev)

    // Clamp to [min, max]
    if value < dist.min {
        value = dist.min
    }
    if value > dist.max {
        value = dist.max
    }

    return value
}

// samplePackageCount samples a package count from the distribution.
func samplePackageCount(rng *rand.Rand, dist packageDistribution) int {
    value := sampleFromDistribution(rng, dist.distributionConfig)
    return int(math.Round(value))
}

// sampleReportInterval samples a report interval from the distribution.
func sampleReportInterval(rng *rand.Rand, dist intervalDistribution) time.Duration {
    value := sampleFromDistribution(rng, dist.distributionConfig)
    return time.Duration(math.Round(value))
}
```

### Task 2.2: VM Configuration Assignment

**File:** `distribution.go`

Create per-VM configurations:

```go
// vmConfig holds configuration for a single VM.
type vmConfig struct {
    cid            uint32
    packageCount   int
    reportInterval time.Duration
}

// assignVMConfigs creates per-VM configurations by sampling from distributions.
// Uses a deterministic seed for reproducibility.
func assignVMConfigs(vmCount int, startCID uint32, pkgDist packageDistribution, intervalDist intervalDistribution, seed int64) []vmConfig {
    rng := rand.New(rand.NewSource(seed))
    configs := make([]vmConfig, vmCount)

    for i := 0; i < vmCount; i++ {
        configs[i] = vmConfig{
            cid:            startCID + uint32(i),
            packageCount:   samplePackageCount(rng, pkgDist),
            reportInterval: sampleReportInterval(rng, intervalDist),
        }
    }

    return configs
}
```

### Task 2.3: Distribution Statistics

**File:** `distribution.go`

Calculate statistics for logging:

```go
// distributionStats holds statistical summary of a distribution.
type distributionStats struct {
    mean   float64
    stddev float64
    min    float64
    max    float64
    p50    float64
    p95    float64
    p99    float64
}

// computeStats calculates statistics from a sample of values.
func computeStats(values []float64) distributionStats {
    if len(values) == 0 {
        return distributionStats{}
    }

    // Sort for percentile calculations
    sorted := make([]float64, len(values))
    copy(sorted, values)
    sort.Float64s(sorted)

    // Calculate mean
    sum := 0.0
    for _, v := range values {
        sum += v
    }
    mean := sum / float64(len(values))

    // Calculate stddev
    variance := 0.0
    for _, v := range values {
        diff := v - mean
        variance += diff * diff
    }
    stddev := math.Sqrt(variance / float64(len(values)))

    // Percentiles
    p50 := sorted[len(sorted)*50/100]
    p95 := sorted[len(sorted)*95/100]
    p99 := sorted[len(sorted)*99/100]

    return distributionStats{
        mean:   mean,
        stddev: stddev,
        min:    sorted[0],
        max:    sorted[len(sorted)-1],
        p50:    p50,
        p95:    p95,
        p99:    p99,
    }
}

// logDistributionStats logs statistics about VM configurations.
func logDistributionStats(configs []vmConfig) {
    if len(configs) == 0 {
        return
    }

    // Extract package counts
    pkgCounts := make([]float64, len(configs))
    for i, cfg := range configs {
        pkgCounts[i] = float64(cfg.packageCount)
    }
    pkgStats := computeStats(pkgCounts)

    // Extract report intervals (in seconds)
    intervals := make([]float64, len(configs))
    for i, cfg := range configs {
        intervals[i] = cfg.reportInterval.Seconds()
    }
    intervalStats := computeStats(intervals)

    log.Infof("Package distribution: mean=%.0f stddev=%.0f min=%.0f max=%.0f p50=%.0f p95=%.0f p99=%.0f",
        pkgStats.mean, pkgStats.stddev, pkgStats.min, pkgStats.max,
        pkgStats.p50, pkgStats.p95, pkgStats.p99)

    log.Infof("Report interval distribution: mean=%.1fs stddev=%.1fs min=%.1fs max=%.1fs p50=%.1fs p95=%.1fs p99=%.1fs",
        intervalStats.mean, intervalStats.stddev, intervalStats.min, intervalStats.max,
        intervalStats.p50, intervalStats.p95, intervalStats.p99)
}
```

**Estimated Effort:** 3-4 hours
**Risk:** Low (standard statistics algorithms)

---

## Phase 3: Payload Generation Refactoring

### Task 3.1: Multi-Generator Support

**File:** `payload.go`

Change from single generator to multiple generators:

```go
// payloadProvider provides pre-generated and pre-marshaled payloads for each CID.
type payloadProvider struct {
    payloads map[uint32][]byte
}

// newPayloadProvider creates payloads for VMs with varying package counts.
func newPayloadProvider(vmConfigs []vmConfig) (*payloadProvider, error) {
    if len(vmConfigs) == 0 {
        return nil, fmt.Errorf("no VM configurations provided")
    }

    log.Infof("pre-generating %d unique reports with varying package counts...", len(vmConfigs))
    start := time.Now()

    // Group VMs by package count to reuse generators
    pkgCountToCIDs := make(map[int][]uint32)
    for _, cfg := range vmConfigs {
        pkgCountToCIDs[cfg.packageCount] = append(pkgCountToCIDs[cfg.packageCount], cfg.cid)
    }

    log.Infof("found %d unique package count values across %d VMs", len(pkgCountToCIDs), len(vmConfigs))

    payloads := make(map[uint32][]byte)

    // Create one generator per unique package count
    for pkgCount, cids := range pkgCountToCIDs {
        // Use package count as seed for reproducibility
        generator := vmindexreport.NewGeneratorWithSeed(pkgCount, int64(pkgCount))

        log.Infof("generating %d reports with %d packages (CIDs: %d VMs)",
            len(cids), pkgCount, len(cids))

        for _, cid := range cids {
            report := generator.GenerateV1IndexReport(cid)
            data, err := proto.Marshal(report)
            if err != nil {
                return nil, fmt.Errorf("marshal report for CID %d: %w", cid, err)
            }
            payloads[cid] = data
        }
    }

    log.Infof("pre-generated %d unique reports in %s", len(payloads), time.Since(start))
    return &payloadProvider{payloads: payloads}, nil
}

// get remains the same
func (p *payloadProvider) get(cid uint32) ([]byte, error) {
    payload, ok := p.payloads[cid]
    if !ok {
        return nil, fmt.Errorf("CID %d not in pre-generated range", cid)
    }
    return payload, nil
}
```

### Task 3.2: Remove Old createReportGenerator

**File:** `config.go`

Delete the old `createReportGenerator` function (no longer needed):

```go
// DELETE THIS:
// func createReportGenerator(cfg config) (*vmindexreport.Generator, error) {
//     generator := vmindexreport.NewGeneratorWithSeed(cfg.numPackages, 0)
//     ...
// }
```

**Estimated Effort:** 2 hours
**Risk:** Low (straightforward refactoring)

---

## Phase 4: VM Simulation Updates

### Task 4.1: Update simulateVM Signature

**File:** `simulator.go`

Change to accept per-VM config instead of global config:

```go
// simulateVM simulates a single VM sending index reports periodically.
// Uses the vmConfig for per-VM settings (package count, report interval).
func simulateVM(ctx context.Context, vmCfg vmConfig, globalCfg config, provider *payloadProvider, stats *statsCollector, metrics *metricsRegistry) {
    payload, err := provider.get(vmCfg.cid)
    if err != nil {
        log.Errorf("VM[%d]: failed to get payload: %v", vmCfg.cid, err)
        return
    }

    rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(vmCfg.cid)))

    // Stagger VM starts with random initial delay [0, reportInterval)
    // Use per-VM report interval
    initialDelay := time.Duration(rng.Int63n(int64(vmCfg.reportInterval)))
    select {
    case <-ctx.Done():
        return
    case <-time.After(initialDelay):
    }

    sendVMReport(vmCfg.cid, payload, globalCfg.port, globalCfg.requestTimeout, stats, metrics)

    for {
        // Add ±5% jitter to per-VM report interval
        jitter := time.Duration(float64(vmCfg.reportInterval) * (rng.Float64()*0.1 - 0.05))
        nextInterval := vmCfg.reportInterval + jitter

        select {
        case <-ctx.Done():
            return
        case <-time.After(nextInterval):
            sendVMReport(vmCfg.cid, payload, globalCfg.port, globalCfg.requestTimeout, stats, metrics)
        }
    }
}
```

**Estimated Effort:** 1 hour
**Risk:** Low

---

## Phase 5: Main Loop Integration

### Task 5.1: Update main.go

**File:** `main.go`

Wire everything together:

```go
func main() {
    cfg := parseConfig()

    ctx, cancel := context.WithCancel(context.Background())
    if cfg.duration > 0 {
        ctx, cancel = context.WithTimeout(ctx, cfg.duration)
    }
    defer cancel()

    setupSignalHandler(cancel)

    nodeName := os.Getenv("NODE_NAME")
    if nodeName == "" {
        log.Error("NODE_NAME environment variable not set")
        os.Exit(1)
    }

    cidInfo, err := calculateCIDRange(ctx, nodeName, cfg.vmCount)
    if err != nil {
        log.Errorf("calculating CID range: %v", err)
        os.Exit(1)
    }

    log.Infof("Node %s (index %d/%d) assigned CID range [%d-%d] for %d VMs (total cluster: %d VMs)",
        nodeName, cidInfo.NodeIndex, cidInfo.TotalNodes, cidInfo.StartCID, cidInfo.EndCID, cidInfo.VMsThisNode, cfg.vmCount)

    // NEW: Assign per-VM configurations
    vmConfigs := assignVMConfigs(cidInfo.VMsThisNode, cidInfo.StartCID, cfg.packages, cfg.reportInterval, 12345)

    // Log distribution statistics
    logDistributionStats(vmConfigs)

    // NEW: Create payload provider with VM configs
    payloads, err := newPayloadProvider(vmConfigs)
    if err != nil {
        log.Errorf("creating payload provider: %v", err)
        os.Exit(1)
    }

    stats := newStatsCollector()
    metrics := newMetricsRegistry()

    if cfg.metricsPort > 0 {
        go serveMetrics(ctx, cfg.metricsPort)
    }

    var wg sync.WaitGroup
    for _, vmCfg := range vmConfigs {
        wg.Add(1)
        go func(vmCfg vmConfig) {
            defer wg.Done()
            simulateVM(ctx, vmCfg, cfg, payloads, stats, metrics)
        }(vmCfg)
    }

    log.Infof("vsock-loadgen starting: vms=%d duration=%s cid-range=[%d-%d] port=%d",
        cidInfo.VMsThisNode, cfg.duration, cidInfo.StartCID, cidInfo.EndCID, cfg.port)

    start := time.Now()
    ticker := time.NewTicker(cfg.statsInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            wg.Wait()
            logSnapshot("final", stats.snapshot(time.Since(start)))
            return
        case <-ticker.C:
            logSnapshot("progress", stats.snapshot(time.Since(start)))
        }
    }
}
```

**Estimated Effort:** 1 hour
**Risk:** Low

---

## Phase 6: Observability

### Task 6.1: Prometheus Metrics

**File:** `metrics.go`

Add new metrics for distribution statistics:

```go
var (
    // Existing metrics...

    // New: Package count distribution
    vmPackageCountMean = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "vm_package_count_mean",
        Help: "Mean package count across all VMs",
    })
    vmPackageCountP50 = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "vm_package_count_p50",
        Help: "P50 (median) package count across all VMs",
    })
    vmPackageCountP95 = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "vm_package_count_p95",
        Help: "P95 package count across all VMs",
    })
    vmPackageCountP99 = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "vm_package_count_p99",
        Help: "P99 package count across all VMs",
    })

    // Report interval distribution
    vmReportIntervalMean = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "vm_report_interval_mean_seconds",
        Help: "Mean report interval across all VMs in seconds",
    })
    vmReportIntervalP50 = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "vm_report_interval_p50_seconds",
        Help: "P50 (median) report interval across all VMs in seconds",
    })
    vmReportIntervalP95 = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "vm_report_interval_p95_seconds",
        Help: "P95 report interval across all VMs in seconds",
    })
    vmReportIntervalP99 = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "vm_report_interval_p99_seconds",
        Help: "P99 report interval across all VMs in seconds",
    })
)

// recordDistributionMetrics sets Prometheus metrics based on VM configs.
func recordDistributionMetrics(configs []vmConfig) {
    if len(configs) == 0 {
        return
    }

    // Extract package counts
    pkgCounts := make([]float64, len(configs))
    for i, cfg := range configs {
        pkgCounts[i] = float64(cfg.packageCount)
    }
    pkgStats := computeStats(pkgCounts)

    vmPackageCountMean.Set(pkgStats.mean)
    vmPackageCountP50.Set(pkgStats.p50)
    vmPackageCountP95.Set(pkgStats.p95)
    vmPackageCountP99.Set(pkgStats.p99)

    // Extract report intervals (in seconds)
    intervals := make([]float64, len(configs))
    for i, cfg := range configs {
        intervals[i] = cfg.reportInterval.Seconds()
    }
    intervalStats := computeStats(intervals)

    vmReportIntervalMean.Set(intervalStats.mean)
    vmReportIntervalP50.Set(intervalStats.p50)
    vmReportIntervalP95.Set(intervalStats.p95)
    vmReportIntervalP99.Set(intervalStats.p99)
}
```

### Task 6.2: Call Metrics Recording

**File:** `main.go`

Add metric recording after creating VM configs:

```go
// In main() after assignVMConfigs:
vmConfigs := assignVMConfigs(...)
logDistributionStats(vmConfigs)
recordDistributionMetrics(vmConfigs)  // NEW
```

**Estimated Effort:** 1-2 hours
**Risk:** Low

---

## Phase 7: Testing

### Task 7.1: Unit Tests for Distribution Sampling

**File:** `distribution_test.go` (new file)

```go
package main

import (
    "math"
    "math/rand"
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestNormalSample(t *testing.T) {
    rng := rand.New(rand.NewSource(12345))
    samples := 10000

    values := make([]float64, samples)
    for i := 0; i < samples; i++ {
        values[i] = normalSample(rng, 100.0, 15.0)
    }

    stats := computeStats(values)

    // Mean should be close to 100
    assert.InDelta(t, 100.0, stats.mean, 1.0, "mean should be ~100")

    // Stddev should be close to 15
    assert.InDelta(t, 15.0, stats.stddev, 1.0, "stddev should be ~15")
}

func TestSampleFromDistribution_Clamping(t *testing.T) {
    rng := rand.New(rand.NewSource(12345))

    dist := distributionConfig{
        mean:           100,
        stddev:         50,
        min:            80,
        max:            120,
        isDistribution: true,
    }

    samples := 1000
    for i := 0; i < samples; i++ {
        value := sampleFromDistribution(rng, dist)
        assert.GreaterOrEqual(t, value, 80.0, "value should be >= min")
        assert.LessOrEqual(t, value, 120.0, "value should be <= max")
    }
}

func TestSampleFromDistribution_SimpleMode(t *testing.T) {
    rng := rand.New(rand.NewSource(12345))

    dist := distributionConfig{
        simpleValue:    700,
        isDistribution: false,
    }

    value := sampleFromDistribution(rng, dist)
    assert.Equal(t, 700.0, value, "simple mode should return exact value")
}

func TestSamplePackageCount(t *testing.T) {
    rng := rand.New(rand.NewSource(12345))

    dist := packageDistribution{
        distributionConfig: distributionConfig{
            mean:           700,
            stddev:         100,
            min:            500,
            max:            1000,
            isDistribution: true,
        },
    }

    count := samplePackageCount(rng, dist)
    assert.GreaterOrEqual(t, count, 500)
    assert.LessOrEqual(t, count, 1000)
}

func TestAssignVMConfigs(t *testing.T) {
    pkgDist := packageDistribution{
        distributionConfig: distributionConfig{
            mean:           700,
            stddev:         100,
            min:            500,
            max:            1000,
            isDistribution: true,
        },
    }

    intervalDist := intervalDistribution{
        distributionConfig: distributionConfig{
            mean:           60.0,
            stddev:         10.0,
            min:            30.0,
            max:            120.0,
            isDistribution: true,
        },
    }

    configs := assignVMConfigs(100, 1000, pkgDist, intervalDist, 12345)

    assert.Len(t, configs, 100)

    // Check CID assignment
    for i, cfg := range configs {
        assert.Equal(t, uint32(1000+i), cfg.cid)
    }

    // Check package counts are in range
    for _, cfg := range configs {
        assert.GreaterOrEqual(t, cfg.packageCount, 500)
        assert.LessOrEqual(t, cfg.packageCount, 1000)
    }

    // Check report intervals are in range
    for _, cfg := range configs {
        assert.GreaterOrEqual(t, cfg.reportInterval.Seconds(), 30.0)
        assert.LessOrEqual(t, cfg.reportInterval.Seconds(), 120.0)
    }
}

func TestAssignVMConfigs_Reproducibility(t *testing.T) {
    pkgDist := packageDistribution{
        distributionConfig: distributionConfig{
            mean:           700,
            stddev:         100,
            min:            500,
            max:            1000,
            isDistribution: true,
        },
    }

    intervalDist := intervalDistribution{
        distributionConfig: distributionConfig{
            mean:           60.0,
            stddev:         10.0,
            min:            30.0,
            max:            120.0,
            isDistribution: true,
        },
    }

    configs1 := assignVMConfigs(50, 1000, pkgDist, intervalDist, 99999)
    configs2 := assignVMConfigs(50, 1000, pkgDist, intervalDist, 99999)

    // Same seed should produce identical results
    for i := 0; i < 50; i++ {
        assert.Equal(t, configs1[i].cid, configs2[i].cid)
        assert.Equal(t, configs1[i].packageCount, configs2[i].packageCount)
        assert.Equal(t, configs1[i].reportInterval, configs2[i].reportInterval)
    }
}

func TestComputeStats(t *testing.T) {
    values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

    stats := computeStats(values)

    assert.Equal(t, 5.5, stats.mean)
    assert.Equal(t, 1.0, stats.min)
    assert.Equal(t, 10.0, stats.max)
    assert.Equal(t, 5.0, stats.p50) // median of 10 values
    assert.Equal(t, 9.0, stats.p95)
    assert.Equal(t, 9.0, stats.p99) // with 10 values, p99 is at index 9
}
```

### Task 7.2: Config Parsing Tests

**File:** `config_test.go`

```go
func TestParseConfig_BackwardCompatibility(t *testing.T) {
    yaml := `
loadgen:
  vmCount: 100
  numPackages: 700
  reportInterval: 60s
  statsInterval: 30s
  port: 818
  metricsPort: 9090
`

    yamlCfg := parseYAML(yaml)
    cfg := configFromYAML(yamlCfg)

    assert.False(t, cfg.packages.isDistribution)
    assert.Equal(t, 700, cfg.packages.simpleValue)

    assert.False(t, cfg.reportInterval.isDistribution)
    // Check interval is 60s
}

func TestParseConfig_DistributionMode(t *testing.T) {
    yaml := `
loadgen:
  vmCount: 100
  packages:
    mean: 700
    stddev: 100
    min: 500
    max: 1000
  reportInterval:
    mean: 60s
    stddev: 10s
    min: 30s
    max: 120s
  statsInterval: 30s
  port: 818
`

    yamlCfg := parseYAML(yaml)
    cfg := configFromYAML(yamlCfg)

    assert.True(t, cfg.packages.isDistribution)
    assert.Equal(t, 700.0, cfg.packages.mean)
    assert.Equal(t, 100.0, cfg.packages.stddev)
    assert.Equal(t, 500.0, cfg.packages.min)
    assert.Equal(t, 1000.0, cfg.packages.max)

    assert.True(t, cfg.reportInterval.isDistribution)
    assert.Equal(t, 60.0, cfg.reportInterval.mean)
    // ... check other fields
}

func TestValidateDistribution(t *testing.T) {
    tests := []struct {
        name      string
        dist      distributionConfig
        expectErr bool
    }{
        {
            name: "valid distribution",
            dist: distributionConfig{
                mean: 700, stddev: 100, min: 500, max: 1000,
                isDistribution: true,
            },
            expectErr: false,
        },
        {
            name: "negative stddev",
            dist: distributionConfig{
                mean: 700, stddev: -10, min: 500, max: 1000,
                isDistribution: true,
            },
            expectErr: true,
        },
        {
            name: "max <= min",
            dist: distributionConfig{
                mean: 700, stddev: 100, min: 1000, max: 500,
                isDistribution: true,
            },
            expectErr: true,
        },
        {
            name: "simple mode (no validation)",
            dist: distributionConfig{
                simpleValue: 700,
                isDistribution: false,
            },
            expectErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateDistribution(tt.dist, "test")
            if tt.expectErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Task 7.3: Integration Test

**File:** `integration_test.go`

```go
func TestEndToEnd_DistributionMode(t *testing.T) {
    // Create test config
    pkgDist := packageDistribution{
        distributionConfig: distributionConfig{
            mean: 700, stddev: 100, min: 500, max: 1000,
            isDistribution: true,
        },
    }

    intervalDist := intervalDistribution{
        distributionConfig: distributionConfig{
            mean: 60.0, stddev: 10.0, min: 30.0, max: 120.0,
            isDistribution: true,
        },
    }

    // Assign VM configs
    vmConfigs := assignVMConfigs(100, 1000, pkgDist, intervalDist, 12345)

    // Create payload provider
    provider, err := newPayloadProvider(vmConfigs)
    assert.NoError(t, err)

    // Verify all VMs have payloads
    for _, cfg := range vmConfigs {
        payload, err := provider.get(cfg.cid)
        assert.NoError(t, err)
        assert.NotEmpty(t, payload)
    }

    // Verify distribution statistics
    pkgCounts := make([]float64, len(vmConfigs))
    for i, cfg := range vmConfigs {
        pkgCounts[i] = float64(cfg.packageCount)
    }
    stats := computeStats(pkgCounts)

    // Mean should be close to 700
    assert.InDelta(t, 700.0, stats.mean, 50.0)

    // All values should be in range
    assert.GreaterOrEqual(t, stats.min, 500.0)
    assert.LessOrEqual(t, stats.max, 1000.0)
}
```

**Estimated Effort:** 4-5 hours
**Risk:** Low

---

## Phase 8: Documentation & Migration

### Task 8.1: Update Config Documentation

**File:** `deploy/loadgen-config.yaml`

Update comments with examples:

```yaml
# Vsock Load Generator Configuration

loadgen:
  vmCount: 192

  # Option 1: Simple mode (backward compatible)
  # All VMs get exactly the same package count
  # numPackages: 700

  # Option 2: Distribution mode (realistic load)
  # VMs get varying package counts sampled from normal distribution
  packages:
    mean: 700      # Average package count
    stddev: 300    # Standard deviation (controls spread)
    min: 100       # Minimum packages (floor)
    max: 3000      # Maximum packages (ceiling)

  # Option 1: Simple mode (backward compatible)
  # reportInterval: 60s

  # Option 2: Distribution mode
  reportInterval:
    mean: 60s      # Average report interval
    stddev: 20s    # Standard deviation
    min: 30s       # Minimum interval
    max: 300s      # Maximum interval

  statsInterval: 30s
  port: 818
  metricsPort: 9090
  requestTimeout: 10s
```

### Task 8.2: Create Migration Guide

**File:** `MIGRATION_GUIDE.md` (new file)

```markdown
# Migration Guide: Upgrading to Distribution-Based Load

## Overview

Version X.Y introduces distribution-based load generation for more realistic testing.

## What Changed

**Before (uniform load):**
```yaml
numPackages: 700
reportInterval: 60s
```
All VMs identical: exactly 700 packages, exactly 60s interval.

**After (realistic load):**
```yaml
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
VMs vary: most ~700 packages, some much higher/lower.

## Backward Compatibility

**Old configs still work!** No changes required.

If your config has `numPackages: 700`, it will continue to work exactly as before.

## Migration Steps

### Step 1: Review Current Load

Check your current config:
```bash
kubectl get configmap vsock-loadgen-config -n stackrox -o yaml
```

### Step 2: Choose Distribution Parameters

Based on your testing goals:

**Conservative (slight variation):**
```yaml
packages:
  mean: 700
  stddev: 50    # ±50 packages variation
  min: 600
  max: 800
```

**Moderate (realistic):**
```yaml
packages:
  mean: 700
  stddev: 200   # wider spread
  min: 300
  max: 1500
```

**Aggressive (stress test):**
```yaml
packages:
  mean: 700
  stddev: 500   # huge variation
  min: 100
  max: 3000
```

### Step 3: Update ConfigMap

Edit and apply:
```bash
kubectl edit configmap vsock-loadgen-config -n stackrox
```

### Step 4: Restart Pods

```bash
kubectl rollout restart daemonset vsock-loadgen -n stackrox
```

### Step 5: Verify Distribution

Check logs for statistics:
```bash
kubectl logs -n stackrox -l app=vsock-loadgen | grep "Package distribution"
```

Expected output:
```
Package distribution: mean=702 stddev=298 min=150 max=2890 p50=690 p95=1456 p99=2103
```

Check Prometheus metrics:
```
vm_package_count_mean
vm_package_count_p50
vm_package_count_p95
vm_package_count_p99
```

## Troubleshooting

**Q: My distribution doesn't match config?**
A: Check that stddev, min, max are reasonable. If stddev=0, all VMs get mean value.

**Q: Can I go back to old behavior?**
A: Yes! Either revert config to `numPackages: 700` or set `stddev: 0`.

**Q: How do I calculate stddev for target P95?**
A: Roughly: `P95 ≈ mean + 1.65*stddev`. For P95=1200 with mean=700: `stddev ≈ (1200-700)/1.65 ≈ 300`.
```

**Estimated Effort:** 2 hours
**Risk:** Low

---

## Summary

### Total Estimated Effort: 20-25 hours

### Phase Breakdown:
1. Config Schema & Parsing: 4-6 hours
2. Distribution Sampling: 3-4 hours
3. Payload Generation: 2 hours
4. VM Simulation: 1 hour
5. Main Loop Integration: 1 hour
6. Observability: 1-2 hours
7. Testing: 4-5 hours
8. Documentation: 2 hours

### Risk Assessment:
- **Low Risk:** Distribution sampling, VM simulation, metrics
- **Medium Risk:** YAML config parsing (type detection can be tricky)
- **Overall:** LOW-MEDIUM risk

### Dependencies:
- No external dependencies needed
- Uses standard Go `math/rand` for normal distribution
- Backward compatible (no breaking changes)

### Testing Strategy:
1. Unit tests for distribution algorithms
2. Config parsing tests (both formats)
3. Integration test (end-to-end)
4. Manual testing with real deployment

### Rollout Plan:
1. Implement and test locally
2. Deploy to dev cluster with old config (verify backward compat)
3. Deploy with new config (verify distributions)
4. Monitor for 24h
5. Document results
6. Roll out to test/prod

### Files to Create/Modify:

**New Files:**
- `distribution.go` - distribution sampling logic
- `distribution_test.go` - unit tests
- `MIGRATION_GUIDE.md` - user documentation

**Modified Files:**
- `config.go` - new types, parsing, validation
- `payload.go` - multi-generator support
- `simulator.go` - per-VM config
- `main.go` - wire everything together
- `metrics.go` - new Prometheus metrics
- `deploy/loadgen-config.yaml` - updated documentation

**Total:** 3 new files, 6 modified files

---

## Next Steps

1. Review this plan with team
2. Create JIRA ticket
3. Implement Phase 1 (config parsing)
4. Write tests as you go
5. Manual testing after Phase 5
6. Document and roll out

## Alternative: Quick Prototype

For rapid testing, could implement a simplified version first:
- Skip YAML distribution config
- Hardcode normal distribution in code
- Verify it works
- Then add full config support

This reduces initial complexity for proof-of-concept.
