package globaldb

import (
	"os"
	"strings"
	"time"

	"github.com/stackrox/stackrox/central/globaldb/metrics"
	bolt "go.etcd.io/bbolt"
)

const gatherFrequency = 5 * time.Minute

func gatherBucketStats(name string, stats bolt.BucketStats) {
	// Ignore Bolt extra mapping buckets
	if strings.HasSuffix(name, "-unique") || strings.HasSuffix(name, "-mapper") {
		return
	}
	metrics.SetGaugeBucketInt(metrics.BranchPageN, stats.BranchPageN, name)
	metrics.SetGaugeBucketInt(metrics.BranchOverflowN, stats.BranchOverflowN, name)
	metrics.SetGaugeBucketInt(metrics.LeafPageN, stats.LeafPageN, name)
	metrics.SetGaugeBucketInt(metrics.LeafOverflowN, stats.LeafOverflowN, name)

	metrics.SetGaugeBucketInt(metrics.KeyN, stats.KeyN, name)
	metrics.SetGaugeBucketInt(metrics.Depth, stats.Depth, name)

	metrics.SetGaugeBucketInt(metrics.BranchAlloc, stats.BranchAlloc, name)
	metrics.SetGaugeBucketInt(metrics.BranchInuse, stats.BranchInuse, name)
	metrics.SetGaugeBucketInt(metrics.LeafAlloc, stats.LeafAlloc, name)
	metrics.SetGaugeBucketInt(metrics.LeafInuse, stats.LeafInuse, name)

	metrics.SetGaugeBucketInt(metrics.BucketN, stats.BucketN, name)
	metrics.SetGaugeBucketInt(metrics.InlineBucketN, stats.InlineBucketN, name)
	metrics.SetGaugeBucketInt(metrics.InlineBucketInuse, stats.InlineBucketInuse, name)
}

func gather(db *bolt.DB) {
	topLevelStats := db.Stats()

	metrics.SetGaugeInt(metrics.FreePageN, topLevelStats.FreePageN)
	metrics.SetGaugeInt(metrics.PendingPageN, topLevelStats.PendingPageN)
	metrics.SetGaugeInt(metrics.FreeAlloc, topLevelStats.FreeAlloc)
	metrics.SetGaugeInt(metrics.FreelistInuse, topLevelStats.FreelistInuse)
	metrics.SetGaugeInt(metrics.TxN, topLevelStats.TxN)
	metrics.SetGaugeInt(metrics.OpenTxN, topLevelStats.OpenTxN)

	// TxStats
	txStats := topLevelStats.TxStats

	metrics.SetGaugeInt(metrics.TxStatsPageCount, txStats.PageCount)
	metrics.SetGaugeInt(metrics.TxStatsPageAlloc, txStats.PageAlloc)

	metrics.SetGaugeInt(metrics.TxStatsCursorCount, txStats.CursorCount)

	metrics.SetGaugeInt(metrics.TxStatsNodeCount, txStats.NodeCount)
	metrics.SetGaugeInt(metrics.TxStatsNodeDeref, txStats.NodeDeref)

	metrics.SetGaugeInt(metrics.TxStatsRebalance, txStats.Rebalance)
	metrics.SetGaugeDuration(metrics.TxStatsRebalanceSeconds, txStats.RebalanceTime)

	metrics.SetGaugeInt(metrics.TxStatsSplit, txStats.Split)
	metrics.SetGaugeInt(metrics.TxStatsSpill, txStats.Spill)
	metrics.SetGaugeDuration(metrics.TxStatsSpillSeconds, txStats.SpillTime)

	metrics.SetGaugeInt(metrics.TxStatsWrite, txStats.Write)
	metrics.SetGaugeDuration(metrics.TxStatsWriteTime, txStats.WriteTime)

	// gather bucket stats
	_ = db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			gatherBucketStats(string(name), b.Stats())
			return nil
		})
	})

	fi, err := os.Stat(db.Path())
	if err != nil {
		log.Errorf("error getting Bolt file size: %v", err)
		return
	}
	metrics.BoltDBSize.Set(float64(fi.Size()))
}

func startMonitoring(db *bolt.DB) {
	ticker := time.NewTicker(gatherFrequency)
	for {
		<-ticker.C
		gather(db)
	}
}
