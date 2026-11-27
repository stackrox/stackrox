package inmem

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	inmemRowsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ScannerSubsystem.String(),
		Name:      "matcher_inmem_rows",
		Help:      "Total number of vulnerability rows in the live in-memory matcher store.",
	})
	inmemFragmentsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ScannerSubsystem.String(),
		Name:      "matcher_inmem_fragments",
		Help:      "Total number of vulnerability fragments in the live in-memory matcher store.",
	})
	inmemOffHeapBytesGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ScannerSubsystem.String(),
		Name:      "matcher_inmem_offheap_bytes",
		Help:      "Total off-heap mmap file bytes backing the live in-memory matcher store, by kind.",
	}, []string{"kind"})
	inmemUpdateDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ScannerSubsystem.String(),
		Name:      "matcher_inmem_update_duration_seconds",
		Help:      "Duration of individual updater fragment builds in the in-memory matcher store.",
		Buckets:   []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10, 30, 60, 120},
	}, []string{"kind"})
)

func recordLiveMetrics(ls *liveSet) {
	st := liveStats(ls)
	inmemRowsGauge.Set(float64(st.Rows))
	inmemFragmentsGauge.Set(float64(st.Fragments))
	inmemOffHeapBytesGauge.WithLabelValues("cols").Set(float64(st.ColsOffHeap))
	inmemOffHeapBytesGauge.WithLabelValues("display").Set(float64(st.DisplayOffHeap))
	inmemOffHeapBytesGauge.WithLabelValues("enrich").Set(float64(st.EnrichOffHeap))
}
