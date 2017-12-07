package inmem

import (
	"sort"
	"sync"

	"fmt"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type benchmarkStore struct {
	benchmarks     map[string]*v1.Benchmark
	benchmarkMutex sync.Mutex

	persistent db.BenchmarkStorage
}

func newBenchmarkStore(persistent db.BenchmarkStorage) *benchmarkStore {
	return &benchmarkStore{
		benchmarks: make(map[string]*v1.Benchmark),
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
		s.benchmarks[benchmark.Name] = benchmark
	}
	return nil
}

// GetBenchmarkResult retrieves a benchmark by id
func (s *benchmarkStore) GetBenchmark(name string) (benchmark *v1.Benchmark, exists bool, err error) {
	s.benchmarkMutex.Lock()
	defer s.benchmarkMutex.Unlock()
	benchmark, exists = s.benchmarks[name]
	return
}

// GetBenchmarkResults applies the filters from GetBenchmarkResultsRequest and returns the Benchmarks
func (s *benchmarkStore) GetBenchmarks(request *v1.GetBenchmarksRequest) ([]*v1.Benchmark, error) {
	s.benchmarkMutex.Lock()
	defer s.benchmarkMutex.Unlock()
	var benchmarks []*v1.Benchmark
	for _, benchmark := range s.benchmarks {
		benchmarks = append(benchmarks, benchmark)
	}
	sort.SliceStable(benchmarks, func(i, j int) bool {
		return benchmarks[i].Name < benchmarks[j].Name
	})
	return benchmarks, nil
}

// AddBenchmark inserts a benchmark into memory
func (s *benchmarkStore) AddBenchmark(benchmark *v1.Benchmark) error {
	s.benchmarkMutex.Lock()
	if _, ok := s.benchmarks[benchmark.Name]; ok {
		return fmt.Errorf("benchmark %v already exists", benchmark.Name)
	}
	s.benchmarkMutex.Unlock()
	if err := s.persistent.AddBenchmark(benchmark); err != nil {
		return err
	}
	s.benchmarkMutex.Lock()
	defer s.benchmarkMutex.Unlock()
	s.benchmarks[benchmark.Name] = benchmark
	return nil
}

func (s *benchmarkStore) UpdateBenchmark(benchmark *v1.Benchmark) error {
	if err := s.persistent.UpdateBenchmark(benchmark); err != nil {
		return err
	}
	s.benchmarkMutex.Lock()
	defer s.benchmarkMutex.Unlock()
	s.benchmarks[benchmark.Name] = benchmark
	return nil
}

func (s *benchmarkStore) RemoveBenchmark(name string) error {
	if err := s.persistent.RemoveBenchmark(name); err != nil {
		return err
	}
	s.benchmarkMutex.Lock()
	defer s.benchmarkMutex.Unlock()
	delete(s.benchmarks, name)
	return nil
}
