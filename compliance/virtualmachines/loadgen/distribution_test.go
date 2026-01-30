package main

import (
	"math/rand"
	"testing"
	"time"
)

func TestSampleNormal(t *testing.T) {
	rng := newDeterministicRNG(42)
	mean := 100.0
	stddev := 20.0
	min := 50.0
	max := 150.0

	// Sample many values and verify they're within bounds
	for i := 0; i < 1000; i++ {
		value := sampleNormal(rng, mean, stddev, min, max)
		if value < min || value > max {
			t.Errorf("sampleNormal returned value %.2f outside bounds [%.2f, %.2f]", value, min, max)
		}
	}
}

func TestSampleNormalClamping(t *testing.T) {
	rng := newDeterministicRNG(123)
	mean := 100.0
	stddev := 50.0 // Large stddev to ensure we hit bounds
	min := 90.0
	max := 110.0

	// Sample many values - all should be clamped
	for i := 0; i < 1000; i++ {
		value := sampleNormal(rng, mean, stddev, min, max)
		if value < min || value > max {
			t.Errorf("sampleNormal returned value %.2f outside bounds [%.2f, %.2f]", value, min, max)
		}
	}
}

func TestAssignVMConfigsDeterministic(t *testing.T) {
	vmCount := 10
	startCID := uint32(3)
	pkgDist := packageDistribution{
		distributionConfig: distributionConfig{
			Mean:   700,
			Stddev: 300,
			Min:    100,
			Max:    3000,
		},
	}
	intervalDist := intervalDistribution{
		distributionConfig: distributionConfig{
			Mean:   60,
			Stddev: 20,
			Min:    30,
			Max:    300,
		},
		meanDuration:   60 * time.Second,
		stddevDuration: 20 * time.Second,
		minDuration:    30 * time.Second,
		maxDuration:    300 * time.Second,
	}
	seed := int64(42)

	// Run twice with same seed - should get identical results
	configs1 := assignVMConfigs(vmCount, startCID, pkgDist, intervalDist, seed)
	configs2 := assignVMConfigs(vmCount, startCID, pkgDist, intervalDist, seed)

	if len(configs1) != len(configs2) {
		t.Fatalf("configs1 length %d != configs2 length %d", len(configs1), len(configs2))
	}

	for i := range configs1 {
		if configs1[i].cid != configs2[i].cid {
			t.Errorf("configs1[%d].cid = %d, configs2[%d].cid = %d", i, configs1[i].cid, i, configs2[i].cid)
		}
		if configs1[i].numPackages != configs2[i].numPackages {
			t.Errorf("configs1[%d].numPackages = %d, configs2[%d].numPackages = %d", i, configs1[i].numPackages, i, configs2[i].numPackages)
		}
		if configs1[i].reportInterval != configs2[i].reportInterval {
			t.Errorf("configs1[%d].reportInterval = %v, configs2[%d].reportInterval = %v", i, configs1[i].reportInterval, i, configs2[i].reportInterval)
		}
	}
}

func TestAssignVMConfigsBounds(t *testing.T) {
	vmCount := 100
	startCID := uint32(3)
	pkgDist := packageDistribution{
		distributionConfig: distributionConfig{
			Mean:   700,
			Stddev: 300,
			Min:    100,
			Max:    3000,
		},
	}
	intervalDist := intervalDistribution{
		distributionConfig: distributionConfig{
			Mean:   60,
			Stddev: 20,
			Min:    30,
			Max:    300,
		},
		meanDuration:   60 * time.Second,
		stddevDuration: 20 * time.Second,
		minDuration:    30 * time.Second,
		maxDuration:    300 * time.Second,
	}
	seed := int64(12345)

	configs := assignVMConfigs(vmCount, startCID, pkgDist, intervalDist, seed)

	if len(configs) != vmCount {
		t.Fatalf("expected %d configs, got %d", vmCount, len(configs))
	}

	for i, cfg := range configs {
		// Verify CID assignment
		expectedCID := startCID + uint32(i)
		if cfg.cid != expectedCID {
			t.Errorf("configs[%d].cid = %d, expected %d", i, cfg.cid, expectedCID)
		}

		// Verify package count bounds
		if cfg.numPackages < 1 {
			t.Errorf("configs[%d].numPackages = %d, must be >= 1", i, cfg.numPackages)
		}
		if cfg.numPackages < int(pkgDist.Min) || cfg.numPackages > int(pkgDist.Max) {
			// Allow some tolerance for rounding
			if float64(cfg.numPackages) < pkgDist.Min-0.5 || float64(cfg.numPackages) > pkgDist.Max+0.5 {
				t.Errorf("configs[%d].numPackages = %d, expected in [%.0f, %.0f]", i, cfg.numPackages, pkgDist.Min, pkgDist.Max)
			}
		}

		// Verify interval bounds
		if cfg.reportInterval < time.Second {
			t.Errorf("configs[%d].reportInterval = %v, must be >= 1s", i, cfg.reportInterval)
		}
		intervalSeconds := cfg.reportInterval.Seconds()
		if intervalSeconds < intervalDist.Min || intervalSeconds > intervalDist.Max {
			// Allow some tolerance for rounding
			if intervalSeconds < intervalDist.Min-0.5 || intervalSeconds > intervalDist.Max+0.5 {
				t.Errorf("configs[%d].reportInterval = %v (%.2fs), expected in [%.2fs, %.2fs]",
					i, cfg.reportInterval, intervalSeconds, intervalDist.Min, intervalDist.Max)
			}
		}
	}
}

