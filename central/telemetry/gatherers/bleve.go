package gatherers

import (
	"errors"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/pkg/telemetry/data"
)

var (
	errIndexStats = errors.New("no index stats in Bleve stats map")
)

type bleveGatherer struct {
	index bleve.Index
}

func newBleveGatherer(index bleve.Index) *bleveGatherer {
	return &bleveGatherer{
		index: index,
	}
}

func (b *bleveGatherer) getBleveSize() (int64, error) {
	statsMap := b.index.StatsMap()
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
func (b *bleveGatherer) Gather() *data.DatabaseStats {
	diskBytes, err := b.getBleveSize()
	var errList []string
	if err != nil {
		errList = append(errList, err.Error())
	}
	return &data.DatabaseStats{
		Type:      "bleve",
		Path:      globalindex.DefaultBlevePath,
		UsedBytes: diskBytes,
		Buckets:   nil,
		Errors:    errList,
	}
}
