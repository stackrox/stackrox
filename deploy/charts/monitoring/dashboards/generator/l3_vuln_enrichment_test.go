package generator

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestL3VulnEnrichment_BasicMetadata(t *testing.T) {
	d := L3VulnEnrichment()

	assert.Equal(t, "central-vuln-enrichment", d.UID)
	assert.Equal(t, "Central: Vulnerability Enrichment", d.Title)
	assert.Contains(t, d.Tags, "level-3")
	assert.Contains(t, d.Tags, "vulnerability-enrichment")
	assert.Contains(t, d.Tags, "stackrox")
	assert.Contains(t, d.Tags, "central")
}

func TestL3VulnEnrichment_HasBackLinkToCentralInternals(t *testing.T) {
	d := L3VulnEnrichment()

	require.Len(t, d.Links, 1)
	assert.Equal(t, "← Central Internals", d.Links[0].Title)
	assert.Equal(t, "central-internals", d.Links[0].TargetUID)
}

func TestL3VulnEnrichment_HasRequiredRows(t *testing.T) {
	d := L3VulnEnrichment()

	require.Equal(t, 5, len(d.Rows), "Should have exactly 5 rows")

	// Verify row titles
	rowTitles := make([]string, len(d.Rows))
	for i, row := range d.Rows {
		rowTitles[i] = row.Title
	}

	assert.Contains(t, rowTitles, "Scan Semaphore")
	assert.Contains(t, rowTitles, "Image Scanning")
	assert.Contains(t, rowTitles, "Node Scanning")
	assert.Contains(t, rowTitles, "Image Deduplication")
	assert.Contains(t, rowTitles, "Registry Client")
}

func TestL3VulnEnrichment_ScanSemaphoreRow(t *testing.T) {
	d := L3VulnEnrichment()

	var row *Row
	for i := range d.Rows {
		if d.Rows[i].Title == "Scan Semaphore" {
			row = &d.Rows[i]
			break
		}
	}

	require.NotNil(t, row)
	require.Len(t, row.Panels, 3)

	// Verify Scans In-Flight panel
	assert.Equal(t, "Scans In-Flight", row.Panels[0].Title)
	assert.Equal(t, "stat", row.Panels[0].Type)
	assert.Equal(t, 8, row.Panels[0].Width)
	assert.Equal(t, `sum(rox_image_scan_semaphore_holding_size)`, row.Panels[0].Queries[0].Expr)

	// Verify Semaphore Utilization panel - should have 2 queries
	assert.Equal(t, "Semaphore Utilization", row.Panels[1].Title)
	assert.Equal(t, "timeseries", row.Panels[1].Type)
	assert.Equal(t, 8, row.Panels[1].Width)
	require.Len(t, row.Panels[1].Queries, 2, "Semaphore Utilization should have 2 queries")
	assert.Equal(t, `rox_image_scan_semaphore_holding_size`, row.Panels[1].Queries[0].Expr)
	assert.Equal(t, `holding`, row.Panels[1].Queries[0].LegendFormat)
	assert.Equal(t, `rox_image_scan_semaphore_limit`, row.Panels[1].Queries[1].Expr)
	assert.Equal(t, `limit`, row.Panels[1].Queries[1].LegendFormat)

	// Verify Queue Waiting panel
	assert.Equal(t, "Queue Waiting", row.Panels[2].Title)
	assert.Equal(t, "timeseries", row.Panels[2].Type)
	assert.Equal(t, 8, row.Panels[2].Width)
	assert.Equal(t, `rox_image_scan_semaphore_queue_size`, row.Panels[2].Queries[0].Expr)
	assert.Equal(t, `{{subsystem}} - {{entity}}`, row.Panels[2].Queries[0].LegendFormat)
}

