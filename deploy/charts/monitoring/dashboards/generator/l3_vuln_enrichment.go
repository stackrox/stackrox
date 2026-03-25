package generator

// L3VulnEnrichment creates the Level 3 "Central: Vulnerability Enrichment" detail dashboard.
// This dashboard provides deep visibility into the vulnerability enrichment pipeline including
// scan semaphores, image/node scanning performance, deduplication, and registry client metrics.
func L3VulnEnrichment() Dashboard {
	return Dashboard{
		UID:   "central-vuln-enrichment",
		Title: "Central: Vulnerability Enrichment",
		Tags:  []string{"stackrox", "central", "level-3", "vulnerability-enrichment"},
		Links: []DashboardLink{
			{
				Title:     "← Central Internals",
				TargetUID: "central-internals",
				Type:      "link",
			},
		},
		Rows: []Row{
			scanSemaphoreRow(),
			imageScanningRow(),
			nodeScanningRow(),
			imageDeduplicationRow(),
			registryClientRow(),
		},
	}
}

func scanSemaphoreRow() Row {
	return Row{
		Title: "Scan Semaphore",
		Panels: []Panel{
			{
				Title:  "Scans In-Flight",
				Type:   "stat",
				Width:  8,
				Height: 8,
				Queries: []Query{
					{
						Expr:  `sum(rox_image_scan_semaphore_holding_size)`,
						RefID: "A",
					},
				},
			},
			{
				Title:  "Semaphore Utilization",
				Type:   "timeseries",
				Width:  8,
				Height: 8,
				Queries: []Query{
					{
						Expr:         `rox_image_scan_semaphore_holding_size`,
						LegendFormat: `holding`,
						RefID:        "A",
					},
					{
						Expr:         `rox_image_scan_semaphore_limit`,
						LegendFormat: `limit`,
						RefID:        "B",
					},
				},
			},
			{
				Title:  "Queue Waiting",
				Type:   "timeseries",
				Width:  8,
				Height: 8,
				Queries: []Query{
					{
						Expr:         `rox_image_scan_semaphore_queue_size`,
						LegendFormat: `{{subsystem}} - {{entity}}`,
						RefID:        "A",
					},
				},
			},
		},
	}
}

func imageScanningRow() Row {
	return Row{
		Title: "Image Scanning",
		Panels: []Panel{
			{
				Title:  "Scan Duration p50/p95/p99",
				Type:   "timeseries",
				Width:  12,
				Height: 8,
				Queries: []Query{
					{
						Expr:         `histogram_quantile(0.5, rate(rox_central_scan_duration_bucket[5m]))`,
						LegendFormat: `p50`,
						RefID:        "A",
					},
					{
						Expr:         `histogram_quantile(0.95, rate(rox_central_scan_duration_bucket[5m]))`,
						LegendFormat: `p95`,
						RefID:        "B",
					},
					{
						Expr:         `histogram_quantile(0.99, rate(rox_central_scan_duration_bucket[5m]))`,
						LegendFormat: `p99`,
						RefID:        "C",
					},
				},
			},
			{
				Title:  "Vuln Retrieval Duration",
				Type:   "timeseries",
				Width:  12,
				Height: 8,
				Queries: []Query{
					{
						Expr:         `histogram_quantile(0.95, rate(rox_central_image_vuln_retrieval_duration_bucket[5m]))`,
						LegendFormat: `p95`,
						RefID:        "A",
					},
				},
			},
			{
				Title:  "Metadata Cache Hit Rate",
				Type:   "timeseries",
				Width:  12,
				Height: 8,
				Unit:   "percentunit",
				Queries: []Query{
					{
						Expr:         `rate(rox_central_metadata_cache_hits[5m]) / (rate(rox_central_metadata_cache_hits[5m]) + rate(rox_central_metadata_cache_misses[5m]))`,
						LegendFormat: `hit rate`,
						RefID:        "A",
					},
				},
			},
			{
				Title:   "GAP: Enrichment Calls",
				Width:   12,
				Height:  4,
				GapNote: `**Metric Needed**: ` + "`central_vuln_enrichment_requests_total{type,result}`" + ` — No counter for total enrichment requests (inline vs background). Cannot calculate enrichment failure rate.`,
			},
		},
	}
}

func nodeScanningRow() Row {
	return Row{
		Title: "Node Scanning",
		Panels: []Panel{
			{
				Title:  "Node Scan Duration",
				Type:   "timeseries",
				Width:  12,
				Height: 8,
				Queries: []Query{
					{
						Expr:         `histogram_quantile(0.95, rate(rox_central_node_scan_duration_bucket[5m]))`,
						LegendFormat: `p95`,
						RefID:        "A",
					},
				},
			},
			{
				Title:   "GAP: Node Scan Count",
				Width:   12,
				Height:  4,
				GapNote: `**Metric Needed**: ` + "`central_vuln_enrichment_node_scans_total{result}`" + ` — No counter for total node scans.`,
			},
		},
	}
}

func imageDeduplicationRow() Row {
	return Row{
		Title: "Image Deduplication",
		Panels: []Panel{
			{
				Title:  "Image Upsert Deduper",
				Type:   "timeseries",
				Width:  12,
				Height: 8,
				Queries: []Query{
					{
						Expr:         `rate(rox_central_image_upsert_deduper[5m])`,
						LegendFormat: `{{status}}`,
						RefID:        "A",
					},
				},
			},
			{
				Title:  "Deployment Enhancement",
				Type:   "timeseries",
				Width:  12,
				Height: 8,
				Queries: []Query{
					{
						Expr:         `rox_central_deployment_enhancement_duration_ms`,
						LegendFormat: `duration`,
						RefID:        "A",
					},
				},
			},
		},
	}
}

func registryClientRow() Row {
	return Row{
		Title: "Registry Client",
		Panels: []Panel{
			{
				Title:  "Registry Requests",
				Type:   "timeseries",
				Width:  8,
				Height: 8,
				Queries: []Query{
					{
						Expr:         `rate(rox_central_registry_client_requests_total[5m])`,
						LegendFormat: `{{code}} - {{type}}`,
						RefID:        "A",
					},
				},
			},
			{
				Title:  "Registry Latency",
				Type:   "timeseries",
				Width:  8,
				Height: 8,
				Unit:   "s",
				Queries: []Query{
					{
						Expr:         `histogram_quantile(0.95, rate(rox_central_registry_client_request_duration_seconds_bucket[5m]))`,
						LegendFormat: `p95`,
						RefID:        "A",
					},
				},
			},
			{
				Title:  "Registry Timeouts",
				Type:   "timeseries",
				Width:  8,
				Height: 8,
				Queries: []Query{
					{
						Expr:         `rate(rox_central_registry_client_error_timeouts_total[5m])`,
						LegendFormat: `timeouts`,
						RefID:        "A",
					},
				},
			},
		},
	}
}
