package inmem

import (
	"sort"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/protoconv"
)

type benchmarkResultStore struct {
	db.BenchmarkResultsStorage
}

func newBenchmarkResultsStore(persistent db.BenchmarkResultsStorage) *benchmarkResultStore {
	return &benchmarkResultStore{
		BenchmarkResultsStorage: persistent,
	}
}

// GetBenchmarkResults applies the filters from GetBenchmarkResultsRequest and returns the Benchmarks
func (s *benchmarkResultStore) GetBenchmarkResults(request *v1.GetBenchmarkResultsRequest) ([]*v1.BenchmarkResult, error) {
	benchmarks, err := s.BenchmarkResultsStorage.GetBenchmarkResults(request)
	if err != nil {
		return nil, err
	}
	var filtered []*v1.BenchmarkResult
	clusterSet := stringWrap(request.GetClusters()).asSet()
	for _, benchmark := range benchmarks {
		if request.Host != "" && benchmark.Host != request.Host {
			continue
		}
		if request.ScanId != "" && benchmark.ScanId != request.ScanId {
			continue
		}
		if _, ok := clusterSet[benchmark.GetClusterId()]; len(clusterSet) > 0 && !ok {
			continue
		}
		filtered = append(filtered, benchmark)
	}
	// Filter by start and end time if defined
	if request.ToEndTime != nil || request.FromEndTime != nil {
		filteredBenchmarks := filtered[:0]
		for _, benchmark := range filtered {
			if (request.FromEndTime == nil || protoconv.CompareProtoTimestamps(request.FromEndTime, benchmark.EndTime) != 1) &&
				(request.ToEndTime == nil || protoconv.CompareProtoTimestamps(benchmark.EndTime, request.ToEndTime) != 1) {
				filteredBenchmarks = append(filteredBenchmarks, benchmark)
			}
		}
		filtered = filteredBenchmarks
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		return protoconv.CompareProtoTimestamps(filtered[i].EndTime, filtered[j].EndTime) == -1
	})
	return filtered, nil
}
