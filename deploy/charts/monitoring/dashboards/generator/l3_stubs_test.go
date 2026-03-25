package generator

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestL3Stubs_Returns8Dashboards(t *testing.T) {
	stubs := L3Stubs()
	assert.Len(t, stubs, 8, "L3Stubs should return exactly 8 dashboards")
}

func TestL3Stubs_AllHaveLevel3Tag(t *testing.T) {
	stubs := L3Stubs()

	for _, d := range stubs {
		assert.Contains(t, d.Tags, "level-3", "Dashboard %s should have level-3 tag", d.UID)
		assert.Contains(t, d.Tags, "stackrox", "Dashboard %s should have stackrox tag", d.UID)
		assert.Contains(t, d.Tags, "central", "Dashboard %s should have central tag", d.UID)
	}
}

func TestL3Stubs_AllHaveBackLinkToCentralInternals(t *testing.T) {
	stubs := L3Stubs()

	for _, d := range stubs {
		require.Len(t, d.Links, 1, "Dashboard %s should have exactly 1 link", d.UID)
		assert.Equal(t, "← Central Internals", d.Links[0].Title, "Dashboard %s link title", d.UID)
		assert.Equal(t, "central-internals", d.Links[0].TargetUID, "Dashboard %s link target", d.UID)
	}
}

func TestL3Stubs_SpecificUIDs(t *testing.T) {
	stubs := L3Stubs()

	expectedUIDs := []string{
		"central-deployment-processing",
		"central-detection-alerts",
		"central-risk-calculation",
		"central-background-reprocessing",
		"central-pruning-gc",
		"central-network-analysis",
		"central-report-generation",
		"central-api-ui",
	}

	actualUIDs := make([]string, len(stubs))
	for i, d := range stubs {
		actualUIDs[i] = d.UID
	}

	for _, expectedUID := range expectedUIDs {
		assert.Contains(t, actualUIDs, expectedUID, "Expected UID %s to be present", expectedUID)
	}
}

func TestL3Stubs_AllProduceValidJSON(t *testing.T) {
	stubs := L3Stubs()

	for _, d := range stubs {
		t.Run(d.UID, func(t *testing.T) {
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
			assert.Equal(t, d.UID, unmarshaled["uid"])
			assert.Equal(t, d.Title, unmarshaled["title"])
		})
	}
}

func TestL3Stubs_DeploymentProcessing(t *testing.T) {
	d := findDashboard(L3Stubs(), "central-deployment-processing")
	require.NotNil(t, d, "central-deployment-processing dashboard should exist")

	assert.Equal(t, "Central: Deployment Processing", d.Title)
	assert.Contains(t, d.Tags, "deployment-processing")

	// Should have 2 rows: Resource Processing, Store Operations
	assert.Equal(t, 2, len(d.Rows))

	// Resource Processing row
	resourceRow := findRow(d.Rows, "Resource Processing")
	require.NotNil(t, resourceRow)
	require.Len(t, resourceRow.Panels, 2) // 2 metrics

	// Resources/sec panel
	assert.Equal(t, "Resources/sec", resourceRow.Panels[0].Title)
	assert.Equal(t, "timeseries", resourceRow.Panels[0].Type)
	assert.Contains(t, resourceRow.Panels[0].Queries[0].Expr, "rox_central_resource_processed_count")
	assert.Equal(t, "{{Resource}} - {{Operation}}", resourceRow.Panels[0].Queries[0].LegendFormat)

	// K8s Event Duration panel
	assert.Equal(t, "K8s Event Duration", resourceRow.Panels[1].Title)
	assert.Contains(t, resourceRow.Panels[1].Queries[0].Expr, "rox_central_k8s_event_processing_duration_bucket")

	// Store Operations row
	storeRow := findRow(d.Rows, "Store Operations")
	require.NotNil(t, storeRow)
	require.Len(t, storeRow.Panels, 2) // 1 metric + 1 gap

	// Postgres Op Duration panel
	assert.Equal(t, "Postgres Op Duration", storeRow.Panels[0].Title)
	assert.Contains(t, storeRow.Panels[0].Queries[0].Expr, "rox_central_postgres_op_duration_bucket")
	assert.Contains(t, storeRow.Panels[0].Queries[0].Expr, "Type=~\"deployments|pods|namespaces\"")

	// Gap panel
	assert.Equal(t, "GAP: Per-Fragment Handler Metrics", storeRow.Panels[1].Title)
	assert.Equal(t, 4, storeRow.Panels[1].Height)
	assert.Contains(t, storeRow.Panels[1].GapNote, "per-fragment handler metrics")
}

