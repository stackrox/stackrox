package vmscraper

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stackrox/rox/sensor/common/virtualmachine/metrics"
	"github.com/stretchr/testify/assert"
)

func TestLogAndRecordDiscoveredFacts(t *testing.T) {
	t.Run("empty facts map records nothing", func(t *testing.T) {
		before := testutil.ToFloat64(metrics.VMDiscoveredDataScanTrigger.WithLabelValues("unknown"))
		logAndRecordDiscoveredFacts("ns/vm-empty", nil)
		assert.Equal(t, before, testutil.ToFloat64(metrics.VMDiscoveredDataScanTrigger.WithLabelValues("unknown")))
	})

	t.Run("populated facts increment discovered data metric", func(t *testing.T) {
		facts := map[string]string{
			"detected_os":         "RHEL",
			"os_version":          "9.7",
			"activation_status":   "ACTIVE",
			"dnf_metadata_status": "AVAILABLE",
		}

		beforeData := testutil.ToFloat64(metrics.VMDiscoveredData.WithLabelValues("RHEL", "ACTIVE", "AVAILABLE"))
		logAndRecordDiscoveredFacts("ns/vm-1", facts)
		assert.Equal(t, beforeData+1, testutil.ToFloat64(metrics.VMDiscoveredData.WithLabelValues("RHEL", "ACTIVE", "AVAILABLE")))
	})

	t.Run("records scan_trigger metric for a reactive report", func(t *testing.T) {
		before := testutil.ToFloat64(metrics.VMDiscoveredDataScanTrigger.WithLabelValues("reactive"))
		logAndRecordDiscoveredFacts("ns/vm-3", map[string]string{"scan_trigger": "reactive"})
		assert.Equal(t, before+1, testutil.ToFloat64(metrics.VMDiscoveredDataScanTrigger.WithLabelValues("reactive")))
	})

	t.Run("missing scan_trigger records the unknown label (older agents)", func(t *testing.T) {
		before := testutil.ToFloat64(metrics.VMDiscoveredDataScanTrigger.WithLabelValues("unknown"))
		logAndRecordDiscoveredFacts("ns/vm-4", map[string]string{"detected_os": "RHEL"})
		assert.Equal(t, before+1, testutil.ToFloat64(metrics.VMDiscoveredDataScanTrigger.WithLabelValues("unknown")))
	})

	t.Run("garbage scan_trigger value records the unknown label", func(t *testing.T) {
		before := testutil.ToFloat64(metrics.VMDiscoveredDataScanTrigger.WithLabelValues("unknown"))
		logAndRecordDiscoveredFacts("ns/vm-5", map[string]string{"scan_trigger": "bogus-value"})
		assert.Equal(t, before+1, testutil.ToFloat64(metrics.VMDiscoveredDataScanTrigger.WithLabelValues("unknown")))
	})
}

func TestIsReactiveTrigger(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		facts    map[string]string
		expected bool
	}{
		"reactive fact returns true":        {facts: map[string]string{"scan_trigger": "reactive"}, expected: true},
		"scheduled fact returns false":      {facts: map[string]string{"scan_trigger": "scheduled"}, expected: false},
		"absent fact defaults to false":     {facts: map[string]string{}, expected: false},
		"nil facts default to false":        {facts: nil, expected: false},
		"unrecognized value defaults false": {facts: map[string]string{"scan_trigger": "bogus"}, expected: false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, isReactiveTrigger(tc.facts))
		})
	}
}
