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

	persistent db.BenchmarkStorage
}

func newBenchmarkStore(persistent db.BenchmarkStorage) *benchmarkStore {
	return &benchmarkStore{
		benchmarks: make(map[string]*v1.BenchmarkPayload),
		persistent: persistent,
	}
}

func (s *benchmarkStore) loadFromPersistent() error {
	s.benchmarkMutex.Lock()
	defer s.benchmarkMutex.Unlock()
	benchmarks, err := s.persistent.GetBenchmarks(&v1.GetBenchmarksRequest{})
	if err != nil {
		return err
	}
	for _, benchmark := range benchmarks {
		s.benchmarks[benchmark.Id] = benchmark
	}
	return nil
}

// GetBenchmark retrieves a benchmark by id
func (s *benchmarkStore) GetBenchmark(id string) (benchmark *v1.BenchmarkPayload, exists bool, err error) {
	s.benchmarkMutex.Lock()
	defer s.benchmarkMutex.Unlock()
	benchmark, exists = s.benchmarks[id]
	return
}

// GetBenchmarks applies the filters from GetBenchmarksRequest and returns the Benchmarks
func (s *benchmarkStore) GetBenchmarks(request *v1.GetBenchmarksRequest) ([]*v1.BenchmarkPayload, error) {
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
			if (request.FromEndTime == nil || compareProtoTimestamps(request.FromEndTime, benchmark.EndTime) != 1) &&
				(request.ToEndTime == nil || compareProtoTimestamps(benchmark.EndTime, request.ToEndTime) != 1) {
				filteredBenchmarks = append(filteredBenchmarks, benchmark)
			}
		}
		benchmarks = filteredBenchmarks
	}
	sort.SliceStable(benchmarks, func(i, j int) bool {
		return compareProtoTimestamps(benchmarks[i].EndTime, benchmarks[j].EndTime) == -1
	})
	return benchmarks, nil
}

// AddBenchmark inserts a benchmark into memory
func (s *benchmarkStore) AddBenchmark(benchmark *v1.BenchmarkPayload) error {
	if err := s.persistent.AddBenchmark(benchmark); err != nil {
		return err
	}
	s.benchmarkMutex.Lock()
	defer s.benchmarkMutex.Unlock()
	s.benchmarks[benchmark.Id] = benchmark
	return nil
}

func compareProtoTimestamps(t1, t2 *timestamp.Timestamp) int {
	if t1 == nil {
		return -1
	}
	if t2 == nil {
		return 1
	}
	if t1.Seconds < t2.Seconds {
		return -1
	} else if t1.Seconds > t2.Seconds {
		return 1
	}
	if t1.Nanos < t2.Nanos {
		return -1
	} else if t1.Nanos > t2.Nanos {
		return 1
	}
	return 0
}
