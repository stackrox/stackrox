package aggregator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_vulnerabilityTracker_getMetricsConfig(t *testing.T) {
	tracker := makeTracker("test")
	assert.NotNil(t, tracker)
	mc := tracker.getMetricsConfig()
	assert.Nil(t, mc)

	t.Run("test 0 period", func(t *testing.T) {
		tracker.reloadConfig(nil, 0)
		mc = tracker.getMetricsConfig()
		assert.Nil(t, mc)
	})

	t.Run("test with test configuration", func(t *testing.T) {
		cfg := makeTestConfig_Vulnerabilities()
		metrics, period, _ := parseVulnerabilitiesConfig(cfg)
		tracker.reloadConfig(metrics, period)
		mc = tracker.getMetricsConfig()
		assert.NotNil(t, mc)
		assert.Equal(t, 42*time.Hour, <-tracker.periodCh)
		assert.Equal(t, "=CRITICAL*", mc["metric1"]["Severity"][0].String())
	})
}
