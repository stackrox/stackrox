package generator

// L3Stubs creates all 8 remaining Level 3 stub dashboards for Central regions.
// Each stub has real panels where metrics exist and prominent gap annotations where they don't.
func L3Stubs() []Dashboard {
	return []Dashboard{
		l3DeploymentProcessing(),
		l3DetectionAlerts(),
		l3RiskCalculation(),
		l3BackgroundReprocessing(),
		l3PruningGC(),
		l3NetworkAnalysis(),
		l3ReportGeneration(),
		l3APIUI(),
	}
}

// l3DeploymentProcessing creates the "Central: Deployment Processing" dashboard.
func l3DeploymentProcessing() Dashboard {
	return Dashboard{
		UID:   "central-deployment-processing",
		Title: "Central: Deployment Processing",
		Tags:  []string{"stackrox", "central", "level-3", "deployment-processing"},
		Links: []DashboardLink{
			{
				Title:     "← Central Internals",
				TargetUID: "central-internals",
				Type:      "link",
			},
		},
		Rows: []Row{
			{
				Title: "Resource Processing",
				Panels: []Panel{
					{
						Title:  "Resources/sec",
						Type:   "timeseries",
						Width:  12,
						Height: 8,
						Queries: []Query{
							{
								Expr:         `rate(rox_central_resource_processed_count[5m])`,
								LegendFormat: `{{Resource}} - {{Operation}}`,
								RefID:        "A",
							},
						},
					},
					{
						Title:  "K8s Event Duration",
						Type:   "timeseries",
						Width:  12,
						Height: 8,
						Queries: []Query{
							{
								Expr:         `histogram_quantile(0.95, rate(rox_central_k8s_event_processing_duration_bucket[5m]))`,
								LegendFormat: `p95 {{Resource}} - {{Action}}`,
								RefID:        "A",
							},
						},
					},
				},
			},
			{
				Title: "Store Operations",
				Panels: []Panel{
					{
						Title:  "Postgres Op Duration",
						Type:   "timeseries",
						Width:  12,
						Height: 8,
						Queries: []Query{
							{
								Expr:         `histogram_quantile(0.95, rate(rox_central_postgres_op_duration_bucket{Type=~"deployments|pods|namespaces"}[5m]))`,
								LegendFormat: `p95`,
								RefID:        "A",
							},
						},
					},
					{
						Title:   "GAP: Per-Fragment Handler Metrics",
						Width:   12,
						Height:  4,
						GapNote: `**Metric Needed**: No per-fragment handler metrics. Cannot distinguish processing time for deployment vs pod vs namespace fragments.`,
					},
				},
			},
		},
	}
}

// l3DetectionAlerts creates the "Central: Detection & Alerts" dashboard.
func l3DetectionAlerts() Dashboard {
	return Dashboard{
		UID:   "central-detection-alerts",
		Title: "Central: Detection & Alerts",
		Tags:  []string{"stackrox", "central", "level-3", "detection-alerts"},
		Links: []DashboardLink{
			{
				Title:     "← Central Internals",
				TargetUID: "central-internals",
				Type:      "link",
			},
		},
		Rows: []Row{
			{
				Title: "Detection",
				Panels: []Panel{
					{
						Title:  "Process Filter",
						Type:   "timeseries",
						Width:  12,
						Height: 8,
						Queries: []Query{
							{
								Expr:         `rox_central_process_filter`,
								LegendFormat: `{{Type}}`,
								RefID:        "A",
							},
						},
					},
					{
						Title:   "GAP: Alert Generation Rate",
						Width:   12,
						Height:  4,
						GapNote: "**Metric Needed**: `central_detection_alerts_generated_total` — No alert generation rate metric. Cannot answer \"how many alerts are being generated?\"",
					},
				},
			},
			{
				Title: "Gaps",
				Panels: []Panel{
					{
						Title:   "GAP: Lifecycle Manager Metrics",
						Width:   24,
						Height:  4,
						GapNote: "**Metric Needed**: No lifecycle manager metrics. Need: `central_detection_lifecycle_duration_seconds`, `central_detection_baseline_evaluations_total`",
					},
				},
			},
		},
	}
}

