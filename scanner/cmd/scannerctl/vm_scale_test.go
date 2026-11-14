package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateVMScaleConfig(t *testing.T) {
	valid := vmScaleConfig{numWorkers: 1, numRequests: 1, numPackages: 1}

	tests := map[string]struct {
		cfg     vmScaleConfig
		wantErr string
	}{
		"should accept a minimal config": {
			cfg: valid,
		},
		"should reject zero workers": {
			cfg:     vmScaleConfig{numRequests: 1},
			wantErr: "--workers must be positive",
		},
		"should reject a negative package count": {
			cfg:     vmScaleConfig{numWorkers: 1, numPackages: -1},
			wantErr: "--packages must be non-negative",
		},
		"should reject a negative request count": {
			cfg:     vmScaleConfig{numWorkers: 1, numRequests: -1},
			wantErr: "--requests must be non-negative",
		},
		"should reject a rate that cannot be scheduled": {
			cfg:     vmScaleConfig{numWorkers: 1, rateLimit: 1e12},
			wantErr: "too high to schedule",
		},
		"should reject combined selection modes": {
			cfg:     vmScaleConfig{numWorkers: 1, zeroVulnsOnly: true, vulnerableOnly: true, vulnDataFile: "v.json"},
			wantErr: "mutually exclusive",
		},
		"should reject a selection mode without learned data": {
			cfg:     vmScaleConfig{numWorkers: 1, targetVulns: 10},
			wantErr: "require --vuln-data",
		},
		"should reject direct pod mode without an address": {
			cfg:     vmScaleConfig{numWorkers: 1, directPodIPs: true},
			wantErr: "--direct-pod-ips requires --matcher-address",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateVMScaleConfig(tt.cfg)
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func learnedVulnFixture() *LearnedVulnData {
	return &LearnedVulnData{
		Packages: []PackageVulnData{
			{Index: 0, Vulns: 0},
			{Index: 1, Vulns: 5},
			{Index: 2, Vulns: 1},
			{Index: 3, Vulns: -1},
			{Index: 4, Vulns: 0},
		},
	}
}

func TestSelectPackagesFromLearnedData(t *testing.T) {
	learned := learnedVulnFixture()

	tests := map[string]struct {
		numPackages    int
		targetVulns    int
		zeroVulnsOnly  bool
		vulnerableOnly bool
		wantIndices    []int
		wantVulns      int
		wantErr        string
	}{
		"should use the first usable packages and skip query errors": {
			numPackages: 3,
			wantIndices: []int{0, 1, 2},
			wantVulns:   6,
		},
		"should use only zero-vuln packages": {
			numPackages:   2,
			zeroVulnsOnly: true,
			wantIndices:   []int{0, 4},
		},
		"should use vulnerable packages with the most vulns first": {
			numPackages:    2,
			vulnerableOnly: true,
			wantIndices:    []int{1, 2},
			wantVulns:      6,
		},
		"should reach the target then fill with zero-vuln packages": {
			numPackages: 3,
			targetVulns: 6,
			wantIndices: []int{1, 2, 0},
			wantVulns:   6,
		},
		"should keep a package that overshoots the target": {
			numPackages: 1,
			targetVulns: 3,
			wantIndices: []int{1},
			wantVulns:   5,
		},
		"should fail when not enough packages match": {
			numPackages:   3,
			zeroVulnsOnly: true,
			wantErr:       "fewer than --packages 3",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			indices, vulns, err := selectPackagesFromLearnedData(learned, tt.numPackages, tt.targetVulns, tt.zeroVulnsOnly, tt.vulnerableOnly)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantIndices, indices)
			assert.Equal(t, tt.wantVulns, vulns)
		})
	}
}

func TestDispatchVMScaleWork_SendsRequestedCount(t *testing.T) {
	workC := make(chan int, 4)
	dispatchVMScaleWork(context.Background(), workC, 3, 0, nil)
	close(workC)

	var got []int
	for id := range workC {
		got = append(got, id)
	}
	assert.Equal(t, []int{0, 1, 2}, got)
}

func TestDispatchVMScaleWork_ZeroRequestsDoesNotBlock(t *testing.T) {
	workC := make(chan int)
	done := make(chan struct{})
	go func() {
		defer close(done)
		dispatchVMScaleWork(context.Background(), workC, 0, 0, nil)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("dispatch blocked when no requests were requested")
	}
}

func TestDispatchVMScaleWork_StopsWhenContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	workC := make(chan int)
	done := make(chan struct{})
	go func() {
		defer close(done)
		dispatchVMScaleWork(ctx, workC, 100, 0, nil)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("dispatch did not return after cancel")
	}
}

func TestDispatchVMScaleWork_StopsAtDurationWithoutAReceiver(t *testing.T) {
	workC := make(chan int)
	start := time.Now()
	dispatchVMScaleWork(context.Background(), workC, 1000, 30*time.Millisecond, nil)
	assert.Less(t, time.Since(start), time.Second)
}
