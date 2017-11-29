package inmem

import (
	"sort"
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
)

type benchmarkStore struct {
	benchmarks     map[string]*v1.BenchmarkPayload
	benchmarkMutex sync.Mutex

	persistent db.Storage
}

func newBenchmarkStore(persistent db.Storage) *benchmarkStore {
	return &benchmarkStore{
		benchmarks: make(map[string]*v1.BenchmarkPayload),
		persistent: persistent,
	}
}

// AddBenchmark adds a benchmark result
func (s *benchmarkStore) AddBenchmark(benchmark *v1.BenchmarkPayload) {
	s.benchmarkMutex.Lock()
	defer s.benchmarkMutex.Unlock()
	s.benchmarks[benchmark.Id] = benchmark
}

// GetBenchmarks applies the filters from GetBenchmarksRequest and returns the Benchmarks
func (s *benchmarkStore) GetBenchmarks(request *v1.GetBenchmarksRequest) []*v1.BenchmarkPayload {
	s.benchmarkMutex.Lock()
	defer s.benchmarkMutex.Unlock()
	var benchmarks []*v1.BenchmarkPayload
	for _, benchmark := range s.benchmarks {
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
