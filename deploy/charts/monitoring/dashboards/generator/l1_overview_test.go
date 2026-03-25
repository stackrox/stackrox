package generator

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestL1Overview_BasicMetadata(t *testing.T) {
	d := L1Overview()

	assert.Equal(t, "stackrox-overview", d.UID)
	assert.Equal(t, "StackRox Overview", d.Title)
	assert.Contains(t, d.Tags, "level-1")
	assert.Contains(t, d.Tags, "stackrox")
	assert.Contains(t, d.Tags, "overview")
}

func TestL1Overview_HasLinkToCentralInternals(t *testing.T) {
	d := L1Overview()

	require.Len(t, d.Links, 1)
	assert.Equal(t, "Central Internals", d.Links[0].Title)
	assert.Equal(t, "central-internals", d.Links[0].TargetUID)
}

func TestL1Overview_HasRequiredRows(t *testing.T) {
	d := L1Overview()

	require.GreaterOrEqual(t, len(d.Rows), 3, "Should have at least 3 rows")

	// Verify row titles
	rowTitles := make([]string, len(d.Rows))
	for i, row := range d.Rows {
		rowTitles[i] = row.Title
	}

	assert.Contains(t, rowTitles, "Service Health")
	assert.Contains(t, rowTitles, "Connected Sensors")
	assert.Contains(t, rowTitles, "Database")
}

func TestL1Overview_ServiceHealthRow(t *testing.T) {
	d := L1Overview()

	var serviceHealthRow *Row
	for i := range d.Rows {
		if d.Rows[i].Title == "Service Health" {
			serviceHealthRow = &d.Rows[i]
			break
		}
	}

	require.NotNil(t, serviceHealthRow, "Service Health row should exist")
	require.Len(t, serviceHealthRow.Panels, 5, "Service Health should have 5 panels")

	// Verify panel titles
	panelTitles := make([]string, len(serviceHealthRow.Panels))
	for i, panel := range serviceHealthRow.Panels {
		panelTitles[i] = panel.Title
	}

	assert.Contains(t, panelTitles, "Central Up")
	assert.Contains(t, panelTitles, "Central CPU")
	assert.Contains(t, panelTitles, "Central Memory")
	assert.Contains(t, panelTitles, "Central Goroutines")
	assert.Contains(t, panelTitles, "Central Version")

	// Verify Central Up panel
	var centralUpPanel *Panel
	for i := range serviceHealthRow.Panels {
		if serviceHealthRow.Panels[i].Title == "Central Up" {
			centralUpPanel = &serviceHealthRow.Panels[i]
			break
		}
	}
	require.NotNil(t, centralUpPanel)
	assert.Equal(t, "stat", centralUpPanel.Type)
	assert.Equal(t, 4, centralUpPanel.Width)
	require.Len(t, centralUpPanel.Queries, 1)
	assert.Equal(t, `up{job="central"}`, centralUpPanel.Queries[0].Expr)

	// Verify Central CPU panel
	var centralCPUPanel *Panel
	for i := range serviceHealthRow.Panels {
		if serviceHealthRow.Panels[i].Title == "Central CPU" {
			centralCPUPanel = &serviceHealthRow.Panels[i]
			break
		}
	}
	require.NotNil(t, centralCPUPanel)
	assert.Equal(t, "timeseries", centralCPUPanel.Type)
	assert.Equal(t, 5, centralCPUPanel.Width)
	require.Len(t, centralCPUPanel.Queries, 1)
	assert.Equal(t, `rate(process_cpu_seconds_total{job="central"}[5m])`, centralCPUPanel.Queries[0].Expr)

	// Verify Central Memory panel
	var centralMemoryPanel *Panel
	for i := range serviceHealthRow.Panels {
		if serviceHealthRow.Panels[i].Title == "Central Memory" {
			centralMemoryPanel = &serviceHealthRow.Panels[i]
			break
		}
	}
	require.NotNil(t, centralMemoryPanel)
	assert.Equal(t, "timeseries", centralMemoryPanel.Type)
	assert.Equal(t, 5, centralMemoryPanel.Width)
	require.Len(t, centralMemoryPanel.Queries, 1)
	assert.Equal(t, `process_resident_memory_bytes{job="central"}`, centralMemoryPanel.Queries[0].Expr)

	// Verify Central Goroutines panel
	var centralGoroutinesPanel *Panel
	for i := range serviceHealthRow.Panels {
		if serviceHealthRow.Panels[i].Title == "Central Goroutines" {
			centralGoroutinesPanel = &serviceHealthRow.Panels[i]
			break
		}
	}
	require.NotNil(t, centralGoroutinesPanel)
	assert.Equal(t, "stat", centralGoroutinesPanel.Type)
	assert.Equal(t, 5, centralGoroutinesPanel.Width)
	require.Len(t, centralGoroutinesPanel.Queries, 1)
	assert.Equal(t, `go_goroutines{job="central"}`, centralGoroutinesPanel.Queries[0].Expr)

	// Verify Central Version panel
	var centralVersionPanel *Panel
	for i := range serviceHealthRow.Panels {
		if serviceHealthRow.Panels[i].Title == "Central Version" {
			centralVersionPanel = &serviceHealthRow.Panels[i]
			break
		}
	}
	require.NotNil(t, centralVersionPanel)
	assert.Equal(t, "stat", centralVersionPanel.Type)
	assert.Equal(t, 5, centralVersionPanel.Width)
	require.Len(t, centralVersionPanel.Queries, 1)
	assert.Equal(t, `rox_central_info`, centralVersionPanel.Queries[0].Expr)
	assert.Equal(t, `{{central_version}}`, centralVersionPanel.Queries[0].LegendFormat)
}

