package generator

// Panel types
type Panel struct {
	Title       string
	Description string
	Width       int // out of 24
	Height      int // grid units, typically 8
	Type        string // "timeseries", "stat", "gauge", "text", "table"
	Queries     []Query
	Unit        string // "short", "s", "bytes", "percentunit", "ops", etc.
	Thresholds  []Threshold
	GapNote     string // non-empty = this is a gap annotation panel
}

// Query represents a Prometheus query
type Query struct {
	Expr         string
	LegendFormat string
	RefID        string
}

// Threshold represents a threshold configuration
type Threshold struct {
	Value float64
	Color string // "green", "yellow", "red"
}

const datasourceUID = "PBFA97CFB590B2093"

// generate creates a Grafana panel JSON structure
func (p *Panel) generate(id, x, y int) map[string]any {
	// If this is a gap annotation, render as text panel
	if p.GapNote != "" {
		return p.generateGapPanel(id, x, y)
	}

	panel := map[string]any{
		"datasource": map[string]any{
			"type": "prometheus",
			"uid":  datasourceUID,
		},
		"fieldConfig": p.generateFieldConfig(),
		"gridPos": map[string]int{
			"h": p.Height,
			"w": p.Width,
			"x": x,
			"y": y,
		},
		"id":    id,
		"title": p.Title,
		"type":  p.Type,
	}

	// Add description if present
	if p.Description != "" {
		panel["description"] = p.Description
	}

	// Add targets (queries)
	if len(p.Queries) > 0 {
		panel["targets"] = p.generateTargets()
	}

	// Add type-specific options
	switch p.Type {
	case "timeseries":
		panel["options"] = p.generateTimeseriesOptions()
	case "stat":
		panel["options"] = p.generateStatOptions()
	case "gauge":
		panel["options"] = p.generateGaugeOptions()
	case "table":
		panel["options"] = p.generateTableOptions()
	}

	return panel
}

func (p *Panel) generateGapPanel(id, x, y int) map[string]any {
	content := "⚠️ " + p.GapNote

	return map[string]any{
		"datasource": map[string]any{
			"type": "datasource",
			"uid":  "grafana",
		},
		"gridPos": map[string]int{
			"h": p.Height,
			"w": p.Width,
			"x": x,
			"y": y,
		},
		"id":    id,
		"title": p.Title,
		"type":  "text",
		"options": map[string]any{
			"mode":    "markdown",
			"content": content,
		},
	}
}

func (p *Panel) generateFieldConfig() map[string]any {
	defaults := map[string]any{
		"color": map[string]any{
			"mode": "palette-classic",
		},
		"custom": map[string]any{
			"axisCenteredZero": false,
			"axisColorMode":    "text",
			"axisLabel":        "",
			"axisPlacement":    "auto",
			"barAlignment":     0,
			"drawStyle":        "line",
			"fillOpacity":      0,
			"gradientMode":     "none",
			"hideFrom": map[string]any{
				"tooltip": false,
				"viz":     false,
				"legend":  false,
			},
			"lineInterpolation": "linear",
			"lineWidth":         1,
			"pointSize":         5,
			"scaleDistribution": map[string]any{
				"type": "linear",
			},
			"showPoints":  "auto",
			"spanNulls":   false,
			"stacking": map[string]any{
				"group": "A",
				"mode":  "none",
			},
			"thresholdsStyle": map[string]any{
				"mode": "off",
			},
		},
		"mappings": []any{},
	}

	// Add unit if specified
	if p.Unit != "" {
		defaults["unit"] = p.Unit
	}

	// Add thresholds if specified
	if len(p.Thresholds) > 0 {
		defaults["thresholds"] = p.generateThresholds()
	} else {
		// Default thresholds
		defaults["thresholds"] = map[string]any{
			"mode": "absolute",
			"steps": []map[string]any{
				{"color": "green", "value": nil},
			},
		}
	}

	return map[string]any{
		"defaults":  defaults,
		"overrides": []any{},
	}
}

func (p *Panel) generateThresholds() map[string]any {
	steps := []map[string]any{
		{"color": "green", "value": nil},
	}

	for _, th := range p.Thresholds {
		steps = append(steps, map[string]any{
			"color": th.Color,
			"value": th.Value,
		})
	}

	return map[string]any{
		"mode":  "absolute",
		"steps": steps,
	}
}

func (p *Panel) generateTargets() []map[string]any {
	targets := make([]map[string]any, 0, len(p.Queries))

	for _, q := range p.Queries {
		target := map[string]any{
			"datasource": map[string]any{
				"type": "prometheus",
				"uid":  datasourceUID,
			},
			"editorMode":   "code",
			"expr":         q.Expr,
			"instant":      false,
			"legendFormat": q.LegendFormat,
			"range":        true,
			"refId":        q.RefID,
		}
		targets = append(targets, target)
	}

	return targets
}

func (p *Panel) generateTimeseriesOptions() map[string]any {
	return map[string]any{
		"legend": map[string]any{
			"calcs":       []string{},
			"displayMode": "list",
			"placement":   "bottom",
			"showLegend":  true,
		},
		"tooltip": map[string]any{
			"mode": "single",
			"sort": "none",
		},
	}
}

func (p *Panel) generateStatOptions() map[string]any {
	return map[string]any{
		"colorMode":   "value",
		"graphMode":   "none",
		"justifyMode": "auto",
		"orientation": "auto",
		"reduceOptions": map[string]any{
			"values": false,
			"calcs":  []string{"lastNotNull"},
		},
		"textMode": "auto",
	}
}

func (p *Panel) generateGaugeOptions() map[string]any {
	return map[string]any{
		"orientation": "auto",
		"reduceOptions": map[string]any{
			"values": false,
			"calcs":  []string{"lastNotNull"},
		},
		"showThresholdLabels":  false,
		"showThresholdMarkers": true,
	}
}

func (p *Panel) generateTableOptions() map[string]any {
	return map[string]any{
		"showHeader": true,
		"footer": map[string]any{
			"show":        false,
			"reducer":     []string{"sum"},
			"countRows":   false,
			"enablePagination": false,
		},
	}
}
