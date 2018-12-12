package datastore

import (
	"sort"

	"github.com/stackrox/rox/central/benchmarktrigger/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/set"
)

type datastoreImpl struct {
	storage store.Store
}

// GetBenchmarkTriggers returns a slice of triggers based on the request
func (ds *datastoreImpl) GetBenchmarkTriggers(request *v1.GetBenchmarkTriggersRequest) ([]*storage.BenchmarkTrigger, error) {
	triggers, err := ds.storage.GetBenchmarkTriggers(request)
	if err != nil {
		return nil, err
	}

	idSet := set.NewStringSet(request.GetIds()...)
	clusterSet := set.NewStringSet(request.GetClusterIds()...)
	filteredTriggers := triggers[:0]
	for _, trigger := range triggers {
		if idSet.Cardinality() > 0 && !idSet.Contains(trigger.GetId()) {
			continue
		}
		// If request clusters is empty then return all
		// If the trigger has no cluster set, then it applies to all clusters
		if clusterSet.Cardinality() != 0 && len(trigger.ClusterIds) != 0 {
			var clusterMatch bool
			for _, cluster := range trigger.ClusterIds {
				if clusterSet.Contains(cluster) {
					clusterMatch = true
					break
				}
			}
			if !clusterMatch {
				continue
			}
		}

		// Check from_time <-> end_time
		// If FromTime is after trigger time then filter out
		if request.FromTime != nil && protoconv.CompareProtoTimestamps(request.FromTime, trigger.Time) == 1 {
			continue
		}
		// If the ToTime is less than the trigger time, then filter out
		if request.ToTime != nil && protoconv.CompareProtoTimestamps(request.ToTime, trigger.Time) == -1 {
			continue
		}
		filteredTriggers = append(filteredTriggers, trigger)
	}

	sort.Slice(filteredTriggers, func(i, j int) bool {
		cmp := protoconv.CompareProtoTimestamps(filteredTriggers[i].Time, filteredTriggers[j].Time)
		if cmp < 0 {
			return true
		}
		if cmp > 0 {
			return false
		}
		return filteredTriggers[i].Id < filteredTriggers[j].Id
	})
	return filteredTriggers, nil
}

func (ds *datastoreImpl) AddBenchmarkTrigger(trigger *storage.BenchmarkTrigger) error {
	return ds.storage.AddBenchmarkTrigger(trigger)
}
