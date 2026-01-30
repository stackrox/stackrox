# Prompt for LLM: Create Implementation Plan for Percentile-Based Load Distribution

## Context

I'm working on a Go-based load generator (`vsock-loadgen`) that simulates virtual machines (VMs) sending index reports to a vulnerability scanner. Currently, all VMs use uniform values (same package count, same report interval), which doesn't reflect real-world scenarios.

**Goal:** Implement percentile-based distributions for package counts and report intervals to create more realistic load patterns.

## Current Architecture

### File Structure
```
compliance/virtualmachines/loadgen/
├── main.go           # Entry point, orchestrates VM simulation
├── config.go         # YAML config parsing and validation
├── simulator.go      # VM simulation logic (sends reports periodically)
├── payload.go        # Pre-generates protobuf payloads for each VM
├── stats.go          # Statistics collection
├── metrics.go        # Prometheus metrics
├── cid.go            # CID (VM ID) range calculation
└── deploy/
    └── loadgen-config.yaml  # Configuration file
```

### Current Config Format
```yaml
loadgen:
  vmCount: 192           # Total VMs across all nodes
  numPackages: 700       # All VMs have exactly 700 packages
  reportInterval: 60s    # All VMs report every 60s ± 5% jitter
  statsInterval: 30s
  port: 818
  metricsPort: 9090
  requestTimeout: 10s
```

### How It Works Now

1. **Config Parsing** (`config.go`):
   - `parseConfig()` reads YAML, validates, returns `config` struct
   - Single `numPackages` integer for all VMs

2. **Report Generation** (`payload.go`):
   - Creates ONE `vmindexreport.Generator` with fixed package count
   - Pre-generates unique payloads for each VM (CID-based)
   - `newPayloadProvider(generator, vmCount, startCID)` returns map of `cid -> []byte`

3. **VM Simulation** (`simulator.go`):
   - Each VM runs in a goroutine: `simulateVM(ctx, cid, cfg, provider, stats, metrics)`
   - Uses same `cfg.reportInterval` for all VMs
   - Adds ±5% jitter to prevent perfect synchronization
   - Sends reports by calling `sendVMReport()` which does vsock write

4. **Main Loop** (`main.go`):
   - Parses config
   - Creates single generator
   - Creates payload provider
   - Spawns `vmCount` goroutines (one per VM)
   - Each goroutine calls `simulateVM()`

### Key Code Snippets

**Current config struct:**
```go
type config struct {
    vmCount        int
    numPackages    int           // ← Single value for all VMs
    reportInterval time.Duration // ← Single value for all VMs
    duration       time.Duration
    port           uint
    metricsPort    int
    statsInterval  time.Duration
    requestTimeout time.Duration
}
```

**Current payload generation:**
```go
// Creates ONE generator for all VMs
func createReportGenerator(cfg config) (*vmindexreport.Generator, error) {
    generator := vmindexreport.NewGeneratorWithSeed(cfg.numPackages, 0)
    return generator, nil
}

// All VMs use the same generator
func newPayloadProvider(generator *vmindexreport.Generator, vmCount int, startCID uint32) (*payloadProvider, error) {
    payloads := make(map[uint32][]byte)
    for i := 0; i < vmCount; i++ {
        cid := startCID + uint32(i)
        report := generator.GenerateV1IndexReport(cid)  // ← Same package count
        data, err := proto.Marshal(report)
        payloads[cid] = data
    }
    return &payloadProvider{payloads: payloads}, nil
}
```

**Current VM simulation:**
```go
func simulateVM(ctx context.Context, cid uint32, cfg config, provider *payloadProvider, stats *statsCollector, metrics *metricsRegistry) {
    payload, _ := provider.get(cid)
    rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(cid)))

    // All VMs use same cfg.reportInterval
    initialDelay := time.Duration(rng.Int63n(int64(cfg.reportInterval)))
    time.Sleep(initialDelay)

    sendVMReport(cid, payload, cfg.port, cfg.requestTimeout, stats, metrics)

    for {
        jitter := time.Duration(float64(cfg.reportInterval) * (rng.Float64()*0.1 - 0.05))
        nextInterval := cfg.reportInterval + jitter
        time.Sleep(nextInterval)
        sendVMReport(...)
    }
}
```

## Desired Behavior (Option 2: Percentile-Based)

### New Config Format

```yaml
loadgen:
  vmCount: 200

  # Percentile-based package distribution
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

  # Percentile-based report interval distribution
  reportInterval:
    type: percentile
    buckets:
      - percent: 50
        value: 30s
      - percent: 30
        value: 60s
      - percent: 20
        value: 300s

  # Backward compatibility: old format still works
  # packages: 700           # Simple int = all VMs get 700 (uniform)
  # reportInterval: 60s     # Simple duration = all VMs get 60s (uniform)
```