func TestL3Stubs_DetectionAlerts(t *testing.T) {
	d := findDashboard(L3Stubs(), "central-detection-alerts")
	require.NotNil(t, d)

	assert.Equal(t, "Central: Detection & Alerts", d.Title)
	assert.Contains(t, d.Tags, "detection-alerts")

	// Should have 2 rows
	assert.Equal(t, 2, len(d.Rows))

	// Detection row
	detectionRow := findRow(d.Rows, "Detection")
	require.NotNil(t, detectionRow)
	require.Len(t, detectionRow.Panels, 2) // 1 metric + 1 gap

	assert.Equal(t, "Process Filter", detectionRow.Panels[0].Title)
	assert.Contains(t, detectionRow.Panels[0].Queries[0].Expr, "rox_central_process_filter")

	assert.Equal(t, "GAP: Alert Generation Rate", detectionRow.Panels[1].Title)
	assert.Contains(t, detectionRow.Panels[1].GapNote, "central_detection_alerts_generated_total")
}

func TestL3Stubs_RiskCalculation(t *testing.T) {
	d := findDashboard(L3Stubs(), "central-risk-calculation")
	require.NotNil(t, d)

	assert.Equal(t, "Central: Risk Calculation", d.Title)
	assert.Contains(t, d.Tags, "risk-calculation")

	// Should have 2 rows
	assert.Equal(t, 2, len(d.Rows))

	// Risk Processing row
	riskRow := findRow(d.Rows, "Risk Processing")
	require.NotNil(t, riskRow)
	require.Len(t, riskRow.Panels, 2)

	assert.Equal(t, "Risk Duration", riskRow.Panels[0].Title)
	assert.Contains(t, riskRow.Panels[0].Queries[0].Expr, "rox_central_risk_processing_duration")

	assert.Equal(t, "Reprocessor Duration", riskRow.Panels[1].Title)
	assert.Contains(t, riskRow.Panels[1].Queries[0].Expr, "rox_central_reprocessor_duration_seconds")
}

func TestL3Stubs_BackgroundReprocessing(t *testing.T) {
	d := findDashboard(L3Stubs(), "central-background-reprocessing")
	require.NotNil(t, d)

	assert.Equal(t, "Central: Background Reprocessing", d.Title)
	assert.Contains(t, d.Tags, "background-reprocessing")

	// Should have 2 rows
	assert.Equal(t, 2, len(d.Rows))

	// Gaps row
	gapsRow := findRow(d.Rows, "Gaps — Loop Instrumentation")
	require.NotNil(t, gapsRow)
	require.Len(t, gapsRow.Panels, 1)

	// This gap is larger (Height=6)
	assert.Equal(t, 6, gapsRow.Panels[0].Height)
	assert.Contains(t, gapsRow.Panels[0].GapNote, "19+ background loops")
}

func TestL3Stubs_PruningGC(t *testing.T) {
	d := findDashboard(L3Stubs(), "central-pruning-gc")
	require.NotNil(t, d)

	assert.Equal(t, "Central: Pruning & GC", d.Title)
	assert.Contains(t, d.Tags, "pruning-gc")

	// Should have 2 rows
	assert.Equal(t, 2, len(d.Rows))

	// Pruning row
	pruningRow := findRow(d.Rows, "Pruning")
	require.NotNil(t, pruningRow)
	require.Len(t, pruningRow.Panels, 3)

	assert.Equal(t, "Prune Duration", pruningRow.Panels[0].Title)
	assert.Equal(t, 8, pruningRow.Panels[0].Width)

	// Additional Metrics row
	additionalRow := findRow(d.Rows, "Additional Metrics")
	require.NotNil(t, additionalRow)
	require.Len(t, additionalRow.Panels, 2)

	// Cache Hits/Misses panel should have 2 queries
	cachePanel := additionalRow.Panels[1]
	assert.Equal(t, "Cache Hits/Misses", cachePanel.Title)
	require.Len(t, cachePanel.Queries, 2)
	assert.Equal(t, "hits", cachePanel.Queries[0].LegendFormat)
	assert.Equal(t, "misses", cachePanel.Queries[1].LegendFormat)
}

