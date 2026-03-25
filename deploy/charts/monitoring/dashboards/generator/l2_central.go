package generator

// L2CentralInternals creates the Level 2 "Central Internals" dashboard.
// This dashboard provides a grid view of all 10 logical regions within Central,
// showing headline metrics for each region with links to detailed Level 3 dashboards.
func L2CentralInternals() Dashboard {
	return Dashboard{
		UID:   "central-internals",
		Title: "Central Internals",
		Tags:  []string{"stackrox", "central", "level-2"},
		Links: []DashboardLink{
			{
				Title:     "← StackRox Overview",
				TargetUID: "stackrox-overview",
				Type:      "link",
			},
		},
		Rows: []Row{
			sensorIngestionRow(),
			deploymentProcessingRow(),
			vulnerabilityEnrichmentRow(),
			detectionAlertsRow(),
			riskCalculationRow(),
			backgroundReprocessingRow(),
			pruningGCRow(),
			networkAnalysisRow(),
			reportGenerationRow(),
			apiUIRow(),
		},
	}
}

func sensorIngestionRow() Row {
	return Row{
		Title: "Sensor Ingestion",
		Panels: []Panel{
			{
				Title:  "events/sec",
				Type:   "timeseries",
				Width:  7,
				Height: 8,
				Queries: []Query{
					{
						Expr:  `rate(rox_central_sensor_event_queue[5m])`,
						RefID: "A",
					},
				},
			},
			{
				Title:  "deduper",
				Type:   "timeseries",
				Width:  7,
				Height: 8,
				Queries: []Query{
					{
						Expr:  `rate(rox_central_sensor_event_deduper[5m])`,
						RefID: "A",
					},
				},
			},
			{
				Title:  "processing latency p95",
				Type:   "timeseries",
				Width:  7,
				Height: 8,
				Unit:   "s",
				Queries: []Query{
					{
						Expr:  `histogram_quantile(0.95, rate(rox_central_sensor_event_duration_bucket[5m]))`,
						RefID: "A",
					},
				},
			},
			{
				Title:   "",
				Width:   3,
				Height:  8,
				GapNote: "### [→ Details](/d/central-sensor-ingestion)\n\nDrill into Sensor Ingestion metrics",
			},
		},
	}
}

func deploymentProcessingRow() Row {
	return Row{
		Title: "Deployment Processing",
		Panels: []Panel{
			{
				Title:  "resources/sec",
				Type:   "timeseries",
				Width:  8,
				Height: 8,
				Queries: []Query{
					{
						Expr:  `rate(rox_central_resource_processed_count[5m])`,
						RefID: "A",
					},
				},
			},
			{
				Title:  "K8s event latency",
				Type:   "timeseries",
				Width:  8,
				Height: 8,
				Unit:   "s",
				Queries: []Query{
					{
						Expr:  `histogram_quantile(0.95, rate(rox_central_k8s_event_processing_duration_bucket[5m]))`,
						RefID: "A",
					},
				},
			},
			{
				Title:   "",
				Width:   8,
				Height:  8,
				GapNote: "### [→ Details](/d/central-deployment-processing)\n\nDrill into Deployment Processing metrics",
			},
		},
	}
}

func vulnerabilityEnrichmentRow() Row {
	return Row{
		Title: "Vulnerability Enrichment",
		Panels: []Panel{
			{
				Title:  "scans in-flight",
				Type:   "stat",
				Width:  6,
				Height: 8,
				Queries: []Query{
					{
						Expr:  `rox_image_scan_semaphore_holding_size`,
						RefID: "A",
					},
				},
			},
			{
				Title:  "scan duration p95",
				Type:   "timeseries",
				Width:  7,
				Height: 8,
				Unit:   "s",
				Queries: []Query{
					{
						Expr:  `histogram_quantile(0.95, rate(rox_central_scan_duration_bucket[5m]))`,
						RefID: "A",
					},
				},
			},
			{
				Title:  "queue waiting",
				Type:   "timeseries",
				Width:  7,
				Height: 8,
				Queries: []Query{
					{
						Expr:  `rox_image_scan_semaphore_queue_size`,
						RefID: "A",
					},
				},
			},
			{
				Title:   "",
				Width:   4,
				Height:  8,
				GapNote: "### [→ Details](/d/central-vuln-enrichment)\n\nDrill into Vulnerability Enrichment metrics",
			},
		},
	}
}

func detectionAlertsRow() Row {
	return Row{
		Title: "Detection & Alerts",
		Panels: []Panel{
			{
				Title:  "process filter",
				Type:   "timeseries",
				Width:  10,
				Height: 8,
				Queries: []Query{
					{
						Expr:  `rox_central_process_filter`,
						RefID: "A",
					},
				},
			},
			{
				Title:   "Alert Generation Rate",
				Width:   10,
				Height:  8,
				GapNote: "⚠️ No alert generation rate metric available",
			},
			{
				Title:   "",
				Width:   4,
				Height:  8,
				GapNote: "### [→ Details](/d/central-detection-alerts)\n\nDrill into Detection & Alerts metrics",
			},
		},
	}
}

