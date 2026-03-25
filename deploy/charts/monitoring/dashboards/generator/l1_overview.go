package generator

// L1Overview creates the Level 1 "StackRox Overview" dashboard.
// This is the top-level service map dashboard that provides a high-level view
// of the entire StackRox deployment including Central health, connected sensors,
// and database status.
func L1Overview() Dashboard {
	return Dashboard{
		UID:   "stackrox-overview",
		Title: "StackRox Overview",
		Tags:  []string{"stackrox", "overview", "level-1"},
		Links: []DashboardLink{
			{
				Title:     "Central Internals",
				TargetUID: "central-internals",
				Type:      "link",
			},
		},
		Rows: []Row{
			serviceHealthRow(),
			connectedSensorsRow(),
			databaseRow(),
		},
	}
}

func serviceHealthRow() Row {
	return Row{
		Title: "Service Health",
		Panels: []Panel{
			{
				Title:  "Central Up",
				Type:   "stat",
				Width:  4,
				Height: 8,
				Queries: []Query{
					{
						Expr:  `up{job="central"}`,
						RefID: "A",
					},
				},
			},
			{
				Title:  "Central CPU",
				Type:   "timeseries",
				Width:  5,
				Height: 8,
				Queries: []Query{
					{
						Expr:  `rate(process_cpu_seconds_total{job="central"}[5m])`,
						RefID: "A",
					},
				},
			},
			{
				Title:  "Central Memory",
				Type:   "timeseries",
				Width:  5,
				Height: 8,
				Queries: []Query{
					{
						Expr:  `process_resident_memory_bytes{job="central"}`,
						RefID: "A",
					},
				},
			},
			{
				Title:  "Central Goroutines",
				Type:   "stat",
				Width:  5,
				Height: 8,
				Queries: []Query{
					{
						Expr:  `go_goroutines{job="central"}`,
						RefID: "A",
					},
				},
			},
			{
				Title:  "Central Version",
				Type:   "stat",
				Width:  5,
				Height: 8,
				Queries: []Query{
					{
						Expr:         `rox_central_info`,
						LegendFormat: `{{central_version}}`,
						RefID:        "A",
					},
				},
			},
		},
	}
}

func connectedSensorsRow() Row {
	return Row{
		Title: "Connected Sensors",
		Panels: []Panel{
			{
				Title:  "Sensors Connected",
				Type:   "stat",
				Width:  6,
				Height: 8,
				Queries: []Query{
					{
						Expr:  `count by (connection_state) (rox_central_sensor_connected)`,
						RefID: "A",
					},
				},
			},
			{
				Title:  "Secured Clusters",
				Type:   "stat",
				Width:  6,
				Height: 8,
				Queries: []Query{
					{
						Expr:  `rox_central_secured_clusters`,
						RefID: "A",
					},
				},
			},
			{
				Title:  "Secured Nodes",
				Type:   "stat",
				Width:  6,
				Height: 8,
				Queries: []Query{
					{
						Expr:  `rox_central_secured_nodes`,
						RefID: "A",
					},
				},
			},
			{
				Title:  "Secured vCPUs",
				Type:   "stat",
				Width:  6,
				Height: 8,
				Queries: []Query{
					{
						Expr:  `rox_central_secured_vcpus`,
						RefID: "A",
					},
				},
			},
		},
	}
}

func databaseRow() Row {
	return Row{
		Title: "Database",
		Panels: []Panel{
			{
				Title:  "Postgres Connected",
				Type:   "stat",
				Width:  6,
				Height: 8,
				Queries: []Query{
					{
						Expr:  `rox_central_postgres_connected`,
						RefID: "A",
					},
				},
			},
			{
				Title:  "DB Size",
				Type:   "stat",
				Width:  6,
				Height: 8,
				Unit:   "bytes",
				Queries: []Query{
					{
						Expr:  `rox_central_postgres_total_size_bytes`,
						RefID: "A",
					},
				},
			},
			{
				Title:  "Active Connections",
				Type:   "stat",
				Width:  6,
				Height: 8,
				Queries: []Query{
					{
						Expr:  `rox_central_postgres_total_connections{state="active"}`,
						RefID: "A",
					},
				},
			},
			{
				Title:  "Available Space",
				Type:   "stat",
				Width:  6,
				Height: 8,
				Unit:   "bytes",
				Queries: []Query{
					{
						Expr:  `rox_central_postgres_available_size_bytes`,
						RefID: "A",
					},
				},
			},
		},
	}
}