func TestAssignVMConfigsPackageMinimum(t *testing.T) {
	// Test that packages are always >= 1 even if distribution would produce 0
	vmCount := 10
	startCID := uint32(3)
	pkgDist := packageDistribution{
		distributionConfig: distributionConfig{
			Mean:   0.5, // Very low mean
			Stddev: 0.1,
			Min:    0, // Min is 0, but we should clamp to 1
			Max:    10,
		},
	}
	intervalDist := intervalDistribution{
		distributionConfig: distributionConfig{
			Mean:   60,
			Stddev: 20,
			Min:    30,
			Max:    300,
		},
		meanDuration:   60 * time.Second,
		stddevDuration: 20 * time.Second,
		minDuration:    30 * time.Second,
		maxDuration:    300 * time.Second,
	}
	seed := int64(999)

	configs := assignVMConfigs(vmCount, startCID, pkgDist, intervalDist, seed)

	for i, cfg := range configs {
		if cfg.numPackages < 1 {
			t.Errorf("configs[%d].numPackages = %d, must be >= 1", i, cfg.numPackages)
		}
	}
}

func TestAssignVMConfigsIntervalMinimum(t *testing.T) {
	// Test that intervals are always >= 1s even if distribution would produce less
	vmCount := 10
	startCID := uint32(3)
	pkgDist := packageDistribution{
		distributionConfig: distributionConfig{
			Mean:   700,
			Stddev: 300,
			Min:    100,
			Max:    3000,
		},
	}
	intervalDist := intervalDistribution{
		distributionConfig: distributionConfig{
			Mean:   0.5, // Very low mean (0.5 seconds)
			Stddev: 0.1,
			Min:    0, // Min is 0, but we should clamp to 1s
			Max:    10,
		},
		meanDuration:   500 * time.Millisecond,
		stddevDuration: 100 * time.Millisecond,
		minDuration:    0,
		maxDuration:    10 * time.Second,
	}
	seed := int64(888)

	configs := assignVMConfigs(vmCount, startCID, pkgDist, intervalDist, seed)

	for i, cfg := range configs {
		if cfg.reportInterval < time.Second {
			t.Errorf("configs[%d].reportInterval = %v, must be >= 1s", i, cfg.reportInterval)
		}
	}
}

// newDeterministicRNG creates a new RNG with a fixed seed for testing.
func newDeterministicRNG(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(seed))
}

