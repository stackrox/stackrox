package store

import (
	"os"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stretchr/testify/suite"
)

func TestBenchmarkStore(t *testing.T) {
	suite.Run(t, new(BenchmarkStoreTestSuite))
}

type BenchmarkStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
}

func (suite *BenchmarkStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store = New(db)
}

func (suite *BenchmarkStoreTestSuite) TeardownSuite() {
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *BenchmarkStoreTestSuite) TestBenchmarks() {
	benchmarks := []*v1.Benchmark{
		{
			Name:     "bench1",
			Editable: true,
			Checks:   []string{"CIS Docker v1.1.0 - 1", "CIS Docker v1.1.0 - 2"},
		},
		{
			Name:     "bench2",
			Editable: true,
			Checks:   []string{"CIS Docker v1.1.0 - 3", "CIS Docker v1.1.0 - 4"},
		},
	}

	// Test Add
	for _, b := range benchmarks {
		id, err := suite.store.AddBenchmark(b)
		suite.NoError(err)
		suite.NotEmpty(id)
	}

	for _, b := range benchmarks {
		got, exists, err := suite.store.GetBenchmark(b.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, b)
	}

	// Test Remove
	for _, b := range benchmarks {
		suite.NoError(suite.store.RemoveBenchmark(b.GetId()))
	}

	for _, b := range benchmarks {
		_, exists, err := suite.store.GetBenchmark(b.GetId())
		suite.NoError(err)
		suite.False(exists)
	}

	// Test Add again with no editable
	for _, b := range benchmarks {
		id, err := suite.store.AddBenchmark(b)
		suite.NoError(err)
		suite.NotEmpty(id)
	}

	// Test Update
	for _, b := range benchmarks {
		b.Editable = false
		suite.NoError(suite.store.UpdateBenchmark(b))
	}
	// Should be an error because the benchmarks are no longer editable
	for _, b := range benchmarks {
		suite.Error(suite.store.UpdateBenchmark(b))
	}

	for _, b := range benchmarks {
		suite.Error(suite.store.RemoveBenchmark(b.GetId()))
	}
}
