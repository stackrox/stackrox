package inmem

import (
	"sort"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
)

// AddBenchmark adds a benchmark result
func (i *InMemoryStore) AddBenchmark(benchmark *v1.BenchmarkPayload) {
	i.benchmarkMutex.Lock()
	defer i.benchmarkMutex.Unlock()
	i.benchmarks[benchmark.Id] = benchmark
}

func compareProtoTimestamps(t1, t2 *timestamp.Timestamp) bool {
	if t1 == nil {
		return true
	}
	if t2 == nil {
		return false
	}
	if t1.Seconds < t2.Seconds {
		return true
	} else if t2.Seconds > t1.Seconds {
		return false
	}
	return t1.Nanos <= t2.Nanos
}

// GetBenchmarks applies the filters from GetBenchmarksRequest and returns the Benchmarks
func (i *InMemoryStore) GetBenchmarks(request *v1.GetBenchmarksRequest) []*v1.BenchmarkPayload {
	i.benchmarkMutex.Lock()
	defer i.benchmarkMutex.Unlock()
	var benchmarks []*v1.BenchmarkPayload
	for _, benchmark := range i.benchmarks {
		if request.Host == "" {
			benchmarks = append(benchmarks, benchmark)
		} else if benchmark.Host == request.Host {
			benchmarks = append(benchmarks, benchmark)
		}
	}
	// Filter by start and end time if defined
	if request.ToEndTime != nil || request.FromEndTime != nil {
		filteredBenchmarks := benchmarks[:0]
		for _, benchmark := range benchmarks {
			if compareProtoTimestamps(request.FromEndTime, benchmark.EndTime) &&
				compareProtoTimestamps(benchmark.EndTime, request.ToEndTime) {
				filteredBenchmarks = append(filteredBenchmarks, benchmark)
			}
		}
		benchmarks = filteredBenchmarks
	}
	sort.SliceStable(benchmarks, func(i, j int) bool { return compareProtoTimestamps(benchmarks[i].EndTime, benchmarks[j].EndTime) })
	return benchmarks
}