### Expected Outcome (200 VMs)

**Package Distribution:**
- 140 VMs (70%) with exactly 500 packages
- 40 VMs (20%) with exactly 1000 packages
- 16 VMs (8%) with exactly 1500 packages
- 4 VMs (2%) with exactly 2000 packages

**Report Interval Distribution:**
- 100 VMs (50%) report every 30s
- 60 VMs (30%) report every 60s
- 40 VMs (20%) report every 300s

**Statistics:**
- Package mean: ~710, P50: 500, P95: 1500, P99: 2000
- Report interval mean: ~84s, P50: 30s, P95: 300s

## Requirements

1. **Backward Compatibility:**
   - Old config format (simple int/duration) must still work
   - Detect format automatically (type assertion / YAML unmarshaling)

2. **Validation:**
   - Bucket percentages must sum to 100%
   - Values must be positive
   - At least one bucket required

3. **Assignment Strategy:**
   - VMs should be randomly assigned to buckets (not sequential)
   - Prevents "all small VMs start first" artifacts
   - Deterministic if using a fixed seed (for reproducibility)

4. **Payload Generation:**
   - Need MULTIPLE generators (one per unique package count)
   - Reuse generators for VMs with same package count
   - Pre-generate all payloads at startup (maintain performance)

5. **VM Configuration:**
   - Each VM needs to know its specific package count and report interval
   - Pass per-VM config to `simulateVM()` instead of global config

6. **Observability:**
   - Log distribution statistics at startup (mean, P50, P95, P99)
   - Add Prometheus metrics for package/interval distributions
   - Verify bucket percentages were applied correctly

## Tasks to Include in Plan

Please create a detailed implementation plan that covers:

1. **Config Schema Changes:**
   - How to extend YAML schema to support both formats
   - New Go types for percentile distributions
   - Backward compatibility detection

2. **Config Parsing:**
   - YAML unmarshaling strategy (custom UnmarshalYAML?)
   - Validation logic
   - Error handling

3. **Distribution Sampling:**
   - Algorithm to assign VMs to buckets
   - How to handle rounding errors (e.g., 200 * 8% = 16, but 197 * 8% = 15.76)
   - Randomization strategy (shuffle VMs before assignment?)

4. **Payload Provider Refactoring:**
   - Change from single generator to multiple generators
   - Map package count → generator
   - Efficient pre-generation for mixed package counts

5. **VM Simulation Changes:**
   - New `vmConfig` struct to hold per-VM settings
   - How to pass per-VM report interval to `simulateVM()`
   - Remove/adjust existing jitter logic

6. **Observability:**
   - Prometheus metrics to add
   - Startup logging (distribution statistics)
   - Runtime verification

7. **Testing Strategy:**
   - Unit tests for distribution sampling
   - Config parsing tests (both formats)
   - Validation tests (percentages sum to 100, etc.)
   - Integration test: verify actual VM assignments match config

8. **File-by-File Changes:**
   - Which files need modification
   - Specific functions to change
   - New functions to add

9. **Migration Path:**
   - How to update existing deployments
   - Documentation updates needed

## Constraints

- **Performance:** Pre-generation of payloads is critical (don't slow down startup significantly)
- **Simplicity:** Keep the implementation as simple as possible while meeting requirements
- **No Breaking Changes:** Old configs must continue to work without modification
- **Reproducibility:** Same config + seed should produce same distribution

## References

See attached files:
- `DISTRIBUTION_DESIGN.md` - Overall design discussion (Option 1 vs Option 2)
- `OPTION2_PERCENTILE_EXAMPLES.md` - Detailed examples and use cases
- Current source code in `compliance/virtualmachines/loadgen/`

## Your Task

Create a **step-by-step implementation plan** that:
1. Breaks down the work into logical phases/tasks
2. Identifies all files that need changes
3. Specifies new types, functions, and algorithms
4. Includes testing strategy
5. Addresses backward compatibility
6. Provides code snippets/pseudocode where helpful
7. Estimates complexity/risk for each task

The plan should be detailed enough for another developer to implement without further design work.

---

**Bonus Question:** Should we implement Option 1 (normal distribution with mean/stddev) first as a simpler stepping stone, or jump straight to Option 2?

**Note:** The end goal is to support realistic load testing as described in the GAPS_AND_IMPROVEMENTS.md document, which calls for testing non-uniform package distributions and bursty traffic patterns.
