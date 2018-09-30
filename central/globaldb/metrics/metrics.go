package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

const bucketKey = "Bucket"

func init() {
	prometheus.MustRegister(
		FreePageN,
		PendingPageN,
		FreeAlloc,
		FreelistInuse,
		TxN,
		OpenTxN,
		TxStatsPageCount,
		TxStatsPageAlloc,
		TxStatsCursorCount,
		TxStatsNodeCount,
		TxStatsNodeDeref,
		TxStatsRebalance,
		TxStatsRebalanceSeconds,
		TxStatsSplit,
		TxStatsSpill,
		TxStatsSpillSeconds,
		TxStatsWrite,
		TxStatsWriteTime,
		BranchPageN,
		BranchOverflowN,
		LeafPageN,
		LeafOverflowN,
		KeyN,
		Depth,
		BranchAlloc,
		BranchInuse,
		LeafAlloc,
		LeafInuse,
		BucketN,
		InlineBucketN,
		InlineBucketInuse,
	)
}

func newGauge(name, help string) prometheus.Gauge {
	return prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.Namespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      name,
		Help:      help,
	})
}

func newBucketGauge(name, help string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.Namespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      name,
		Help:      help,
	}, []string{bucketKey})
}

// These variables are all of the stats for BoltDB
var (
	// Freelist stats
	FreePageN     = newGauge("free_page_n", "total number of free pages on the freelist")
	PendingPageN  = newGauge("pending_page_n", "total number of pending pages on the freelist")
	FreeAlloc     = newGauge("free_alloc", "total bytes allocated in free pages")
	FreelistInuse = newGauge("freelist_inuse", "total bytes used by the freelist")

	// TxN stats
	TxN     = newGauge("tx_n", "total number of started read transactions")
	OpenTxN = newGauge("open_txn", "total bytes used by the freelist")

	// Page statistics
	TxStatsPageCount = newGauge("tx_stats_page_count", "number of page allocations")
	TxStatsPageAlloc = newGauge("tx_stats_page_alloc", "total bytes allocated")

	// Cursor statistics.
	TxStatsCursorCount = newGauge("tx_stats_cursor_count", "number of cursors created")

	// Node statistics
	TxStatsNodeCount = newGauge("tx_stats_node_count", "number of node allocations")
	TxStatsNodeDeref = newGauge("tx_stats_node_deref", "number of node dereferences")

	// Rebalance statistics.
	TxStatsRebalance        = newGauge("tx_stats_rebalance", "number of node rebalances")
	TxStatsRebalanceSeconds = newGauge("tx_stats_rebalance_seconds", "total time spent rebalancing")

	// Split/Spill statistics.
	TxStatsSplit        = newGauge("tx_stats_split", "number of nodes split")
	TxStatsSpill        = newGauge("tx_stats_spill", "number of nodes spilled")
	TxStatsSpillSeconds = newGauge("tx_stats_spill_seconds", "total time spent spilling")

	// Write statistics.
	TxStatsWrite     = newGauge("tx_stats_write", "number of writes performed")
	TxStatsWriteTime = newGauge("tx_stats_write_seconds", "total time spent writing to disk")

	////  Bucket Stats

	// Page count statistics.
	BranchPageN     = newBucketGauge("branch_page_n", "number of logical branch pages")
	BranchOverflowN = newBucketGauge("branch_overflow_n", "number of physical branch overflow pages")
	LeafPageN       = newBucketGauge("leaf_page_n", "number of logical leaf pages")
	LeafOverflowN   = newBucketGauge("leaf_overflow_n", "number of physical leaf overflow pages")

	// Tree statistics.
	KeyN  = newBucketGauge("key_n", "number of keys/value pairs")
	Depth = newBucketGauge("depth", "number of levels in B+tree")

	// Page size utilization.
	BranchAlloc = newBucketGauge("branch_alloc", "bytes allocated for physical branch pages")
	BranchInuse = newBucketGauge("branch_inuse", "bytes actually used for branch data")
	LeafAlloc   = newBucketGauge("leaf_alloc", "bytes allocated for physical leaf pages")
	LeafInuse   = newBucketGauge("leaf_inuse", "bytes actually used for leaf data")

	// Bucket statistics
	BucketN           = newBucketGauge("bucket_n", "total number of buckets including the top bucket")
	InlineBucketN     = newBucketGauge("inline_bucket_n", "total number on inlined buckets")
	InlineBucketInuse = newBucketGauge("inline_bucket_inuse", "bytes used for inlined buckets (also accounted for in LeafInuse)")
)

// SetGaugeInt sets a value for a gauge from an int
func SetGaugeInt(gauge prometheus.Gauge, value int) {
	gauge.Set(float64(value))
}

// SetGaugeDuration sets a value for the gauge in seconds from an time duration
func SetGaugeDuration(gauge prometheus.Gauge, value time.Duration) {
	gauge.Set(value.Seconds())
}

// SetGaugeBucketInt sets a value for a gauge from an int
func SetGaugeBucketInt(gauge *prometheus.GaugeVec, value int, name string) {
	gauge.With(prometheus.Labels{bucketKey: name}).Set(float64(value))
}
