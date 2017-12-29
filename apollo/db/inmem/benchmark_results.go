package inmem

import (
	"fmt"
	"sort"
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/protoconv"
	"github.com/golang/protobuf/proto"
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

func (s *benchmarkResultStore) clone(result *v1.BenchmarkResult) *v1.BenchmarkResult {
	return proto.Clone(result).(*v1.BenchmarkResult)
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
	return s.clone(benchmark), exists, nil
}

// GetBenchmarkResults applies the filters from GetBenchmarkResultsRequest and returns the Benchmarks
func (s *benchmarkResultStore) GetBenchmarkResults(request *v1.GetBenchmarkResultsRequest) ([]*v1.BenchmarkResult, error) {
	s.benchmarkMutex.Lock()
	defer s.benchmarkMutex.Unlock()
	var benchmarks []*v1.BenchmarkResult
	for _, benchmark := range s.benchmarkResults {
		if request.Host == "" {
			benchmarks = append(benchmarks, s.clone(benchmark))
		} else if benchmark.Host == request.Host {
			benchmarks = append(benchmarks, s.clone(benchmark))
		}
	}
	// Filter by start and end time if defined
	if request.ToEndTime != nil || request.FromEndTime != nil {
		filteredBenchmarks := benchmarks[:0]
		for _, benchmark := range benchmarks {
			if (request.FromEndTime == nil || protoconv.CompareProtoTimestamps(request.FromEndTime, benchmark.EndTime) != 1) &&
				(request.ToEndTime == nil || protoconv.CompareProtoTimestamps(benchmark.EndTime, request.ToEndTime) != 1) {
				filteredBenchmarks = append(filteredBenchmarks, benchmark)
			}
		}
		benchmarks = filteredBenchmarks
	}
	sort.SliceStable(benchmarks, func(i, j int) bool {
		return protoconv.CompareProtoTimestamps(benchmarks[i].EndTime, benchmarks[j].EndTime) == -1
	})
	return benchmarks, nil
}

// AddBenchmarkResult inserts a benchmark into memory
func (s *benchmarkResultStore) AddBenchmarkResult(benchmark *v1.BenchmarkResult) error {
	s.benchmarkMutex.Lock()
	defer s.benchmarkMutex.Unlock()
	if _, ok := s.benchmarkResults[benchmark.Id]; ok {
		return fmt.Errorf("Benchmark result %v cannot be added because it already exists", benchmark.Id)
	}
	if err := s.persistent.AddBenchmarkResult(benchmark); err != nil {
		return err
	}
	s.benchmarkResults[benchmark.Id] = s.clone(benchmark)
	return nil
}
