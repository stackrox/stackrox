package inmem

import (
	"fmt"
	"sort"
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/golang/protobuf/proto"
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

func (s *benchmarkStore) clone(benchmark *v1.Benchmark) *v1.Benchmark {
	return proto.Clone(benchmark).(*v1.Benchmark)
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
	return s.clone(benchmark), exists, nil
}

// GetBenchmarkResults applies the filters from GetBenchmarkResultsRequest and returns the Benchmarks
func (s *benchmarkStore) GetBenchmarks(request *v1.GetBenchmarksRequest) ([]*v1.Benchmark, error) {
	s.benchmarkMutex.Lock()
	defer s.benchmarkMutex.Unlock()
	var benchmarks []*v1.Benchmark
	for _, benchmark := range s.benchmarks {
		benchmarks = append(benchmarks, s.clone(benchmark))
	}
	sort.SliceStable(benchmarks, func(i, j int) bool {
		return benchmarks[i].Name < benchmarks[j].Name
	})
	return benchmarks, nil
}

// AddBenchmark inserts a benchmark into memory
func (s *benchmarkStore) AddBenchmark(benchmark *v1.Benchmark) error {
	s.benchmarkMutex.Lock()
	defer s.benchmarkMutex.Unlock()
	if _, ok := s.benchmarks[benchmark.Name]; ok {
		return fmt.Errorf("benchmark %v already exists", benchmark.Name)
	}
	if err := s.persistent.AddBenchmark(benchmark); err != nil {
		return err
	}
	s.benchmarks[benchmark.Name] = s.clone(benchmark)
	return nil
}

func (s *benchmarkStore) UpdateBenchmark(benchmark *v1.Benchmark) error {
	s.benchmarkMutex.Lock()
	defer s.benchmarkMutex.Unlock()
	if benchmark, ok := s.benchmarks[benchmark.Name]; ok {
		if !benchmark.Editable {
			return fmt.Errorf("Cannot update benchmark %v because it cannot be edited", benchmark.Name)
		}
	}
	if err := s.persistent.UpdateBenchmark(benchmark); err != nil {
		return err
	}
	s.benchmarks[benchmark.Name] = s.clone(benchmark)
	return nil
}

func (s *benchmarkStore) RemoveBenchmark(name string) error {
	s.benchmarkMutex.Lock()
	defer s.benchmarkMutex.Unlock()
	if benchmark, ok := s.benchmarks[name]; ok {
		if !benchmark.Editable {
			return fmt.Errorf("Cannot remove benchmark %v because it cannot be editted", name)
		}
	}
	if err := s.persistent.RemoveBenchmark(name); err != nil {
		return err
	}
	delete(s.benchmarks, name)
	return nil
}