func TestL3VulnEnrichment_ImageScanningRow(t *testing.T) {
	d := L3VulnEnrichment()

	var row *Row
	for i := range d.Rows {
		if d.Rows[i].Title == "Image Scanning" {
			row = &d.Rows[i]
			break
		}
	}

	require.NotNil(t, row)
	require.Len(t, row.Panels, 4, "Image Scanning should have 4 panels (3 metrics + 1 gap)")

	// Verify Scan Duration p50/p95/p99 panel - should have 3 queries
	assert.Equal(t, "Scan Duration p50/p95/p99", row.Panels[0].Title)
	assert.Equal(t, "timeseries", row.Panels[0].Type)
	assert.Equal(t, 12, row.Panels[0].Width)
	require.Len(t, row.Panels[0].Queries, 3, "Scan Duration should have 3 queries")
	assert.Equal(t, `histogram_quantile(0.5, rate(rox_central_scan_duration_bucket[5m]))`, row.Panels[0].Queries[0].Expr)
	assert.Equal(t, `p50`, row.Panels[0].Queries[0].LegendFormat)
	assert.Equal(t, `histogram_quantile(0.95, rate(rox_central_scan_duration_bucket[5m]))`, row.Panels[0].Queries[1].Expr)
	assert.Equal(t, `p95`, row.Panels[0].Queries[1].LegendFormat)
	assert.Equal(t, `histogram_quantile(0.99, rate(rox_central_scan_duration_bucket[5m]))`, row.Panels[0].Queries[2].Expr)
	assert.Equal(t, `p99`, row.Panels[0].Queries[2].LegendFormat)

	// Verify Vuln Retrieval Duration panel
	assert.Equal(t, "Vuln Retrieval Duration", row.Panels[1].Title)
	assert.Equal(t, "timeseries", row.Panels[1].Type)
	assert.Equal(t, 12, row.Panels[1].Width)
	assert.Equal(t, `histogram_quantile(0.95, rate(rox_central_image_vuln_retrieval_duration_bucket[5m]))`, row.Panels[1].Queries[0].Expr)
	assert.Equal(t, `p95`, row.Panels[1].Queries[0].LegendFormat)

	// Verify Metadata Cache Hit Rate panel
	assert.Equal(t, "Metadata Cache Hit Rate", row.Panels[2].Title)
	assert.Equal(t, "timeseries", row.Panels[2].Type)
	assert.Equal(t, 12, row.Panels[2].Width)
	assert.Equal(t, `rate(rox_central_metadata_cache_hits[5m]) / (rate(rox_central_metadata_cache_hits[5m]) + rate(rox_central_metadata_cache_misses[5m]))`, row.Panels[2].Queries[0].Expr)
	assert.Equal(t, `hit rate`, row.Panels[2].Queries[0].LegendFormat)
	assert.Equal(t, "percentunit", row.Panels[2].Unit)

	// Verify Enrichment Calls gap panel
	assert.Equal(t, "GAP: Enrichment Calls", row.Panels[3].Title)
	assert.Equal(t, 12, row.Panels[3].Width)
	assert.Equal(t, 4, row.Panels[3].Height)
	assert.Contains(t, row.Panels[3].GapNote, "central_vuln_enrichment_requests_total")
	assert.Contains(t, row.Panels[3].GapNote, "Cannot calculate enrichment failure rate")
}

func TestL3VulnEnrichment_NodeScanningRow(t *testing.T) {
	d := L3VulnEnrichment()

	var row *Row
	for i := range d.Rows {
		if d.Rows[i].Title == "Node Scanning" {
			row = &d.Rows[i]
			break
		}
	}

	require.NotNil(t, row)
	require.Len(t, row.Panels, 2, "Node Scanning should have 2 panels (1 metric + 1 gap)")

	// Verify Node Scan Duration panel
	assert.Equal(t, "Node Scan Duration", row.Panels[0].Title)
	assert.Equal(t, "timeseries", row.Panels[0].Type)
	assert.Equal(t, 12, row.Panels[0].Width)
	assert.Equal(t, `histogram_quantile(0.95, rate(rox_central_node_scan_duration_bucket[5m]))`, row.Panels[0].Queries[0].Expr)
	assert.Equal(t, `p95`, row.Panels[0].Queries[0].LegendFormat)

	// Verify Node Scan Count gap panel
	assert.Equal(t, "GAP: Node Scan Count", row.Panels[1].Title)
	assert.Equal(t, 12, row.Panels[1].Width)
	assert.Equal(t, 4, row.Panels[1].Height)
	assert.Contains(t, row.Panels[1].GapNote, "central_vuln_enrichment_node_scans_total")
	assert.Contains(t, row.Panels[1].GapNote, "No counter for total node scans")
}

