package boltdb

import (
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

func TestBoltBenchmarks(t *testing.T) {
	suite.Run(t, new(BoltBenchmarkTestSuite))
}

type BoltBenchmarkTestSuite struct {
	suite.Suite
	*BoltDB
}

func (suite *BoltBenchmarkTestSuite) SetupSuite() {
	db, err := boltFromTmpDir()
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.BoltDB = db
}

func (suite *BoltBenchmarkTestSuite) TeardownSuite() {
	suite.Close()
	os.Remove(suite.Path())
}

func (suite *BoltBenchmarkTestSuite) TestBenchmarks() {
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
		suite.NoError(suite.AddBenchmark(b))
	}

	for _, b := range benchmarks {
		got, exists, err := suite.GetBenchmark(b.Name)
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, b)
	}

	// Test Remove
	for _, b := range benchmarks {
		suite.NoError(suite.RemoveBenchmark(b.GetName()))
	}

	for _, b := range benchmarks {
		_, exists, err := suite.GetBenchmark(b.GetName())
		suite.NoError(err)
		suite.False(exists)
	}

	// Test Add again with no editable
	for _, b := range benchmarks {
		suite.NoError(suite.AddBenchmark(b))
	}

	// Test Update
	for _, b := range benchmarks {
		b.Editable = false
		suite.NoError(suite.UpdateBenchmark(b))
	}
	// Should be an error because the benchmarks are no longer editable
	for _, b := range benchmarks {
		suite.Error(suite.UpdateBenchmark(b))
	}

	for _, b := range benchmarks {
		suite.Error(suite.RemoveBenchmark(b.Name))
	}

}
