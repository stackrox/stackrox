package generator

// Dashboard represents a Grafana dashboard
type Dashboard struct {
	UID        string
	Title      string
	Tags       []string
	Links      []DashboardLink // Links to other dashboards
	Rows       []Row
	Templating []Variable
}

// DashboardLink represents a link to another dashboard
type DashboardLink struct {
	Title     string
	TargetUID string
	Type      string // "link" or "dashboards"
}

// Variable represents a dashboard template variable
type Variable struct {
	Name  string
	Type  string // "datasource", "query", "custom"
	Query string
	Label string
}

// Row is a collapsible row of panels
type Row struct {
	Title  string
	Panels []Panel
}

// Generate produces a map[string]any that marshals to valid Grafana JSON
func (d *Dashboard) Generate() map[string]any {
	result := map[string]any{
		"uid":       d.UID,
		"title":     d.Title,
		"tags":      d.Tags,
		"editable":  true,
		"schemaVersion": 27,
		"version":   0,
		"refresh":   "30s",
		"time": map[string]any{
			"from": "now-6h",
			"to":   "now",
		},
		"timepicker": map[string]any{},
		"timezone":   "",
		"annotations": map[string]any{
			"list": []map[string]any{
				{
					"builtIn":   1,
					"datasource": map[string]any{
						"type": "grafana",
						"uid":  "-- Grafana --",
					},
					"enable":    true,
					"hide":      true,
					"iconColor": "rgba(0, 211, 255, 1)",
					"name":      "Annotations & Alerts",
					"type":      "dashboard",
				},
			},
		},
		"links":  d.generateLinks(),
		"panels": d.generatePanels(),
	}

	// Add templating if present
	if len(d.Templating) > 0 {
		result["templating"] = d.generateTemplating()
	}

	return result
}

func (d *Dashboard) generateLinks() []map[string]any {
	if len(d.Links) == 0 {
		return []map[string]any{}
	}

	links := make([]map[string]any, 0, len(d.Links))
	for _, link := range d.Links {
		linkType := link.Type
		if linkType == "" {
			linkType = "link"
		}

		l := map[string]any{
			"asDropdown":  false,
			"icon":        "external link",
			"includeVars": false,
			"keepTime":    true,
			"tags":        []string{},
			"targetBlank": false,
			"title":       link.Title,
			"tooltip":     "",
			"type":        linkType,
			"url":         "/d/" + link.TargetUID,
		}
		links = append(links, l)
	}

	return links
}

func (d *Dashboard) generateTemplating() map[string]any {
	list := make([]map[string]any, 0, len(d.Templating))

	for _, v := range d.Templating {
		variable := map[string]any{
			"name":  v.Name,
			"type":  v.Type,
			"label": v.Label,
		}

		if v.Query != "" {
			variable["query"] = v.Query
		}

		// Add common fields based on type
		if v.Type == "datasource" {
			variable["query"] = "prometheus"
			variable["current"] = map[string]any{
				"selected": false,
				"text":     "Prometheus",
				"value":    "Prometheus",
			}
		}

		list = append(list, variable)
	}

	return map[string]any{
		"list": list,
	}
}

func (d *Dashboard) generatePanels() []map[string]any {
	panels := []map[string]any{}
	panelID := 1
	yPos := 0

	for _, row := range d.Rows {
		// Add row header
		rowPanel := map[string]any{
			"collapsed": false,
			"datasource": map[string]any{
				"type": "datasource",
				"uid":  "grafana",
			},
			"gridPos": map[string]int{
				"h": 1,
				"w": 24,
				"x": 0,
				"y": yPos,
			},
			"id":     panelID,
			"panels": []any{},
			"title":  row.Title,
			"type":   "row",
		}
		panels = append(panels, rowPanel)
		panelID++
		yPos++

		// Add panels in this row
		xPos := 0
		for _, panel := range row.Panels {
			// Check if panel wraps to next line
			if xPos+panel.Width > 24 {
				xPos = 0
				yPos += panel.Height
			}

			p := panel.generate(panelID, xPos, yPos)
			panels = append(panels, p)
			panelID++

			xPos += panel.Width
		}

		// Move yPos to next row after last panel
		if len(row.Panels) > 0 {
			yPos += row.Panels[0].Height // Assume uniform height in row
		}
	}

	return panels
}
