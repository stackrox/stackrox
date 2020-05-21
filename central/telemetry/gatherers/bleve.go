package gatherers

import (
	"errors"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/pkg/telemetry/data"
)

var (
	errIndexStats = errors.New("no index stats in Bleve stats map")
)

type bleveGatherer struct {
	indexes []bleve.Index
}

func newBleveGatherer(indexes ...bleve.Index) *bleveGatherer {
	return &bleveGatherer{
		indexes: indexes,
	}
}

func getBleveSize(index bleve.Index) (int64, error) {
	statsMap := index.StatsMap()
	indexStatsInterface := statsMap["index"]
	if indexStatsInterface == nil {
		return 0, errIndexStats
	}
	indexStats, ok := indexStatsInterface.(map[string]interface{})
	if !ok {
		return 0, errIndexStats
	}
	diskSizeInterface, ok := indexStats["CurOnDiskBytes"]
	if !ok {
		return 0, errIndexStats
	}
	diskSize, ok := diskSizeInterface.(uint64)
	if !ok {
		return 0, errIndexStats
	}
	return int64(diskSize), nil
}

// Gather returns telemetry information about the bolt database used by Bleve
func (b *bleveGatherer) Gather() []*data.DatabaseStats {
	stats := make([]*data.DatabaseStats, 0, len(b.indexes))
	for _, index := range b.indexes {
		var errList []string
		size, err := getBleveSize(index)
		if err != nil {
			errList = append(errList, err.Error())
		}
		stats = append(stats, &data.DatabaseStats{
			Type:      "bleve",
			Path:      index.Name(),
			UsedBytes: size,
			Buckets:   nil,
			Errors:    errList,
		})
	}
	return stats
}
