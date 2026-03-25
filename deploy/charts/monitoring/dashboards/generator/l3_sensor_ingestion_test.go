package generator

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestL3SensorIngestion_BasicMetadata(t *testing.T) {
	d := L3SensorIngestion()

	assert.Equal(t, "central-sensor-ingestion", d.UID)
	assert.Equal(t, "Central: Sensor Ingestion", d.Title)
	assert.Contains(t, d.Tags, "level-3")
	assert.Contains(t, d.Tags, "sensor-ingestion")
	assert.Contains(t, d.Tags, "stackrox")
	assert.Contains(t, d.Tags, "central")
}

func TestL3SensorIngestion_HasBackLinkToCentralInternals(t *testing.T) {
	d := L3SensorIngestion()

	require.Len(t, d.Links, 1)
	assert.Equal(t, "← Central Internals", d.Links[0].Title)
	assert.Equal(t, "central-internals", d.Links[0].TargetUID)
}

func TestL3SensorIngestion_HasRequiredRows(t *testing.T) {
	d := L3SensorIngestion()

	require.Equal(t, 5, len(d.Rows), "Should have exactly 5 rows")

	// Verify row titles
	rowTitles := make([]string, len(d.Rows))
	for i, row := range d.Rows {
		rowTitles[i] = row.Title
	}

	assert.Contains(t, rowTitles, "Connection Status")
	assert.Contains(t, rowTitles, "Deduper")
	assert.Contains(t, rowTitles, "Worker Queue")
	assert.Contains(t, rowTitles, "Pipeline Processing")
	assert.Contains(t, rowTitles, "Messages Not Sent")
}

func TestL3SensorIngestion_ConnectionStatusRow(t *testing.T) {
	d := L3SensorIngestion()

	var row *Row
	for i := range d.Rows {
		if d.Rows[i].Title == "Connection Status" {
			row = &d.Rows[i]
			break
		}
	}

	require.NotNil(t, row)
	require.Len(t, row.Panels, 2)

	// Verify Sensors Connected panel
	assert.Equal(t, "Sensors Connected", row.Panels[0].Title)
	assert.Equal(t, "stat", row.Panels[0].Type)
	assert.Equal(t, 8, row.Panels[0].Width)
	assert.Equal(t, `count(rox_central_sensor_connected{connection_state="connected"})`, row.Panels[0].Queries[0].Expr)

	// Verify Connection Events panel
	assert.Equal(t, "Connection Events", row.Panels[1].Title)
	assert.Equal(t, "timeseries", row.Panels[1].Type)
	assert.Equal(t, 16, row.Panels[1].Width)
	assert.Equal(t, `rate(rox_central_sensor_connected[5m])`, row.Panels[1].Queries[0].Expr)
	assert.Equal(t, `{{connection_state}}`, row.Panels[1].Queries[0].LegendFormat)
}

