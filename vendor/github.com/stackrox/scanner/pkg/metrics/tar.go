package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	fileCountPerLayer = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "tar_file_count_per_layer",
		Help:    "Number of files in an image layer",
		Buckets: []float64{50, 100, 500, 1000},
	})

	matchedFileCountPerLayer = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "tar_matched_file_count_per_layer",
		Help:    "Number of matched files in an image layer",
		Buckets: []float64{50, 100, 500, 1000},
	})

	extractedContentBytesPerLayer = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "tar_extracted_contents_bytes_per_layer",
		Help:    "Number of file contents (in bytes) extracted from an image layer",
		Buckets: []float64{64, 512, 1024, 4096},
	})
)

func init() {
	prometheus.MustRegister(
		fileCountPerLayer,
		matchedFileCountPerLayer,
		extractedContentBytesPerLayer,
	)
}

// ObserveFileCount observes the number of files in an image layer.
func ObserveFileCount(numFiles int) {
	fileCountPerLayer.Observe(float64(numFiles))
}

// ObserveMatchedFileCount observes the number of matched files in an image layer.
func ObserveMatchedFileCount(numMatchedFiles int) {
	matchedFileCountPerLayer.Observe(float64(numMatchedFiles))
}

// ObserveExtractedContentBytes observes the number of bytes extracted from an image layer.
func ObserveExtractedContentBytes(numBytes int) {
	extractedContentBytesPerLayer.Observe(float64(numBytes))
}
