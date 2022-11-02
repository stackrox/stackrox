package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	fileExtractionCountMetric = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "file_extraction_count",
		Help:    "Number of files in a node filesystem scan",
		Buckets: []float64{50, 100, 500, 1000},
	})

	fileExtractionMatchCountMetric = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "file_extraction_match_count",
		Help:    "Number of matched files in an node filesystem scan",
		Buckets: []float64{50, 100, 500, 1000},
	})

	fileExtractionInaccessibleCountMetric = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "file_extraction_inaccessible_count",
		Help:    "Number of matched files in an node filesystem scan that were not accessible for reading",
		Buckets: []float64{50, 100, 500, 1000},
	})
)

func init() {
	prometheus.MustRegister(
		fileExtractionCountMetric,
		fileExtractionMatchCountMetric,
		fileExtractionInaccessibleCountMetric,
	)
}

// FileExtractionMetrics tracks and emit node extraction metrics.
type FileExtractionMetrics struct {
	fileCount, matchCount, inaccessibleCount float64
}

// FileCount increments the file count.
func (m *FileExtractionMetrics) FileCount() {
	if m != nil {
		m.fileCount++
	}
}

// MatchCount increments the file match count.
func (m *FileExtractionMetrics) MatchCount() {
	if m != nil {
		m.matchCount++
	}
}

// InaccessibleCount increments the file error count that were ignored and treated as
// non-existent files.
func (m *FileExtractionMetrics) InaccessibleCount() {
	if m != nil {
		m.inaccessibleCount++
	}
}

// Emit emits the metrics and reset counters
func (m *FileExtractionMetrics) Emit() {
	if m != nil {
		fileExtractionCountMetric.Observe(m.matchCount)
		fileExtractionMatchCountMetric.Observe(m.fileCount)
		fileExtractionInaccessibleCountMetric.Observe(m.inaccessibleCount)
		*m = FileExtractionMetrics{}
	}
}