func TestL3SensorIngestion_DeduperRow(t *testing.T) {
	d := L3SensorIngestion()

	var row *Row
	for i := range d.Rows {
		if d.Rows[i].Title == "Deduper" {
			row = &d.Rows[i]
			break
		}
	}

	require.NotNil(t, row)
	require.Len(t, row.Panels, 4)

	// Verify Deduper Throughput panel
	assert.Equal(t, "Deduper Throughput", row.Panels[0].Title)
	assert.Equal(t, "timeseries", row.Panels[0].Type)
	assert.Equal(t, 12, row.Panels[0].Width)
	assert.Equal(t, `rate(rox_central_sensor_event_deduper[5m])`, row.Panels[0].Queries[0].Expr)
	assert.Equal(t, `{{status}} - {{type}}`, row.Panels[0].Queries[0].LegendFormat)

	// Verify Deduper Hit Rate panel
	assert.Equal(t, "Deduper Hit Rate", row.Panels[1].Title)
	assert.Equal(t, "timeseries", row.Panels[1].Type)
	assert.Equal(t, 12, row.Panels[1].Width)
	assert.Contains(t, row.Panels[1].Queries[0].Expr, `rate(rox_central_sensor_event_deduper{status="deduplicated"}[5m])`)
	assert.Equal(t, `dedup rate`, row.Panels[1].Queries[0].LegendFormat)

	// Verify Hash Store Size panel
	assert.Equal(t, "Hash Store Size", row.Panels[2].Title)
	assert.Equal(t, "timeseries", row.Panels[2].Type)
	assert.Equal(t, 12, row.Panels[2].Width)
	assert.Equal(t, `rox_central_deduping_hash_size`, row.Panels[2].Queries[0].Expr)
	assert.Equal(t, `{{cluster}}`, row.Panels[2].Queries[0].LegendFormat)

	// Verify Hash Operations panel
	assert.Equal(t, "Hash Operations", row.Panels[3].Title)
	assert.Equal(t, "timeseries", row.Panels[3].Type)
	assert.Equal(t, 12, row.Panels[3].Width)
	assert.Equal(t, `rate(rox_central_deduping_hash_count[5m])`, row.Panels[3].Queries[0].Expr)
	assert.Equal(t, `{{ResourceType}} - {{Operation}}`, row.Panels[3].Queries[0].LegendFormat)
}

func TestL3SensorIngestion_WorkerQueueRow(t *testing.T) {
	d := L3SensorIngestion()

	var row *Row
	for i := range d.Rows {
		if d.Rows[i].Title == "Worker Queue" {
			row = &d.Rows[i]
			break
		}
	}

	require.NotNil(t, row)
	require.Len(t, row.Panels, 4, "Worker Queue should have 4 panels (2 metrics + 2 gaps)")

	// Verify Events Processed panel
	assert.Equal(t, "Events Processed", row.Panels[0].Title)
	assert.Equal(t, "timeseries", row.Panels[0].Type)
	assert.Equal(t, 12, row.Panels[0].Width)
	assert.Equal(t, `rate(rox_central_sensor_event_queue[5m])`, row.Panels[0].Queries[0].Expr)
	assert.Equal(t, `{{Operation}} - {{Type}}`, row.Panels[0].Queries[0].LegendFormat)

	// Verify Processing Duration panel
	assert.Equal(t, "Processing Duration", row.Panels[1].Title)
	assert.Equal(t, "timeseries", row.Panels[1].Type)
	assert.Equal(t, 12, row.Panels[1].Width)
	assert.Equal(t, `histogram_quantile(0.95, rate(rox_central_sensor_event_duration_bucket[5m]))`, row.Panels[1].Queries[0].Expr)
	assert.Equal(t, `p95 - {{Type}}`, row.Panels[1].Queries[0].LegendFormat)

	// Verify Queue Depth gap panel
	assert.Equal(t, "GAP: Queue Depth", row.Panels[2].Title)
	assert.Equal(t, 12, row.Panels[2].Width)
	assert.Equal(t, 4, row.Panels[2].Height)
	assert.Contains(t, row.Panels[2].GapNote, "central_sensor_ingestion_queue_depth")
	assert.Contains(t, row.Panels[2].GapNote, "Cannot answer \"is the queue backing up?\"")

	// Verify In-Flight gap panel
	assert.Equal(t, "GAP: In-Flight", row.Panels[3].Title)
	assert.Equal(t, 12, row.Panels[3].Width)
	assert.Equal(t, 4, row.Panels[3].Height)
	assert.Contains(t, row.Panels[3].GapNote, "central_sensor_ingestion_in_flight")
	assert.Contains(t, row.Panels[3].GapNote, "items currently being processed")
}

