package generator

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDashboard_Generate_BasicStructure(t *testing.T) {
	d := Dashboard{
		UID:   "test-uid",
		Title: "Test Dashboard",
		Tags:  []string{"stackrox", "test"},
	}

	result := d.Generate()

	assert.Equal(t, "test-uid", result["uid"])
	assert.Equal(t, "Test Dashboard", result["title"])
	assert.Equal(t, []string{"stackrox", "test"}, result["tags"])
	assert.Equal(t, true, result["editable"])
	assert.NotNil(t, result["time"])

	// Validate it produces valid JSON
	_, err := json.Marshal(result)
	require.NoError(t, err)
}

func TestDashboard_Generate_WithLinks(t *testing.T) {
	d := Dashboard{
		UID:   "overview",
		Title: "Overview",
		Links: []DashboardLink{
			{Title: "Central Internals", TargetUID: "central-internals", Type: "link"},
		},
	}

	result := d.Generate()

	links, ok := result["links"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, links, 1)
	assert.Equal(t, "Central Internals", links[0]["title"])
	assert.Contains(t, links[0]["url"], "central-internals")
}

func TestDashboard_Generate_WithVariables(t *testing.T) {
	d := Dashboard{
		UID:   "test",
		Title: "Test",
		Templating: []Variable{
			{Name: "datasource", Type: "datasource", Label: "Data Source"},
			{Name: "cluster", Type: "query", Query: "label_values(cluster)", Label: "Cluster"},
		},
	}

	result := d.Generate()

	templating, ok := result["templating"].(map[string]any)
	require.True(t, ok)

	list, ok := templating["list"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, list, 2)

	assert.Equal(t, "datasource", list[0]["name"])
	assert.Equal(t, "cluster", list[1]["name"])
}

func TestDashboard_Generate_WithRows(t *testing.T) {
	d := Dashboard{
		UID:   "test",
		Title: "Test",
		Rows: []Row{
			{
				Title: "Section 1",
				Panels: []Panel{
					{Title: "Panel 1", Width: 12, Height: 8, Type: "timeseries"},
				},
			},
		},
	}

	result := d.Generate()

	panels, ok := result["panels"].([]map[string]any)
	require.True(t, ok)
	// Should have row header + 1 panel = 2 items
	require.Len(t, panels, 2)

	// First item should be row
	assert.Equal(t, "row", panels[0]["type"])
	assert.Equal(t, "Section 1", panels[0]["title"])

	// Second item should be panel
	assert.Equal(t, "Panel 1", panels[1]["title"])
}
