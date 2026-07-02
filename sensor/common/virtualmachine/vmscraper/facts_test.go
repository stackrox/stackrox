package vmscraper

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stackrox/rox/sensor/common/virtualmachine/metrics"
	"github.com/stretchr/testify/assert"
)

func TestLogAndRecordDiscoveredFacts(t *testing.T) {
	t.Run("empty facts map records nothing", func(t *testing.T) {
		before := testutil.ToFloat64(metrics.VMDiscoveredDataDNFStatus.WithLabelValues("none"))
		logAndRecordDiscoveredFacts("ns/vm-empty", nil)
		assert.Equal(t, before, testutil.ToFloat64(metrics.VMDiscoveredDataDNFStatus.WithLabelValues("none")))
	})

	t.Run("populated facts increment discovered data and per-flag DNF metrics", func(t *testing.T) {
		facts := map[string]string{
			"detected_os":         "RHEL",
			"os_version":          "9.7",
			"activation_status":   "ACTIVE",
			"dnf_metadata_status": "AVAILABLE",
			"dnf_status":          "DNF_REPO_CONFIG_FOUND, DNF_V4_CACHE_FOUND",
		}

		beforeData := testutil.ToFloat64(metrics.VMDiscoveredData.WithLabelValues("RHEL", "ACTIVE", "AVAILABLE"))
		beforeRepo := testutil.ToFloat64(metrics.VMDiscoveredDataDNFStatus.WithLabelValues("DNF_REPO_CONFIG_FOUND"))
		beforeCache := testutil.ToFloat64(metrics.VMDiscoveredDataDNFStatus.WithLabelValues("DNF_V4_CACHE_FOUND"))

		logAndRecordDiscoveredFacts("ns/vm-1", facts)

		assert.Equal(t, beforeData+1, testutil.ToFloat64(metrics.VMDiscoveredData.WithLabelValues("RHEL", "ACTIVE", "AVAILABLE")))
		assert.Equal(t, beforeRepo+1, testutil.ToFloat64(metrics.VMDiscoveredDataDNFStatus.WithLabelValues("DNF_REPO_CONFIG_FOUND")))
		assert.Equal(t, beforeCache+1, testutil.ToFloat64(metrics.VMDiscoveredDataDNFStatus.WithLabelValues("DNF_V4_CACHE_FOUND")))
	})

	t.Run("none dnf_status increments the none label", func(t *testing.T) {
		before := testutil.ToFloat64(metrics.VMDiscoveredDataDNFStatus.WithLabelValues("none"))
		logAndRecordDiscoveredFacts("ns/vm-2", map[string]string{"dnf_status": "none"})
		assert.Equal(t, before+1, testutil.ToFloat64(metrics.VMDiscoveredDataDNFStatus.WithLabelValues("none")))
	})
}
