package datastore

import (
	"encoding/json"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildVMTraits(t *testing.T) {
	tests := map[string]struct {
		hasCapability bool
		metrics       *central.VirtualMachineMetrics
		wantTraits    map[string]any
	}{
		"should set VM Scanning Enabled true with counts when metrics are non-nil": {
			hasCapability: true,
			metrics: &central.VirtualMachineMetrics{
				TrackedVms: 10,
				VmsScanned: 7,
				RoxagentVersionCounts: map[string]int32{
					"v1.0.0":  5,
					"v2.0.0":  3,
					"unknown": 2,
				},
			},
			wantTraits: map[string]any{
				"VM Scanning Enabled": true,
				"VM Tracked Count":    int32(10),
				"VM Scanned Count":    int32(7),
			},
		},
		"should zero traits when capable Sensor sends nil metrics (feature off)": {
			hasCapability: true,
			metrics:       nil,
			wantTraits: map[string]any{
				"VM Scanning Enabled": false,
				"VM Tracked Count":    int32(0),
				"VM Scanned Count":    int32(0),
			},
		},
		"should set traits for non-nil-but-all-zero metrics (feature on, zero VMs)": {
			hasCapability: true,
			metrics:       &central.VirtualMachineMetrics{},
			wantTraits: map[string]any{
				"VM Scanning Enabled": true,
				"VM Tracked Count":    int32(0),
				"VM Scanned Count":    int32(0),
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
			got := buildVMTraits(tc.hasCapability, tc.metrics)
			if tc.wantTraits == nil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tc.wantTraits["VM Scanning Enabled"], got["VM Scanning Enabled"])
			assert.Equal(t, tc.wantTraits["VM Tracked Count"], got["VM Tracked Count"])
			assert.Equal(t, tc.wantTraits["VM Scanned Count"], got["VM Scanned Count"])

			if tc.metrics != nil && len(tc.metrics.GetRoxagentVersionCounts()) > 0 {
				raw, ok := got["Roxagent Version Counts"].(string)
				require.True(t, ok, "Roxagent Version Counts should be a JSON string")
				var parsed []map[string]any
				require.NoError(t, json.Unmarshal([]byte(raw), &parsed))
				assert.Len(t, parsed, len(tc.metrics.GetRoxagentVersionCounts()))
			}
		})
	}
}
