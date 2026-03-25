package generator

// L3SensorIngestion creates the Level 3 "Central: Sensor Ingestion" detail dashboard.
// This dashboard provides deep visibility into the Sensor Ingestion pipeline with full metric
// breakdowns for connection status, deduplication, worker queues, and pipeline processing.
func L3SensorIngestion() Dashboard {
	return Dashboard{
		UID:   "central-sensor-ingestion",
		Title: "Central: Sensor Ingestion",
		Tags:  []string{"stackrox", "central", "level-3", "sensor-ingestion"},
		Links: []DashboardLink{
			{
				Title:     "← Central Internals",
				TargetUID: "central-internals",
				Type:      "link",
			},
		},
		Rows: []Row{
			connectionStatusRow(),
			deduperRow(),
			workerQueueRow(),
			pipelineProcessingRow(),
			messagesNotSentRow(),
		},
	}
}

func connectionStatusRow() Row {
	return Row{
		Title: "Connection Status",
		Panels: []Panel{
			{
				Title:  "Sensors Connected",
				Type:   "stat",
				Width:  8,
				Height: 8,
				Queries: []Query{
					{
						Expr:  `count(rox_central_sensor_connected{connection_state="connected"})`,
						RefID: "A",
					},
				},
			},
			{
				Title:  "Connection Events",
				Type:   "timeseries",
				Width:  16,
				Height: 8,
				Queries: []Query{
					{
						Expr:         `rate(rox_central_sensor_connected[5m])`,
						LegendFormat: `{{connection_state}}`,
						RefID:        "A",
					},
				},
			},
		},
	}
}

func deduperRow() Row {
	return Row{
		Title: "Deduper",
		Panels: []Panel{
			{
				Title:  "Deduper Throughput",
				Type:   "timeseries",
				Width:  12,
				Height: 8,
				Queries: []Query{
					{
						Expr:         `rate(rox_central_sensor_event_deduper[5m])`,
						LegendFormat: `{{status}} - {{type}}`,
						RefID:        "A",
					},
				},
			},
			{
				Title:  "Deduper Hit Rate",
				Type:   "timeseries",
				Width:  12,
				Height: 8,
				Queries: []Query{
					{
						Expr:         `rate(rox_central_sensor_event_deduper{status="deduplicated"}[5m]) / rate(rox_central_sensor_event_deduper[5m])`,
						LegendFormat: `dedup rate`,
						RefID:        "A",
					},
				},
			},
			{
				Title:  "Hash Store Size",
				Type:   "timeseries",
				Width:  12,
				Height: 8,
				Queries: []Query{
					{
						Expr:         `rox_central_deduping_hash_size`,
						LegendFormat: `{{cluster}}`,
						RefID:        "A",
					},
				},
			},
			{
				Title:  "Hash Operations",
				Type:   "timeseries",
				Width:  12,
				Height: 8,
				Queries: []Query{
					{
						Expr:         `rate(rox_central_deduping_hash_count[5m])`,
						LegendFormat: `{{ResourceType}} - {{Operation}}`,
						RefID:        "A",
					},
				},
			},
		},
	}
}

func workerQueueRow() Row {
	return Row{
		Title: "Worker Queue",
		Panels: []Panel{
			{
				Title:  "Events Processed",
				Type:   "timeseries",
				Width:  12,
				Height: 8,
				Queries: []Query{
					{
						Expr:         `rate(rox_central_sensor_event_queue[5m])`,
						LegendFormat: `{{Operation}} - {{Type}}`,
						RefID:        "A",
					},
				},
			},
			{
				Title:  "Processing Duration",
				Type:   "timeseries",
				Width:  12,
				Height: 8,
				Queries: []Query{
					{
						Expr:         `histogram_quantile(0.95, rate(rox_central_sensor_event_duration_bucket[5m]))`,
						LegendFormat: `p95 - {{Type}}`,
						RefID:        "A",
					},
				},
			},
			{
				Title:   "GAP: Queue Depth",
				Width:   12,
				Height:  4,
				GapNote: `**Metric Needed**: ` + "`central_sensor_ingestion_queue_depth`" + ` — No gauge exists for worker queue shard depth. Cannot answer "is the queue backing up?"`,
			},
			{
				Title:   "GAP: In-Flight",
				Width:   12,
				Height:  4,
				GapNote: `**Metric Needed**: ` + "`central_sensor_ingestion_in_flight`" + ` — No gauge for items currently being processed per shard.`,
			},
		},
	}
}

func pipelineProcessingRow() Row {
	return Row{
		Title: "Pipeline Processing",
		Panels: []Panel{
			{
				Title:  "Resources Processed",
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
				Title:  "Pipeline Panics",
				Type:   "timeseries",
				Width:  12,
				Height: 8,
				Queries: []Query{
					{
						Expr:         `rate(rox_central_pipeline_panics[5m])`,
						LegendFormat: `{{resource}}`,
						RefID:        "A",
					},
				},
			},
			{
				Title:  "K8s Event Processing",
				Type:   "timeseries",
				Width:  12,
				Height: 8,
				Queries: []Query{
					{
						Expr:         `histogram_quantile(0.95, rate(rox_central_k8s_event_processing_duration_bucket[5m]))`,
						LegendFormat: `p95 - {{Resource}}`,
						RefID:        "A",
					},
				},
			},
			{
				Title:   "GAP: Per-Fragment Metrics",
				Width:   12,
				Height:  4,
				GapNote: `**Metric Needed**: Per-fragment processing counts and durations. 25 pipeline fragments exist but none have individual metrics.`,
			},
		},
	}
}

func messagesNotSentRow() Row {
	return Row{
		Title: "Messages Not Sent",
		Panels: []Panel{
			{
				Title:  "Failed Sends to Sensor",
				Type:   "timeseries",
				Width:  24,
				Height: 8,
				Queries: []Query{
					{
						Expr:         `rate(rox_central_msg_to_sensor_not_sent_count[5m])`,
						LegendFormat: `{{type}} - {{reason}}`,
						RefID:        "A",
					},
				},
			},
		},
	}
}