func TestL3SensorIngestion_PipelineProcessingRow(t *testing.T) {
	d := L3SensorIngestion()

	var row *Row
	for i := range d.Rows {
		if d.Rows[i].Title == "Pipeline Processing" {
			row = &d.Rows[i]
			break
		}
	}

	require.NotNil(t, row)
	require.Len(t, row.Panels, 4, "Pipeline Processing should have 4 panels (3 metrics + 1 gap)")

	// Verify Resources Processed panel
	assert.Equal(t, "Resources Processed", row.Panels[0].Title)
	assert.Equal(t, "timeseries", row.Panels[0].Type)
	assert.Equal(t, 12, row.Panels[0].Width)
	assert.Equal(t, `rate(rox_central_resource_processed_count[5m])`, row.Panels[0].Queries[0].Expr)
	assert.Equal(t, `{{Resource}} - {{Operation}}`, row.Panels[0].Queries[0].LegendFormat)

	// Verify Pipeline Panics panel
	assert.Equal(t, "Pipeline Panics", row.Panels[1].Title)
	assert.Equal(t, "timeseries", row.Panels[1].Type)
	assert.Equal(t, 12, row.Panels[1].Width)
	assert.Equal(t, `rate(rox_central_pipeline_panics[5m])`, row.Panels[1].Queries[0].Expr)
	assert.Equal(t, `{{resource}}`, row.Panels[1].Queries[0].LegendFormat)

	// Verify K8s Event Processing panel
	assert.Equal(t, "K8s Event Processing", row.Panels[2].Title)
	assert.Equal(t, "timeseries", row.Panels[2].Type)
	assert.Equal(t, 12, row.Panels[2].Width)
	assert.Equal(t, `histogram_quantile(0.95, rate(rox_central_k8s_event_processing_duration_bucket[5m]))`, row.Panels[2].Queries[0].Expr)
	assert.Equal(t, `p95 - {{Resource}}`, row.Panels[2].Queries[0].LegendFormat)

	// Verify Per-Fragment Metrics gap panel
	assert.Equal(t, "GAP: Per-Fragment Metrics", row.Panels[3].Title)
	assert.Equal(t, 12, row.Panels[3].Width)
	assert.Equal(t, 4, row.Panels[3].Height)
	assert.Contains(t, row.Panels[3].GapNote, "Per-fragment processing counts and durations")
	assert.Contains(t, row.Panels[3].GapNote, "25 pipeline fragments exist")
}

func TestL3SensorIngestion_MessagesNotSentRow(t *testing.T) {
	d := L3SensorIngestion()

	var row *Row
	for i := range d.Rows {
		if d.Rows[i].Title == "Messages Not Sent" {
			row = &d.Rows[i]
			break
		}
	}

	require.NotNil(t, row)
	require.Len(t, row.Panels, 1)

	// Verify Failed Sends to Sensor panel
	assert.Equal(t, "Failed Sends to Sensor", row.Panels[0].Title)
	assert.Equal(t, "timeseries", row.Panels[0].Type)
	assert.Equal(t, 24, row.Panels[0].Width)
	assert.Equal(t, `rate(rox_central_msg_to_sensor_not_sent_count[5m])`, row.Panels[0].Queries[0].Expr)
	assert.Equal(t, `{{type}} - {{reason}}`, row.Panels[0].Queries[0].LegendFormat)
}

func TestL3SensorIngestion_ProducesValidJSON(t *testing.T) {
	d := L3SensorIngestion()
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
	assert.Equal(t, "central-sensor-ingestion", unmarshaled["uid"])
	assert.Equal(t, "Central: Sensor Ingestion", unmarshaled["title"])
}

func TestL3SensorIngestion_AllPanelsHaveValidWidth(t *testing.T) {
	d := L3SensorIngestion()

	for _, row := range d.Rows {
		for _, panel := range row.Panels {
			assert.Greater(t, panel.Width, 0, "Panel %s should have positive width", panel.Title)
			assert.LessOrEqual(t, panel.Width, 24, "Panel %s width should not exceed 24", panel.Title)
		}
	}
}
