package generator

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestL2CentralInternals_BasicMetadata(t *testing.T) {
	d := L2CentralInternals()

	assert.Equal(t, "central-internals", d.UID)
	assert.Equal(t, "Central Internals", d.Title)
	assert.Contains(t, d.Tags, "level-2")
	assert.Contains(t, d.Tags, "stackrox")
	assert.Contains(t, d.Tags, "central")
}

func TestL2CentralInternals_HasBackLinkToOverview(t *testing.T) {
	d := L2CentralInternals()

	require.Len(t, d.Links, 1)
	assert.Equal(t, "← StackRox Overview", d.Links[0].Title)
	assert.Equal(t, "stackrox-overview", d.Links[0].TargetUID)
}

func TestL2CentralInternals_HasTenRows(t *testing.T) {
	d := L2CentralInternals()

	require.Len(t, d.Rows, 10, "Should have exactly 10 rows (one per logical region)")

	expectedRows := []string{
		"Sensor Ingestion",
		"Deployment Processing",
		"Vulnerability Enrichment",
		"Detection & Alerts",
		"Risk Calculation",
		"Background Reprocessing",
		"Pruning & GC",
		"Network Analysis",
		"Report Generation",
		"API & UI",
	}

	rowTitles := make([]string, len(d.Rows))
	for i, row := range d.Rows {
		rowTitles[i] = row.Title
	}

	assert.Equal(t, expectedRows, rowTitles)
}

func TestL2CentralInternals_EachRowHasPanels(t *testing.T) {
	d := L2CentralInternals()

	for _, row := range d.Rows {
		assert.Greater(t, len(row.Panels), 0, "Row %s should have at least one panel", row.Title)
	}
}

func TestL2CentralInternals_SensorIngestionRow(t *testing.T) {
	d := L2CentralInternals()

	var row *Row
	for i := range d.Rows {
		if d.Rows[i].Title == "Sensor Ingestion" {
			row = &d.Rows[i]
			break
		}
	}

	require.NotNil(t, row, "Sensor Ingestion row should exist")
	require.Len(t, row.Panels, 4, "Should have 3 metric panels + 1 details link")

	// Verify panel titles
	assert.Equal(t, "events/sec", row.Panels[0].Title)
	assert.Equal(t, "deduper", row.Panels[1].Title)
	assert.Equal(t, "processing latency p95", row.Panels[2].Title)

	// Verify details link panel
	detailsPanel := row.Panels[3]
	assert.NotEmpty(t, detailsPanel.GapNote, "Details link should use GapNote for markdown")
	assert.Contains(t, detailsPanel.GapNote, "central-sensor-ingestion")
	assert.Contains(t, detailsPanel.GapNote, "Details")
}

func TestL2CentralInternals_DeploymentProcessingRow(t *testing.T) {
	d := L2CentralInternals()

	var row *Row
	for i := range d.Rows {
		if d.Rows[i].Title == "Deployment Processing" {
			row = &d.Rows[i]
			break
		}
	}

	require.NotNil(t, row, "Deployment Processing row should exist")

	// Verify metric panels exist
	assert.Equal(t, "resources/sec", row.Panels[0].Title)
	assert.Equal(t, "K8s event latency", row.Panels[1].Title)

	// Verify queries are correct
	assert.Contains(t, row.Panels[0].Queries[0].Expr, "rox_central_resource_processed_count")
	assert.Contains(t, row.Panels[1].Queries[0].Expr, "rox_central_k8s_event_processing_duration_bucket")
}

func TestL2CentralInternals_VulnerabilityEnrichmentRow(t *testing.T) {
	d := L2CentralInternals()

	var row *Row
	for i := range d.Rows {
		if d.Rows[i].Title == "Vulnerability Enrichment" {
			row = &d.Rows[i]
			break
		}
	}

	require.NotNil(t, row, "Vulnerability Enrichment row should exist")
	require.Len(t, row.Panels, 4, "Should have 3 metric panels + 1 details link")

	// Verify panel titles
	assert.Equal(t, "scans in-flight", row.Panels[0].Title)
	assert.Equal(t, "scan duration p95", row.Panels[1].Title)
	assert.Equal(t, "queue waiting", row.Panels[2].Title)

	// Verify panel types
	assert.Equal(t, "stat", row.Panels[0].Type) // scans in-flight is a stat
	assert.Equal(t, "timeseries", row.Panels[1].Type)
	assert.Equal(t, "timeseries", row.Panels[2].Type)
}

func TestL2CentralInternals_DetectionAlertsRow(t *testing.T) {
	d := L2CentralInternals()

	var row *Row
	for i := range d.Rows {
		if d.Rows[i].Title == "Detection & Alerts" {
			row = &d.Rows[i]
			break
		}
	}

	require.NotNil(t, row, "Detection & Alerts row should exist")

	// Should have process filter panel
	assert.Equal(t, "process filter", row.Panels[0].Title)
	assert.Contains(t, row.Panels[0].Queries[0].Expr, "rox_central_process_filter")

	// Should have gap panel for alert generation rate
	var gapPanel *Panel
	for i := range row.Panels {
		if row.Panels[i].GapNote != "" && !isDetailsLink(row.Panels[i].GapNote) {
			gapPanel = &row.Panels[i]
			break
		}
	}
	require.NotNil(t, gapPanel, "Should have gap panel for missing alert generation rate metric")
	assert.Contains(t, gapPanel.GapNote, "alert generation rate")
}

