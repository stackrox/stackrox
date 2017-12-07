package inmem

import (
	"sort"
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
)

type benchmarkResultStore struct {
	benchmarkResults map[string]*v1.BenchmarkResult
	benchmarkMutex   sync.Mutex

	persistent db.BenchmarkResultsStorage
}

func newBenchmarkResultsStore(persistent db.BenchmarkResultsStorage) *benchmarkResultStore {
	return &benchmarkResultStore{
		benchmarkResults: make(map[string]*v1.BenchmarkResult),
		persistent:       persistent,
	}
}

func (s *benchmarkResultStore) loadFromPersistent() error {
	s.benchmarkMutex.Lock()
	defer s.benchmarkMutex.Unlock()
	benchmarks, err := s.persistent.GetBenchmarkResults(&v1.GetBenchmarkResultsRequest{})
	if err != nil {
		return err
	}
	for _, benchmark := range benchmarks {
		s.benchmarkResults[benchmark.Id] = benchmark
	}
	return nil
}

// GetBenchmarkResult retrieves a benchmark by id
func (s *benchmarkResultStore) GetBenchmarkResult(id string) (benchmark *v1.BenchmarkResult, exists bool, err error) {
	s.benchmarkMutex.Lock()
	defer s.benchmarkMutex.Unlock()
	benchmark, exists = s.benchmarkResults[id]
	return
}

// GetBenchmarkResults applies the filters from GetBenchmarkResultsRequest and returns the Benchmarks
func (s *benchmarkResultStore) GetBenchmarkResults(request *v1.GetBenchmarkResultsRequest) ([]*v1.BenchmarkResult, error) {
	s.benchmarkMutex.Lock()
	defer s.benchmarkMutex.Unlock()
	var benchmarks []*v1.BenchmarkResult
	for _, benchmark := range s.benchmarkResults {
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

// AddBenchmarkResult inserts a benchmark into memory
func (s *benchmarkResultStore) AddBenchmarkResult(benchmark *v1.BenchmarkResult) error {
	if err := s.persistent.AddBenchmarkResult(benchmark); err != nil {
		return err
	}
	s.benchmarkMutex.Lock()
	defer s.benchmarkMutex.Unlock()
	s.benchmarkResults[benchmark.Id] = benchmark
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
