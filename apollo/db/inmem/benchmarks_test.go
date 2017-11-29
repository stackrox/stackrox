package inmem

import (
	"testing"
	"time"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
)

func testBenchmarks(t *testing.T, insertStorage, retrievalStorage db.BenchmarkStorage) {
	benchmarks := []*v1.BenchmarkPayload{
		{
			Id:        "bench1",
			StartTime: ptypes.TimestampNow(),
			EndTime:   ptypes.TimestampNow(),
			Host:      "host1",
		},
		{
			Id:        "bench2",
			StartTime: ptypes.TimestampNow(),
			EndTime:   ptypes.TimestampNow(),
			Host:      "host2",
		},
	}

	// Test Add
	for _, b := range benchmarks {
		assert.NoError(t, insertStorage.AddBenchmark(b))
	}

	for _, b := range benchmarks {
		got, exists, err := retrievalStorage.GetBenchmark(b.Id)
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, got, b)
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

	now := time.Now()
	start1, err := ptypes.TimestampProto(now.AddDate(0, 0, -4))
	assert.NoError(t, err)
	end1, err := ptypes.TimestampProto(now.AddDate(0, 0, -3))
	assert.NoError(t, err)
	start2, err := ptypes.TimestampProto(now.AddDate(0, 0, -2))
	assert.NoError(t, err)
	end2, err := ptypes.TimestampProto(now.AddDate(0, 0, -1))
	assert.NoError(t, err)

	benchmarks := []*v1.BenchmarkPayload{
		{
			Id:        "bench1",
			StartTime: start1,
			EndTime:   end1,
			Host:      "host1",
		},
		{
			Id:        "bench2",
			StartTime: start2,
			EndTime:   end2,
			Host:      "host2",
		},
	}

	// Test Add
	for _, r := range benchmarks {
		assert.NoError(t, storage.AddBenchmark(r))
	}

	actualBenchmarks, err := storage.GetBenchmarks(&v1.GetBenchmarksRequest{})
	assert.NoError(t, err)
	assert.Equal(t, benchmarks, actualBenchmarks)

	actualBenchmarks, err = storage.GetBenchmarks(&v1.GetBenchmarksRequest{Host: "host1"})
	assert.NoError(t, err)
	assert.Equal(t, benchmarks[:1], actualBenchmarks)

	// From start of time 1 to now
	actualBenchmarks, err = storage.GetBenchmarks(&v1.GetBenchmarksRequest{FromEndTime: start1})
	assert.NoError(t, err)
	assert.Equal(t, benchmarks, actualBenchmarks)

	// From beginning of time to end2
	actualBenchmarks, err = storage.GetBenchmarks(&v1.GetBenchmarksRequest{ToEndTime: end2})
	assert.NoError(t, err)
	assert.Equal(t, benchmarks, actualBenchmarks)

	// Should just be benchmark one
	actualBenchmarks, err = storage.GetBenchmarks(&v1.GetBenchmarksRequest{FromEndTime: start1, ToEndTime: start2})
	assert.NoError(t, err)
	assert.Equal(t, benchmarks[:1], actualBenchmarks)
}