func riskCalculationRow() Row {
	return Row{
		Title: "Risk Calculation",
		Panels: []Panel{
			{
				Title:  "risk duration",
				Type:   "timeseries",
				Width:  10,
				Height: 8,
				Unit:   "s",
				Queries: []Query{
					{
						Expr:  `rox_central_risk_processing_duration`,
						RefID: "A",
					},
				},
			},
			{
				Title:  "reprocessor",
				Type:   "timeseries",
				Width:  10,
				Height: 8,
				Unit:   "s",
				Queries: []Query{
					{
						Expr:  `rox_central_reprocessor_duration_seconds`,
						RefID: "A",
					},
				},
			},
			{
				Title:   "",
				Width:   4,
				Height:  8,
				GapNote: "### [→ Details](/d/central-risk-calculation)\n\nDrill into Risk Calculation metrics",
			},
		},
	}
}

func backgroundReprocessingRow() Row {
	return Row{
		Title: "Background Reprocessing",
		Panels: []Panel{
			{
				Title:  "reprocessor duration",
				Type:   "timeseries",
				Width:  7,
				Height: 8,
				Unit:   "s",
				Queries: []Query{
					{
						Expr:  `rox_central_reprocessor_duration_seconds`,
						RefID: "A",
					},
				},
			},
			{
				Title:  "sig verification",
				Type:   "timeseries",
				Width:  7,
				Height: 8,
				Unit:   "s",
				Queries: []Query{
					{
						Expr:  `rox_central_signature_verification_reprocessor_duration_seconds`,
						RefID: "A",
					},
				},
			},
			{
				Title:   "Running/Items Processed",
				Width:   7,
				Height:  8,
				GapNote: "⚠️ No running/items-processed metrics available",
			},
			{
				Title:   "",
				Width:   3,
				Height:  8,
				GapNote: "### [→ Details](/d/central-background-reprocessing)\n\nDrill into Background Reprocessing metrics",
			},
		},
	}
}

func pruningGCRow() Row {
	return Row{
		Title: "Pruning & GC",
		Panels: []Panel{
			{
				Title:  "prune duration",
				Type:   "timeseries",
				Width:  7,
				Height: 8,
				Unit:   "s",
				Queries: []Query{
					{
						Expr:  `rox_central_prune_duration`,
						RefID: "A",
					},
				},
			},
			{
				Title:  "process queue",
				Type:   "timeseries",
				Width:  7,
				Height: 8,
				Queries: []Query{
					{
						Expr:  `rox_central_process_queue_length`,
						RefID: "A",
					},
				},
			},
			{
				Title:  "pruned indicators",
				Type:   "timeseries",
				Width:  7,
				Height: 8,
				Queries: []Query{
					{
						Expr:  `rate(rox_central_pruned_process_indicators[5m])`,
						RefID: "A",
					},
				},
			},
			{
				Title:   "",
				Width:   3,
				Height:  8,
				GapNote: "### [→ Details](/d/central-pruning-gc)\n\nDrill into Pruning & GC metrics",
			},
		},
	}
}

func networkAnalysisRow() Row {
	return Row{
		Title: "Network Analysis",
		Panels: []Panel{
			{
				Title:  "flows received",
				Type:   "timeseries",
				Width:  10,
				Height: 8,
				Queries: []Query{
					{
						Expr:  `rate(rox_central_total_network_flows_central_received_counter[5m])`,
						RefID: "A",
					},
				},
			},
			{
				Title:  "endpoints received",
				Type:   "timeseries",
				Width:  10,
				Height: 8,
				Queries: []Query{
					{
						Expr:  `rate(rox_central_total_network_endpoints_received_counter[5m])`,
						RefID: "A",
					},
				},
			},
			{
				Title:   "",
				Width:   4,
				Height:  8,
				GapNote: "### [→ Details](/d/central-network-analysis)\n\nDrill into Network Analysis metrics",
			},
		},
	}
}

func reportGenerationRow() Row {
	return Row{
		Title: "Report Generation",
		Panels: []Panel{
			{
				Title:   "Central-side Reports",
				Width:   10,
				Height:  8,
				GapNote: "⚠️ No Central-side report generation metrics exist",
			},
			{
				Title:  "compliance watchers",
				Type:   "timeseries",
				Width:  10,
				Height: 8,
				Queries: []Query{
					{
						Expr:  `rox_central_complianceoperator_scan_watchers_current`,
						RefID: "A",
					},
				},
			},
			{
				Title:   "",
				Width:   4,
				Height:  8,
				GapNote: "### [→ Details](/d/central-report-generation)\n\nDrill into Report Generation metrics",
			},
		},
	}
}

func apiUIRow() Row {
	return Row{
		Title: "API & UI",
		Panels: []Panel{
			{
				Title:  "GraphQL p95",
				Type:   "timeseries",
				Width:  10,
				Height: 8,
				Unit:   "s",
				Queries: []Query{
					{
						Expr:  `histogram_quantile(0.95, rate(rox_central_graphql_query_duration_bucket[5m]))`,
						RefID: "A",
					},
				},
			},
			{
				Title:  "gRPC errors",
				Type:   "timeseries",
				Width:  10,
				Height: 8,
				Queries: []Query{
					{
						Expr:  `rate(rox_central_grpc_error[5m])`,
						RefID: "A",
					},
				},
			},
			{
				Title:   "",
				Width:   4,
				Height:  8,
				GapNote: "### [→ Details](/d/central-api-ui)\n\nDrill into API & UI metrics",
			},
		},
	}
}