func TestParsePackageDistribution(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid distribution",
			input: map[string]interface{}{
				"mean":   700.0,
				"stddev": 300.0,
				"min":    100.0,
				"max":    3000.0,
			},
			wantErr: false,
		},
		{
			name: "missing mean",
			input: map[string]interface{}{
				"stddev": 300.0,
				"min":    100.0,
				"max":    3000.0,
			},
			wantErr: true,
		},
		{
			name: "missing stddev",
			input: map[string]interface{}{
				"mean": 700.0,
				"min":  100.0,
				"max":  3000.0,
			},
			wantErr: true,
		},
		{
			name: "wrong type for mean",
			input: map[string]interface{}{
				"mean":   "700", // string instead of number
				"stddev": 300.0,
				"min":    100.0,
				"max":    3000.0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parsePackageDistribution(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePackageDistribution() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseIntervalDistribution(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid distribution with duration strings",
			input: map[string]interface{}{
				"mean":   "60s",
				"stddev": "20s",
				"min":    "30s",
				"max":    "300s",
			},
			wantErr: false,
		},
		{
			name: "missing mean",
			input: map[string]interface{}{
				"stddev": "20s",
				"min":    "30s",
				"max":    "300s",
			},
			wantErr: true,
		},
		{
			name: "invalid duration string",
			input: map[string]interface{}{
				"mean":   "60s",
				"stddev": "invalid",
				"min":    "30s",
				"max":    "300s",
			},
			wantErr: true,
		},
		{
			name: "number instead of duration string",
			input: map[string]interface{}{
				"mean":   60.0, // number instead of string
				"stddev": "20s",
				"min":    "30s",
				"max":    "300s",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseIntervalDistribution(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIntervalDistribution() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBackwardCompatibility_LegacyScalarValues(t *testing.T) {
	// Test that legacy scalar values are converted to distributions correctly
	var yamlCfg yamlConfig
	yamlCfg.Loadgen.VmCount = 100
	yamlCfg.Loadgen.NumPackages = 700 // Legacy scalar
	yamlCfg.Loadgen.ReportInterval = "60s" // Legacy scalar string
	yamlCfg.Loadgen.StatsInterval = "30s"
	yamlCfg.Loadgen.Port = 818
	yamlCfg.Loadgen.MetricsPort = 9090

	cfg := configFromYAML(&yamlCfg)

	// Verify packages distribution was created from legacy scalar
	if cfg.packageDist.Mean != 700 {
		t.Errorf("Expected packages mean=700, got %.2f", cfg.packageDist.Mean)
	}
	if cfg.packageDist.Stddev != 0 {
		t.Errorf("Expected packages stddev=0 (legacy scalar), got %.2f", cfg.packageDist.Stddev)
	}
	if cfg.packageDist.Min != 700 || cfg.packageDist.Max != 700 {
		t.Errorf("Expected packages min=max=700 (legacy scalar), got min=%.2f max=%.2f", cfg.packageDist.Min, cfg.packageDist.Max)
	}

	// Verify reportInterval distribution was created from legacy scalar
	expectedIntervalSeconds := 60.0
	if cfg.intervalDist.Mean != expectedIntervalSeconds {
		t.Errorf("Expected reportInterval mean=%.2fs, got %.2fs", expectedIntervalSeconds, cfg.intervalDist.Mean)
	}
	if cfg.intervalDist.Stddev != 0 {
		t.Errorf("Expected reportInterval stddev=0 (legacy scalar), got %.2f", cfg.intervalDist.Stddev)
	}
	if cfg.intervalDist.Min != expectedIntervalSeconds || cfg.intervalDist.Max != expectedIntervalSeconds {
		t.Errorf("Expected reportInterval min=max=%.2fs (legacy scalar), got min=%.2fs max=%.2fs",
			expectedIntervalSeconds, cfg.intervalDist.Min, cfg.intervalDist.Max)
	}
	if cfg.intervalDist.meanDuration != 60*time.Second {
		t.Errorf("Expected reportInterval meanDuration=60s, got %v", cfg.intervalDist.meanDuration)
	}
}

func TestBackwardCompatibility_DistributionFormat(t *testing.T) {
	// Test that new distribution format still works
	var yamlCfg yamlConfig
	yamlCfg.Loadgen.VmCount = 100
	yamlCfg.Loadgen.Packages = map[string]interface{}{
		"mean":   700.0,
		"stddev": 300.0,
		"min":    100.0,
		"max":    3000.0,
	}
	yamlCfg.Loadgen.ReportInterval = map[string]interface{}{
		"mean":   "60s",
		"stddev": "20s",
		"min":    "30s",
		"max":    "300s",
	}
	yamlCfg.Loadgen.StatsInterval = "30s"
	yamlCfg.Loadgen.Port = 818
	yamlCfg.Loadgen.MetricsPort = 9090

	cfg := configFromYAML(&yamlCfg)

	// Verify packages distribution
	if cfg.packageDist.Mean != 700 {
		t.Errorf("Expected packages mean=700, got %.2f", cfg.packageDist.Mean)
	}
	if cfg.packageDist.Stddev != 300 {
		t.Errorf("Expected packages stddev=300, got %.2f", cfg.packageDist.Stddev)
	}

	// Verify reportInterval distribution
	if cfg.intervalDist.Mean != 60 {
		t.Errorf("Expected reportInterval mean=60s, got %.2fs", cfg.intervalDist.Mean)
	}
	if cfg.intervalDist.Stddev != 20 {
		t.Errorf("Expected reportInterval stddev=20s, got %.2fs", cfg.intervalDist.Stddev)
	}
}
