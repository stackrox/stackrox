package image

import (
	"testing"
)

func TestNoChartPanic(t *testing.T) {
	// Verify at runtime that this won't panic
	GetCentralChart()
	GetMonitoringChart()
	GetScannerChart()
}