func TestL1Overview_ConnectedSensorsRow(t *testing.T) {
	d := L1Overview()

	var connectedSensorsRow *Row
	for i := range d.Rows {
		if d.Rows[i].Title == "Connected Sensors" {
			connectedSensorsRow = &d.Rows[i]
			break
		}
	}

	require.NotNil(t, connectedSensorsRow, "Connected Sensors row should exist")
	require.Len(t, connectedSensorsRow.Panels, 4, "Connected Sensors should have 4 panels")

	// Verify panel titles
	panelTitles := make([]string, len(connectedSensorsRow.Panels))
	for i, panel := range connectedSensorsRow.Panels {
		panelTitles[i] = panel.Title
	}

	assert.Contains(t, panelTitles, "Sensors Connected")
	assert.Contains(t, panelTitles, "Secured Clusters")
	assert.Contains(t, panelTitles, "Secured Nodes")
	assert.Contains(t, panelTitles, "Secured vCPUs")

	// Verify Sensors Connected panel
	var sensorsConnectedPanel *Panel
	for i := range connectedSensorsRow.Panels {
		if connectedSensorsRow.Panels[i].Title == "Sensors Connected" {
			sensorsConnectedPanel = &connectedSensorsRow.Panels[i]
			break
		}
	}
	require.NotNil(t, sensorsConnectedPanel)
	assert.Equal(t, "stat", sensorsConnectedPanel.Type)
	assert.Equal(t, 6, sensorsConnectedPanel.Width)
	require.Len(t, sensorsConnectedPanel.Queries, 1)
	assert.Equal(t, `count by (connection_state) (rox_central_sensor_connected)`, sensorsConnectedPanel.Queries[0].Expr)
}

func TestL1Overview_DatabaseRow(t *testing.T) {
	d := L1Overview()

	var databaseRow *Row
	for i := range d.Rows {
		if d.Rows[i].Title == "Database" {
			databaseRow = &d.Rows[i]
			break
		}
	}

	require.NotNil(t, databaseRow, "Database row should exist")
	require.Len(t, databaseRow.Panels, 4, "Database should have 4 panels")

	// Verify panel titles
	panelTitles := make([]string, len(databaseRow.Panels))
	for i, panel := range databaseRow.Panels {
		panelTitles[i] = panel.Title
	}

	assert.Contains(t, panelTitles, "Postgres Connected")
	assert.Contains(t, panelTitles, "DB Size")
	assert.Contains(t, panelTitles, "Active Connections")
	assert.Contains(t, panelTitles, "Available Space")

	// Verify Postgres Connected panel
	var postgresConnectedPanel *Panel
	for i := range databaseRow.Panels {
		if databaseRow.Panels[i].Title == "Postgres Connected" {
			postgresConnectedPanel = &databaseRow.Panels[i]
			break
		}
	}
	require.NotNil(t, postgresConnectedPanel)
	assert.Equal(t, "stat", postgresConnectedPanel.Type)
	assert.Equal(t, 6, postgresConnectedPanel.Width)
	require.Len(t, postgresConnectedPanel.Queries, 1)
	assert.Equal(t, `rox_central_postgres_connected`, postgresConnectedPanel.Queries[0].Expr)

	// Verify DB Size panel has bytes unit
	var dbSizePanel *Panel
	for i := range databaseRow.Panels {
		if databaseRow.Panels[i].Title == "DB Size" {
			dbSizePanel = &databaseRow.Panels[i]
			break
		}
	}
	require.NotNil(t, dbSizePanel)
	assert.Equal(t, "bytes", dbSizePanel.Unit)
	assert.Equal(t, `rox_central_postgres_total_size_bytes`, dbSizePanel.Queries[0].Expr)

	// Verify Available Space panel has bytes unit
	var availableSpacePanel *Panel
	for i := range databaseRow.Panels {
		if databaseRow.Panels[i].Title == "Available Space" {
			availableSpacePanel = &databaseRow.Panels[i]
			break
		}
	}
	require.NotNil(t, availableSpacePanel)
	assert.Equal(t, "bytes", availableSpacePanel.Unit)
	assert.Equal(t, `rox_central_postgres_available_size_bytes`, availableSpacePanel.Queries[0].Expr)
}

func TestL1Overview_ProducesValidJSON(t *testing.T) {
	d := L1Overview()
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
	assert.Equal(t, "stackrox-overview", unmarshaled["uid"])
	assert.Equal(t, "StackRox Overview", unmarshaled["title"])
}

func TestL1Overview_AllPanelsHaveValidWidth(t *testing.T) {
	d := L1Overview()

	for _, row := range d.Rows {
		rowWidth := 0
		for _, panel := range row.Panels {
			assert.Greater(t, panel.Width, 0, "Panel %s should have positive width", panel.Title)
			assert.LessOrEqual(t, panel.Width, 24, "Panel %s width should not exceed 24", panel.Title)
			rowWidth += panel.Width
		}
		// Each row should have reasonable total width (allowing wrapping)
		assert.Greater(t, rowWidth, 0, "Row %s should have panels with total width > 0", row.Title)
	}
}