func TestL3VulnEnrichment_ImageDeduplicationRow(t *testing.T) {
	d := L3VulnEnrichment()

	var row *Row
	for i := range d.Rows {
		if d.Rows[i].Title == "Image Deduplication" {
			row = &d.Rows[i]
			break
		}
	}

	require.NotNil(t, row)
	require.Len(t, row.Panels, 2)

	// Verify Image Upsert Deduper panel
	assert.Equal(t, "Image Upsert Deduper", row.Panels[0].Title)
	assert.Equal(t, "timeseries", row.Panels[0].Type)
	assert.Equal(t, 12, row.Panels[0].Width)
	assert.Equal(t, `rate(rox_central_image_upsert_deduper[5m])`, row.Panels[0].Queries[0].Expr)
	assert.Equal(t, `{{status}}`, row.Panels[0].Queries[0].LegendFormat)

	// Verify Deployment Enhancement panel
	assert.Equal(t, "Deployment Enhancement", row.Panels[1].Title)
	assert.Equal(t, "timeseries", row.Panels[1].Type)
	assert.Equal(t, 12, row.Panels[1].Width)
	assert.Equal(t, `rox_central_deployment_enhancement_duration_ms`, row.Panels[1].Queries[0].Expr)
	assert.Equal(t, `duration`, row.Panels[1].Queries[0].LegendFormat)
}

func TestL3VulnEnrichment_RegistryClientRow(t *testing.T) {
	d := L3VulnEnrichment()

	var row *Row
	for i := range d.Rows {
		if d.Rows[i].Title == "Registry Client" {
			row = &d.Rows[i]
			break
		}
	}

	require.NotNil(t, row)
	require.Len(t, row.Panels, 3)

	// Verify Registry Requests panel
	assert.Equal(t, "Registry Requests", row.Panels[0].Title)
	assert.Equal(t, "timeseries", row.Panels[0].Type)
	assert.Equal(t, 8, row.Panels[0].Width)
	assert.Equal(t, `rate(rox_central_registry_client_requests_total[5m])`, row.Panels[0].Queries[0].Expr)
	assert.Equal(t, `{{code}} - {{type}}`, row.Panels[0].Queries[0].LegendFormat)

	// Verify Registry Latency panel
	assert.Equal(t, "Registry Latency", row.Panels[1].Title)
	assert.Equal(t, "timeseries", row.Panels[1].Type)
	assert.Equal(t, 8, row.Panels[1].Width)
	assert.Equal(t, `histogram_quantile(0.95, rate(rox_central_registry_client_request_duration_seconds_bucket[5m]))`, row.Panels[1].Queries[0].Expr)
	assert.Equal(t, `p95`, row.Panels[1].Queries[0].LegendFormat)
	assert.Equal(t, "s", row.Panels[1].Unit)

	// Verify Registry Timeouts panel
	assert.Equal(t, "Registry Timeouts", row.Panels[2].Title)
	assert.Equal(t, "timeseries", row.Panels[2].Type)
	assert.Equal(t, 8, row.Panels[2].Width)
	assert.Equal(t, `rate(rox_central_registry_client_error_timeouts_total[5m])`, row.Panels[2].Queries[0].Expr)
	assert.Equal(t, `timeouts`, row.Panels[2].Queries[0].LegendFormat)
}

func TestL3VulnEnrichment_ProducesValidJSON(t *testing.T) {
	d := L3VulnEnrichment()
	result := d.Generate()

	// Should marshal to valid JSON
	b, err := json.Marshal(result)
	require.NoError(t, err)
	require.NotEmpty(t, b)

	// Should unmarshal back
	var unmarshaled map[string]any
	err = json.Unmarshal(b, &unmarshaled)
	require.NoError(t, err)

	// Verify key fields survived round-trip
	assert.Equal(t, "central-vuln-enrichment", unmarshaled["uid"])
	assert.Equal(t, "Central: Vulnerability Enrichment", unmarshaled["title"])
}

func TestL3VulnEnrichment_AllPanelsHaveValidWidth(t *testing.T) {
	d := L3VulnEnrichment()

	for _, row := range d.Rows {
		for _, panel := range row.Panels {
			assert.Greater(t, panel.Width, 0, "Panel %s should have positive width", panel.Title)
			assert.LessOrEqual(t, panel.Width, 24, "Panel %s width should not exceed 24", panel.Title)
		}
	}
}
