package tracker

type Label string      // Prometheus label.
type MetricName string // Prometheus metric name.

// MetricsConfiguration is the parsed aggregation configuration.
type MetricsConfiguration map[MetricName][]Label
