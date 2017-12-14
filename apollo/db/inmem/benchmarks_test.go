package inmem

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func testBenchmarks(t *testing.T, insertStorage, retrievalStorage db.BenchmarkStorage) {
	benchmarks := []*v1.Benchmark{
		{
			Name:     "bench1",
			Editable: true,
			Checks:   []string{"CIS 1", "CIS 2"},
		},
		{
			Name:     "bench2",
			Editable: true,
			Checks:   []string{"CIS 3", "CIS 4"},
		},
	}

	// Test Add
	for _, b := range benchmarks {
		assert.NoError(t, insertStorage.AddBenchmark(b))
	}
	// Verify insertion multiple times does not deadlock and causes an error
	for _, b := range benchmarks {
		assert.Error(t, insertStorage.AddBenchmark(b))
	}

	for _, b := range benchmarks {
		got, exists, err := retrievalStorage.GetBenchmark(b.Name)
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, got, b)
	}

	// Test Update
	for _, b := range benchmarks {
		b.Checks = []string{"CIS 10"}
	}

	for _, b := range benchmarks {
		assert.NoError(t, insertStorage.UpdateBenchmark(b))
	}

	for _, b := range benchmarks {
		got, exists, err := retrievalStorage.GetBenchmark(b.GetName())
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, got, b)
	}

	// Test Remove
	for _, b := range benchmarks {
		assert.NoError(t, insertStorage.RemoveBenchmark(b.GetName()))
	}

	for _, b := range benchmarks {
		_, exists, err := retrievalStorage.GetBenchmark(b.GetName())
		assert.NoError(t, err)
		assert.False(t, exists)
	}

}

func TestBenchmarksPersistence(t *testing.T) {
	t.Parallel()
	persistent, err := createBoltDB()
	if err != nil {
		t.Fatal(err)
	}
	storage := newBenchmarkStore(persistent)
	testBenchmarks(t, storage, persistent)
}

func TestBenchmarks(t *testing.T) {
	t.Parallel()
	persistent, err := createBoltDB()
	if err != nil {
		t.Fatal(err)
	}
	storage := newBenchmarkStore(persistent)
	testBenchmarks(t, storage, storage)
}

func TestBenchmarksFiltering(t *testing.T) {
	t.Parallel()
	persistent, err := createBoltDB()
	if err != nil {
		t.Fatal(err)
	}
	storage := newBenchmarkStore(persistent)

	benchmarks := []*v1.Benchmark{
		{
			Name:     "bench1",
			Editable: true,
			Checks:   []string{"CIS 1", "CIS 2"},
		},
		{
			Name:     "bench2",
			Editable: true,
			Checks:   []string{"CIS 3", "CIS 4"},
		},
	}

	// Test Add
	for _, r := range benchmarks {
		assert.NoError(t, storage.AddBenchmark(r))
	}

	actualBenchmarks, err := storage.GetBenchmarks(&v1.GetBenchmarksRequest{})
	assert.NoError(t, err)
	assert.Equal(t, benchmarks, actualBenchmarks)

}
