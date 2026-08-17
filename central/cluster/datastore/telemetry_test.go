package datastore

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stretchr/testify/assert"
)

func TestBuildVMTraits(t *testing.T) {
	tests := map[string]struct {
		hasCapability bool
		metrics       *central.VirtualMachineMetrics
		wantTraits    map[string]any
	}{
		"should set VM Scanning Enabled true with sorted version JSON when metrics are non-nil": {
			hasCapability: true,
			metrics: &central.VirtualMachineMetrics{
				TrackedVms: 10,
				VmsScanned: 7,
				RoxagentVersionCounts: map[string]int32{
					"v1.0.0":  4,
					"v2.0.0":  2,
					"unknown": 1,
				},
			},
			wantTraits: map[string]any{
				"VM Scanning Enabled":     true,
				"VM Tracked Count":        int32(10),
				"VM Scanned Count":        int32(7),
				"VM Unscanned Count":      int32(3),
				"Roxagent Version Counts": `[{"version":"unknown","count":1},{"version":"v1.0.0","count":4},{"version":"v2.0.0","count":2}]`,
			},
		},
		"should zero traits including version counts when capable Sensor sends nil metrics (feature off)": {
			hasCapability: true,
			metrics:       nil,
			wantTraits: map[string]any{
				"VM Scanning Enabled":     false,
				"VM Tracked Count":        int32(0),
				"VM Scanned Count":        int32(0),
				"VM Unscanned Count":      int32(0),
				"Roxagent Version Counts": "[]",
			},
		},
		"should set empty version counts for non-nil-but-all-zero metrics (feature on, zero VMs)": {
			hasCapability: true,
			metrics:       &central.VirtualMachineMetrics{},
			wantTraits: map[string]any{
				"VM Scanning Enabled":     true,
				"VM Tracked Count":        int32(0),
				"VM Scanned Count":        int32(0),
				"VM Unscanned Count":      int32(0),
				"Roxagent Version Counts": "[]",
			},
		},
		"should omit unknown from version JSON when the map has no unknown bucket": {
			hasCapability: true,
			metrics: &central.VirtualMachineMetrics{
				TrackedVms: 4,
				VmsScanned: 4,
				RoxagentVersionCounts: map[string]int32{
					"v1.0.0": 4,
				},
			},
			wantTraits: map[string]any{
				"VM Scanning Enabled":     true,
				"VM Tracked Count":        int32(4),
				"VM Scanned Count":        int32(4),
				"VM Unscanned Count":      int32(0),
				"Roxagent Version Counts": `[{"version":"v1.0.0","count":4}]`,
			},
		},
		"should return nil traits when Sensor lacks capability (old Sensor)": {
			hasCapability: false,
			metrics:       nil,
			wantTraits:    nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.wantTraits, buildVMTraits(tc.hasCapability, tc.metrics))
		})
	}
}
