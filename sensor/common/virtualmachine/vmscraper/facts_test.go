package vmscraper

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stackrox/rox/sensor/common/virtualmachine/metrics"
	"github.com/stretchr/testify/assert"
)

func TestLogAndRecordDiscoveredFacts(t *testing.T) {
	t.Run("empty facts map records nothing", func(t *testing.T) {
		before := testutil.ToFloat64(metrics.VMDiscoveredData.WithLabelValues("RHEL", "ACTIVE", "AVAILABLE"))
		logAndRecordDiscoveredFacts("ns/vm-empty", nil)
		assert.Equal(t, before, testutil.ToFloat64(metrics.VMDiscoveredData.WithLabelValues("RHEL", "ACTIVE", "AVAILABLE")))
	})

	t.Run("populated facts increment discovered data metric", func(t *testing.T) {
		facts := map[string]string{
			"detected_os":         "RHEL",
			"os_version":          "9.7",
			"activation_status":   "ACTIVE",
			"dnf_metadata_status": "AVAILABLE",
		}

		before := testutil.ToFloat64(metrics.VMDiscoveredData.WithLabelValues("RHEL", "ACTIVE", "AVAILABLE"))
		logAndRecordDiscoveredFacts("ns/vm-1", facts)
		assert.Equal(t, before+1, testutil.ToFloat64(metrics.VMDiscoveredData.WithLabelValues("RHEL", "ACTIVE", "AVAILABLE")))
	})
}
