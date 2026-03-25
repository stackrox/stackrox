package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPanel_Timeseries(t *testing.T) {
	p := Panel{
		Title:  "Events per Second",
		Width:  12,
		Height: 8,
		Type:   "timeseries",
		Unit:   "ops",
		Queries: []Query{
			{Expr: `rate(rox_central_sensor_event_queue{Operation="remove"}[5m])`, LegendFormat: "{{Type}}", RefID: "A"},
		},
	}

	result := p.generate(1, 0, 0) // id=1, x=0, y=0

	assert.Equal(t, "Events per Second", result["title"])
	assert.Equal(t, "timeseries", result["type"])
	assert.Equal(t, 1, result["id"])

	gridPos := result["gridPos"].(map[string]int)
	assert.Equal(t, 12, gridPos["w"])
	assert.Equal(t, 8, gridPos["h"])
	assert.Equal(t, 0, gridPos["x"])
	assert.Equal(t, 0, gridPos["y"])

	targets, ok := result["targets"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, targets, 1)
	assert.Contains(t, targets[0]["expr"], "rox_central_sensor_event_queue")
}

func TestPanel_GapAnnotation(t *testing.T) {
	p := Panel{
		Title:   "Queue Depth",
		Width:   12,
		Height:  4,
		GapNote: "**Metric Needed**: `central_sensor_ingestion_queue_depth` — Worker Queue shards lack depth gauges.",
	}

	result := p.generate(2, 0, 8)

	assert.Equal(t, "text", result["type"])
	options := result["options"].(map[string]any)
	assert.Contains(t, options["content"], "Metric Needed")
	assert.Equal(t, "markdown", options["mode"])
}

func TestPanel_Stat(t *testing.T) {
	p := Panel{
		Title:  "Sensors Connected",
		Width:  4,
		Height: 4,
		Type:   "stat",
		Queries: []Query{
			{Expr: `count(rox_central_sensor_connected{connection_state="connected"})`, RefID: "A"},
		},
	}

	result := p.generate(3, 0, 0)
	assert.Equal(t, "stat", result["type"])
}

func TestPanel_Gauge(t *testing.T) {
	p := Panel{
		Title:  "CPU Usage",
		Width:  6,
		Height: 6,
		Type:   "gauge",
		Unit:   "percentunit",
		Queries: []Query{
			{Expr: `rate(process_cpu_seconds_total[5m])`, RefID: "A"},
		},
		Thresholds: []Threshold{
			{Value: 0.7, Color: "yellow"},
			{Value: 0.9, Color: "red"},
		},
	}

	result := p.generate(4, 0, 0)
	assert.Equal(t, "gauge", result["type"])
	assert.Equal(t, "percentunit", result["fieldConfig"].(map[string]any)["defaults"].(map[string]any)["unit"])

	// Check thresholds
	defaults := result["fieldConfig"].(map[string]any)["defaults"].(map[string]any)
	thresholds := defaults["thresholds"].(map[string]any)
	steps := thresholds["steps"].([]map[string]any)
	require.Len(t, steps, 3) // base green + 2 thresholds

	assert.Equal(t, "green", steps[0]["color"])
	assert.Equal(t, "yellow", steps[1]["color"])
	assert.Equal(t, 0.7, steps[1]["value"])
	assert.Equal(t, "red", steps[2]["color"])
	assert.Equal(t, 0.9, steps[2]["value"])
}

func TestPanel_MultipleQueries(t *testing.T) {
	p := Panel{
		Title:  "Multiple Metrics",
		Width:  12,
		Height: 8,
		Type:   "timeseries",
		Queries: []Query{
			{Expr: `metric_a`, LegendFormat: "A", RefID: "A"},
			{Expr: `metric_b`, LegendFormat: "B", RefID: "B"},
			{Expr: `metric_c`, LegendFormat: "C", RefID: "C"},
		},
	}

	result := p.generate(5, 0, 0)
	targets, ok := result["targets"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, targets, 3)

	assert.Equal(t, "A", targets[0]["refId"])
	assert.Equal(t, "B", targets[1]["refId"])
	assert.Equal(t, "C", targets[2]["refId"])
}

func TestRow_Generate(t *testing.T) {
	d := Dashboard{
		UID:   "test",
		Title: "Test",
		Rows: []Row{
			{
				Title: "Sensor Ingestion",
				Panels: []Panel{
					{Title: "P1", Width: 12, Height: 8, Type: "timeseries"},
					{Title: "P2", Width: 12, Height: 8, Type: "timeseries"},
				},
			},
		},
	}

	result := d.Generate()
	panels, ok := result["panels"].([]map[string]any)
	require.True(t, ok)

	// Row header + 2 panels = 3
	require.Len(t, panels, 3)
	assert.Equal(t, "row", panels[0]["type"])
	assert.Equal(t, "P1", panels[1]["title"])
	assert.Equal(t, "P2", panels[2]["title"])

	// P2 should be at x=12 (next to P1)
	p2Grid := panels[2]["gridPos"].(map[string]int)
	assert.Equal(t, 12, p2Grid["x"])
}

func TestRow_PanelWrapping(t *testing.T) {
	d := Dashboard{
		UID:   "test",
		Title: "Test",
		Rows: []Row{
			{
				Title: "Test Row",
				Panels: []Panel{
					{Title: "P1", Width: 12, Height: 8, Type: "timeseries"},
					{Title: "P2", Width: 12, Height: 8, Type: "timeseries"},
					{Title: "P3", Width: 12, Height: 8, Type: "timeseries"},
				},
			},
		},
	}

	result := d.Generate()
	panels, ok := result["panels"].([]map[string]any)
	require.True(t, ok)

	// Row header + 3 panels = 4
	require.Len(t, panels, 4)

	// P1 at x=0, y should be after row header
	p1Grid := panels[1]["gridPos"].(map[string]int)
	assert.Equal(t, 0, p1Grid["x"])

	// P2 at x=12 (wraps because 12+12 > 24)
	p2Grid := panels[2]["gridPos"].(map[string]int)
	assert.Equal(t, 12, p2Grid["x"])
	assert.Equal(t, p1Grid["y"], p2Grid["y"]) // Same row

	// P3 should wrap to next line
	p3Grid := panels[3]["gridPos"].(map[string]int)
	assert.Equal(t, 0, p3Grid["x"])
	assert.Equal(t, p1Grid["y"]+8, p3Grid["y"]) // New row, y increments by height
}