// l3RiskCalculation creates the "Central: Risk Calculation" dashboard.
func l3RiskCalculation() Dashboard {
	return Dashboard{
		UID:   "central-risk-calculation",
		Title: "Central: Risk Calculation",
		Tags:  []string{"stackrox", "central", "level-3", "risk-calculation"},
		Links: []DashboardLink{
			{
				Title:     "← Central Internals",
				TargetUID: "central-internals",
				Type:      "link",
			},
		},
		Rows: []Row{
			{
				Title: "Risk Processing",
				Panels: []Panel{
					{
						Title:  "Risk Duration",
						Type:   "timeseries",
						Width:  12,
						Height: 8,
						Queries: []Query{
							{
								Expr:         `rox_central_risk_processing_duration`,
								LegendFormat: `{{Risk_Reprocessor}}`,
								RefID:        "A",
							},
						},
					},
					{
						Title:  "Reprocessor Duration",
						Type:   "timeseries",
						Width:  12,
						Height: 8,
						Queries: []Query{
							{
								Expr:         `rox_central_reprocessor_duration_seconds`,
								LegendFormat: `duration`,
								RefID:        "A",
							},
						},
					},
				},
			},
			{
				Title: "Gaps",
				Panels: []Panel{
					{
						Title:   "GAP: Items Processed",
						Width:   24,
						Height:  4,
						GapNote: "**Metric Needed**: `central_risk_items_processed_total` — No items-processed counter. Cannot answer \"how many deployments had risk recalculated?\"",
					},
				},
			},
		},
	}
}

// l3BackgroundReprocessing creates the "Central: Background Reprocessing" dashboard.
func l3BackgroundReprocessing() Dashboard {
	return Dashboard{
		UID:   "central-background-reprocessing",
		Title: "Central: Background Reprocessing",
		Tags:  []string{"stackrox", "central", "level-3", "background-reprocessing"},
		Links: []DashboardLink{
			{
				Title:     "← Central Internals",
				TargetUID: "central-internals",
				Type:      "link",
			},
		},
		Rows: []Row{
			{
				Title: "Reprocessor Loops",
				Panels: []Panel{
					{
						Title:  "Reprocessor Duration",
						Type:   "timeseries",
						Width:  12,
						Height: 8,
						Queries: []Query{
							{
								Expr:         `rox_central_reprocessor_duration_seconds`,
								LegendFormat: `duration`,
								RefID:        "A",
							},
						},
					},
					{
						Title:  "Sig Verification Duration",
						Type:   "timeseries",
						Width:  12,
						Height: 8,
						Queries: []Query{
							{
								Expr:         `rox_central_signature_verification_reprocessor_duration_seconds`,
								LegendFormat: `duration`,
								RefID:        "A",
							},
						},
					},
				},
			},
			{
				Title: "Gaps — Loop Instrumentation",
				Panels: []Panel{
					{
						Title:   "GAP: Background Loop Metrics",
						Width:   24,
						Height:  6,
						GapNote: "**Metric Needed**: 19+ background loops lack standard metrics. Need per-loop: `_running` (gauge), `_runs_total{result}` (counter), `_run_duration_seconds` (histogram), `_items_processed_total` (counter), `_last_run_timestamp_seconds` (gauge). Loops include: image-enrich, deployment-risk, active-components, pruning, CVE-suppress, CVE-fetch, indicator-flush, network-baseline-flush, hash-flush, conn-health, vuln-request, network-gatherer, and more.",
					},
				},
			},
		},
	}
}

