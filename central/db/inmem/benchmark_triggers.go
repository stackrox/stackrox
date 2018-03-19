package inmem

import (
	"sort"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/protoconv"
)

type benchmarkTriggerStore struct {
	db.BenchmarkTriggerStorage
}

func newBenchmarkTriggerStore(persistent db.BenchmarkTriggerStorage) *benchmarkTriggerStore {
	return &benchmarkTriggerStore{
		BenchmarkTriggerStorage: persistent,
	}
}

// GetBenchmarkTriggers returns a slice of triggers based on the request
func (s *benchmarkTriggerStore) GetBenchmarkTriggers(request *v1.GetBenchmarkTriggersRequest) ([]*v1.BenchmarkTrigger, error) {
	triggers, err := s.BenchmarkTriggerStorage.GetBenchmarkTriggers(request)
	if err != nil {
		return nil, err
	}
	idSet := stringWrap(request.GetIds()).asSet()
	clusterSet := stringWrap(request.GetClusterIds()).asSet()
	filteredTriggers := triggers[:0]
	for _, trigger := range triggers {
		if _, ok := idSet[trigger.GetId()]; len(idSet) > 0 && !ok {
			continue
		}
		// If request clusters is empty then return all
		// If the trigger has no cluster set, then it applies to all clusters
		if len(clusterSet) != 0 && len(trigger.ClusterIds) != 0 {
			var clusterMatch bool
			for _, cluster := range trigger.ClusterIds {
				if _, ok := clusterSet[cluster]; ok {
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
	sort.SliceStable(filteredTriggers, func(i, j int) bool {
		return protoconv.CompareProtoTimestamps(filteredTriggers[i].Time, filteredTriggers[j].Time) == 1
	})
	return filteredTriggers, nil
}