func TestL2CentralInternals_RiskCalculationRow(t *testing.T) {
	d := L2CentralInternals()

	var row *Row
	for i := range d.Rows {
		if d.Rows[i].Title == "Risk Calculation" {
			row = &d.Rows[i]
			break
		}
	}

	require.NotNil(t, row, "Risk Calculation row should exist")

	// Verify panels
	assert.Equal(t, "risk duration", row.Panels[0].Title)
	assert.Equal(t, "reprocessor", row.Panels[1].Title)

	// Verify queries
	assert.Contains(t, row.Panels[0].Queries[0].Expr, "rox_central_risk_processing_duration")
	assert.Contains(t, row.Panels[1].Queries[0].Expr, "rox_central_reprocessor_duration_seconds")
}

func TestL2CentralInternals_BackgroundReprocessingRow(t *testing.T) {
	d := L2CentralInternals()

	var row *Row
	for i := range d.Rows {
		if d.Rows[i].Title == "Background Reprocessing" {
			row = &d.Rows[i]
			break
		}
	}

	require.NotNil(t, row, "Background Reprocessing row should exist")

	// Should have gap panel for missing metrics
	var gapPanel *Panel
	for i := range row.Panels {
		if row.Panels[i].GapNote != "" && !isDetailsLink(row.Panels[i].GapNote) {
			gapPanel = &row.Panels[i]
			break
		}
	}
	require.NotNil(t, gapPanel, "Should have gap panel for missing running/items-processed metrics")
	assert.Contains(t, gapPanel.GapNote, "running/items-processed")
}

func TestL2CentralInternals_PruningGCRow(t *testing.T) {
	d := L2CentralInternals()

	var row *Row
	for i := range d.Rows {
		if d.Rows[i].Title == "Pruning & GC" {
			row = &d.Rows[i]
			break
		}
	}

	require.NotNil(t, row, "Pruning & GC row should exist")
	require.Len(t, row.Panels, 4, "Should have 3 metric panels + 1 details link")

	// Verify panel titles
	assert.Equal(t, "prune duration", row.Panels[0].Title)
	assert.Equal(t, "process queue", row.Panels[1].Title)
	assert.Equal(t, "pruned indicators", row.Panels[2].Title)
}

func TestL2CentralInternals_NetworkAnalysisRow(t *testing.T) {
	d := L2CentralInternals()

	var row *Row
	for i := range d.Rows {
		if d.Rows[i].Title == "Network Analysis" {
			row = &d.Rows[i]
			break
		}
	}

	require.NotNil(t, row, "Network Analysis row should exist")

	// Verify panels
	assert.Equal(t, "flows received", row.Panels[0].Title)
	assert.Equal(t, "endpoints received", row.Panels[1].Title)

	// Verify queries
	assert.Contains(t, row.Panels[0].Queries[0].Expr, "rox_central_total_network_flows_central_received_counter")
	assert.Contains(t, row.Panels[1].Queries[0].Expr, "rox_central_total_network_endpoints_received_counter")
}

func TestL2CentralInternals_ReportGenerationRow(t *testing.T) {
	d := L2CentralInternals()

	var row *Row
	for i := range d.Rows {
		if d.Rows[i].Title == "Report Generation" {
			row = &d.Rows[i]
			break
		}
	}

	require.NotNil(t, row, "Report Generation row should exist")

	// Should have gap panel for missing Central-side report generation metrics
	var gapPanel *Panel
	for i := range row.Panels {
		if row.Panels[i].GapNote != "" && !isDetailsLink(row.Panels[i].GapNote) {
			gapPanel = &row.Panels[i]
			break
		}
	}
	require.NotNil(t, gapPanel, "Should have gap panel for missing report generation metrics")
	assert.Contains(t, gapPanel.GapNote, "Central-side report generation")

	// Should have compliance watchers panel
	var compliancePanel *Panel
	for i := range row.Panels {
		if row.Panels[i].Title == "compliance watchers" {
			compliancePanel = &row.Panels[i]
			break
		}
	}
	require.NotNil(t, compliancePanel)
	assert.Contains(t, compliancePanel.Queries[0].Expr, "rox_central_complianceoperator_scan_watchers_current")
}

func TestL2CentralInternals_APIUIRow(t *testing.T) {
	d := L2CentralInternals()

	var row *Row
	for i := range d.Rows {
		if d.Rows[i].Title == "API & UI" {
			row = &d.Rows[i]
			break
		}
	}

	require.NotNil(t, row, "API & UI row should exist")

	// Verify panels
	assert.Equal(t, "GraphQL p95", row.Panels[0].Title)
	assert.Equal(t, "gRPC errors", row.Panels[1].Title)

	// Verify queries
	assert.Contains(t, row.Panels[0].Queries[0].Expr, "rox_central_graphql_query_duration_bucket")
	assert.Contains(t, row.Panels[1].Queries[0].Expr, "rox_central_grpc_error")
}

func TestL2CentralInternals_ProducesValidJSON(t *testing.T) {
	d := L2CentralInternals()
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
	assert.Equal(t, "central-internals", unmarshaled["uid"])
	assert.Equal(t, "Central Internals", unmarshaled["title"])
}

func TestL2CentralInternals_AllPanelsHaveValidWidth(t *testing.T) {
	d := L2CentralInternals()

	for _, row := range d.Rows {
		for _, panel := range row.Panels {
			assert.Greater(t, panel.Width, 0, "Panel %s should have positive width", panel.Title)
			assert.LessOrEqual(t, panel.Width, 24, "Panel %s width should not exceed 24", panel.Title)
		}
	}
}

// isDetailsLink checks if a GapNote contains a details link to L3 dashboard
func isDetailsLink(note string) bool {
	return len(note) > 0 && (
		(note[0] == '[' && len(note) > 1) || // Starts with markdown link
		(len(note) >= 3 && note[0:3] == "###")) // Starts with markdown header
}