// l3PruningGC creates the "Central: Pruning & GC" dashboard.
func l3PruningGC() Dashboard {
	return Dashboard{
		UID:   "central-pruning-gc",
		Title: "Central: Pruning & GC",
		Tags:  []string{"stackrox", "central", "level-3", "pruning-gc"},
		Links: []DashboardLink{
			{
				Title:     "← Central Internals",
				TargetUID: "central-internals",
				Type:      "link",
			},
		},
		Rows: []Row{
			{
				Title: "Pruning",
				Panels: []Panel{
					{
						Title:  "Prune Duration",
						Type:   "timeseries",
						Width:  8,
						Height: 8,
						Queries: []Query{
							{
								Expr:         `rox_central_prune_duration`,
								LegendFormat: `{{Type}}`,
								RefID:        "A",
							},
						},
					},
					{
						Title:  "Process Queue Length",
						Type:   "timeseries",
						Width:  8,
						Height: 8,
						Queries: []Query{
							{
								Expr:         `rox_central_process_queue_length`,
								LegendFormat: `queue length`,
								RefID:        "A",
							},
						},
					},
					{
						Title:  "Pruned Indicators",
						Type:   "timeseries",
						Width:  8,
						Height: 8,
						Queries: []Query{
							{
								Expr:         `rate(rox_central_pruned_process_indicators[5m])`,
								LegendFormat: `pruned/sec`,
								RefID:        "A",
							},
						},
					},
				},
			},
			{
				Title: "Additional Metrics",
				Panels: []Panel{
					{
						Title:  "Orphaned PLOPs",
						Type:   "timeseries",
						Width:  12,
						Height: 8,
						Queries: []Query{
							{
								Expr:         `rate(rox_central_orphaned_plop_total[5m])`,
								LegendFormat: `{{ClusterID}}`,
								RefID:        "A",
							},
						},
					},
					{
						Title:  "Cache Hits/Misses",
						Type:   "timeseries",
						Width:  12,
						Height: 8,
						Queries: []Query{
							{
								Expr:         `rate(rox_central_process_pruning_cache_hits[5m])`,
								LegendFormat: `hits`,
								RefID:        "A",
							},
							{
								Expr:         `rate(rox_central_process_pruning_cache_misses[5m])`,
								LegendFormat: `misses`,
								RefID:        "B",
							},
						},
					},
				},
			},
		},
	}
}

// l3NetworkAnalysis creates the "Central: Network Analysis" dashboard.
func l3NetworkAnalysis() Dashboard {
	return Dashboard{
		UID:   "central-network-analysis",
		Title: "Central: Network Analysis",
		Tags:  []string{"stackrox", "central", "level-3", "network-analysis"},
		Links: []DashboardLink{
			{
				Title:     "← Central Internals",
				TargetUID: "central-internals",
				Type:      "link",
			},
		},
		Rows: []Row{
			{
				Title: "Flows & Endpoints",
				Panels: []Panel{
					{
						Title:  "Flows Received",
						Type:   "timeseries",
						Width:  12,
						Height: 8,
						Queries: []Query{
							{
								Expr:         `rate(rox_central_total_network_flows_central_received_counter[5m])`,
								LegendFormat: `{{ClusterID}}`,
								RefID:        "A",
							},
						},
					},
					{
						Title:  "Endpoints Received",
						Type:   "timeseries",
						Width:  12,
						Height: 8,
						Queries: []Query{
							{
								Expr:         `rate(rox_central_total_network_endpoints_received_counter[5m])`,
								LegendFormat: `{{ClusterID}}`,
								RefID:        "A",
							},
						},
					},
				},
			},
			{
				Title: "Gaps",
				Panels: []Panel{
					{
						Title:   "GAP: Network Processing Pipeline",
						Width:   24,
						Height:  4,
						GapNote: "**Metric Needed**: Network baseline flush, external network gatherer, and flow processing pipeline have no Central-side metrics. Need: `central_network_baseline_flush_duration_seconds`, `central_network_flows_processed_total{action}`",
					},
				},
			},
		},
	}
}