func TestL3Stubs_NetworkAnalysis(t *testing.T) {
	d := findDashboard(L3Stubs(), "central-network-analysis")
	require.NotNil(t, d)

	assert.Equal(t, "Central: Network Analysis", d.Title)
	assert.Contains(t, d.Tags, "network-analysis")

	// Flows & Endpoints row
	flowsRow := findRow(d.Rows, "Flows & Endpoints")
	require.NotNil(t, flowsRow)
	require.Len(t, flowsRow.Panels, 2)

	assert.Equal(t, "Flows Received", flowsRow.Panels[0].Title)
	assert.Contains(t, flowsRow.Panels[0].Queries[0].Expr, "rox_central_total_network_flows_central_received_counter")
}

func TestL3Stubs_ReportGeneration(t *testing.T) {
	d := findDashboard(L3Stubs(), "central-report-generation")
	require.NotNil(t, d)

	assert.Equal(t, "Central: Report Generation", d.Title)
	assert.Contains(t, d.Tags, "report-generation")

	// Compliance Operator Reports row
	complianceRow := findRow(d.Rows, "Compliance Operator Reports")
	require.NotNil(t, complianceRow)
	require.Len(t, complianceRow.Panels, 3)

	assert.Equal(t, "Scan Watchers", complianceRow.Panels[0].Title)
	assert.Equal(t, 8, complianceRow.Panels[0].Width)
}

func TestL3Stubs_APIUI(t *testing.T) {
	d := findDashboard(L3Stubs(), "central-api-ui")
	require.NotNil(t, d)

	assert.Equal(t, "Central: API & UI", d.Title)
	assert.Contains(t, d.Tags, "api-ui")

	// GraphQL row
	graphqlRow := findRow(d.Rows, "GraphQL")
	require.NotNil(t, graphqlRow)
	require.Len(t, graphqlRow.Panels, 2)

	assert.Equal(t, "Query Duration p95", graphqlRow.Panels[0].Title)
	assert.Contains(t, graphqlRow.Panels[0].Queries[0].Expr, "rox_central_graphql_query_duration_bucket")

	// Gaps row
	gapsRow := findRow(d.Rows, "Gaps")
	require.NotNil(t, gapsRow)
	require.Len(t, gapsRow.Panels, 2)

	assert.Equal(t, "GAP: Per-Endpoint Metrics", gapsRow.Panels[0].Title)
	assert.Equal(t, "GAP: UI Page Load", gapsRow.Panels[1].Title)
}

func TestL3Stubs_AllPanelsHaveValidDimensions(t *testing.T) {
	stubs := L3Stubs()

	for _, d := range stubs {
		for _, row := range d.Rows {
			for _, panel := range row.Panels {
				assert.Greater(t, panel.Width, 0, "Panel %s in %s should have positive width", panel.Title, d.UID)
				assert.LessOrEqual(t, panel.Width, 24, "Panel %s in %s width should not exceed 24", panel.Title, d.UID)
				assert.Greater(t, panel.Height, 0, "Panel %s in %s should have positive height", panel.Title, d.UID)
			}
		}
	}
}

// Helper functions
func findDashboard(dashboards []Dashboard, uid string) *Dashboard {
	for i := range dashboards {
		if dashboards[i].UID == uid {
			return &dashboards[i]
		}
	}
	return nil
}

func findRow(rows []Row, title string) *Row {
	for i := range rows {
		if rows[i].Title == title {
			return &rows[i]
		}
	}
	return nil
}
