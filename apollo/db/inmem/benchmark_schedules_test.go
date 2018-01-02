package inmem

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
)

func testBenchmarkSchedules(t *testing.T, insertStorage, retrievalStorage db.BenchmarkScheduleStorage) {
	benchmarkSchedules := []*v1.BenchmarkSchedule{
		{
			Name:         "bench1",
			StartTime:    ptypes.TimestampNow(),
			IntervalDays: 1,
			Clusters:     []string{"dev"},
		},
		{
			Name:         "bench2",
			StartTime:    ptypes.TimestampNow(),
			IntervalDays: 2,
			Clusters:     []string{"prod"},
		},
	}

	// Test Add
	for _, b := range benchmarkSchedules {
		assert.NoError(t, insertStorage.AddBenchmarkSchedule(b))
	}
	// Verify insertion multiple times does not deadlock and causes an error
	for _, b := range benchmarkSchedules {
		assert.Error(t, insertStorage.AddBenchmarkSchedule(b))
	}

	for _, b := range benchmarkSchedules {
		got, exists, err := retrievalStorage.GetBenchmarkSchedule(b.Name)
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, got, b)
	}

}

func TestBenchmarkSchedulesPersistence(t *testing.T) {
	t.Parallel()
	persistent, err := createBoltDB()
	if err != nil {
		t.Fatal(err)
	}
	storage := newBenchmarkScheduleStore(persistent)
	testBenchmarkSchedules(t, storage, persistent)
}

func TestBenchmarkSchedules(t *testing.T) {
	t.Parallel()
	persistent, err := createBoltDB()
	if err != nil {
		t.Fatal(err)
	}
	storage := newBenchmarkScheduleStore(persistent)
	testBenchmarkSchedules(t, storage, storage)
}