// l3ReportGeneration creates the "Central: Report Generation" dashboard.
func l3ReportGeneration() Dashboard {
	return Dashboard{
		UID:   "central-report-generation",
		Title: "Central: Report Generation",
		Tags:  []string{"stackrox", "central", "level-3", "report-generation"},
		Links: []DashboardLink{
			{
				Title:     "← Central Internals",
				TargetUID: "central-internals",
				Type:      "link",
			},
		},
		Rows: []Row{
			{
				Title: "Compliance Operator Reports",
				Panels: []Panel{
					{
						Title:  "Scan Watchers",
						Type:   "timeseries",
						Width:  8,
						Height: 8,
						Queries: []Query{
							{
								Expr:         `rox_central_complianceoperator_scan_watchers_current`,
								LegendFormat: `watchers`,
								RefID:        "A",
							},
						},
					},
					{
						Title:  "Parallel Scans",
						Type:   "timeseries",
						Width:  8,
						Height: 8,
						Queries: []Query{
							{
								Expr:         `rox_central_complianceoperator_num_scans_running_in_parallel`,
								LegendFormat: `parallel scans`,
								RefID:        "A",
							},
						},
					},
					{
						Title:  "Watcher Active Time",
						Type:   "timeseries",
						Width:  8,
						Height: 8,
						Queries: []Query{
							{
								Expr:         `rox_central_complianceoperator_scan_watchers_active_time_minutes`,
								LegendFormat: `active time`,
								RefID:        "A",
							},
						},
					},
				},
			},
			{
				Title: "Watcher Outcomes",
				Panels: []Panel{
					{
						Title:  "Finish Types",
						Type:   "timeseries",
						Width:  24,
						Height: 8,
						Queries: []Query{
							{
								Expr:         `rate(rox_central_complianceoperator_scan_watchers_finish_type_total[5m])`,
								LegendFormat: `{{type}}`,
								RefID:        "A",
							},
						},
					},
				},
			},
			{
				Title: "Gaps — Vulnerability Report Pipeline",
				Panels: []Panel{
					{
						Title:   "GAP: Report Generation Pipeline",
						Width:   24,
						Height:  4,
						GapNote: "**Metric Needed**: No metrics for Central's vulnerability report generation pipeline (report scheduler, PDF/CSV generation, email delivery). Need: `central_report_generation_total{type,result}`, `central_report_generation_duration_seconds`, `central_report_delivery_total{method,result}`",
					},
				},
			},
		},
	}
}

// l3APIUI creates the "Central: API & UI" dashboard.
func l3APIUI() Dashboard {
	return Dashboard{
		UID:   "central-api-ui",
		Title: "Central: API & UI",
		Tags:  []string{"stackrox", "central", "level-3", "api-ui"},
		Links: []DashboardLink{
			{
				Title:     "← Central Internals",
				TargetUID: "central-internals",
				Type:      "link",
			},
		},
		Rows: []Row{
			{
				Title: "GraphQL",
				Panels: []Panel{
					{
						Title:  "Query Duration p95",
						Type:   "timeseries",
						Width:  12,
						Height: 8,
						Queries: []Query{
							{
								Expr:         `histogram_quantile(0.95, rate(rox_central_graphql_query_duration_bucket[5m]))`,
								LegendFormat: `{{Query}}`,
								RefID:        "A",
							},
						},
					},
					{
						Title:  "Resolver Duration p95",
						Type:   "timeseries",
						Width:  12,
						Height: 8,
						Queries: []Query{
							{
								Expr:         `histogram_quantile(0.95, rate(rox_central_graphql_op_duration_bucket[5m]))`,
								LegendFormat: `{{Resolver}}`,
								RefID:        "A",
							},
						},
					},
				},
			},
			{
				Title: "gRPC",
				Panels: []Panel{
					{
						Title:  "gRPC Errors",
						Type:   "timeseries",
						Width:  12,
						Height: 8,
						Queries: []Query{
							{
								Expr:         `rate(rox_central_grpc_error[5m])`,
								LegendFormat: `{{Code}}`,
								RefID:        "A",
							},
						},
					},
					{
						Title:  "Message Sizes",
						Type:   "timeseries",
						Width:  12,
						Height: 8,
						Queries: []Query{
							{
								Expr:         `rox_central_grpc_message_size_sent_bytes`,
								LegendFormat: `{{Type}}`,
								RefID:        "A",
							},
						},
					},
				},
			},
			{
				Title: "Gaps",
				Panels: []Panel{
					{
						Title:   "GAP: Per-Endpoint Metrics",
						Width:   12,
						Height:  4,
						GapNote: "**Metric Needed**: No per-API-endpoint latency or error rate metrics. Need: `central_api_request_duration_seconds{method,endpoint}`, `central_api_requests_total{method,endpoint,status}`. Cannot answer \"which API endpoint is slow?\"",
					},
					{
						Title:   "GAP: UI Page Load",
						Width:   12,
						Height:  4,
						GapNote: "**Metric Needed**: No UI page load metrics. Need frontend instrumentation or backend per-page-load latency tracking.",
					},
				},
			},
		},
	}
}
