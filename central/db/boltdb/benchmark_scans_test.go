package boltdb

import (
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/uuid"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/suite"
)

func TestBoltBenchmarkScans(t *testing.T) {
	suite.Run(t, new(BoltBenchmarkScansTestSuite))
}

type BoltBenchmarkScansTestSuite struct {
	suite.Suite
	*BoltDB
}

func (suite *BoltBenchmarkScansTestSuite) SetupSuite() {
	db, err := boltFromTmpDir()
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.BoltDB = db
}

func (suite *BoltBenchmarkScansTestSuite) TeardownSuite() {
	suite.Close()
	os.Remove(suite.Path())
}

func (suite *BoltBenchmarkScansTestSuite) TestResults() {
	cluster1 := uuid.NewV4().String()
	cluster2 := uuid.NewV4().String()
	scanMetadata := []*v1.BenchmarkScanMetadata{
		{
			ScanId:      "scan1",
			BenchmarkId: "benchmark1",
			ClusterIds:  []string{cluster1, cluster2},
			Checks:      []string{"check1", "check2"},
			Reason:      v1.BenchmarkReason_SCHEDULED,
		},
		{
			ScanId:      "scan2",
			BenchmarkId: "benchmark2",
			ClusterIds:  []string{cluster1, cluster2},
			Checks:      []string{"check1", "check2"},
			Reason:      v1.BenchmarkReason_SCHEDULED,
		},
	}
	for _, m := range scanMetadata {
		suite.NoError(suite.AddScan(m))
	}

	benchmarks := []*v1.BenchmarkResult{
		{
			BenchmarkId: "benchmark1",
			ScanId:      "scan1",
			StartTime:   ptypes.TimestampNow(),
			EndTime:     ptypes.TimestampNow(),
			Host:        "host1",
			ClusterId:   cluster1,
			Results: []*v1.CheckResult{
				{
					Definition: &v1.CheckDefinition{
						Name:        "check1",
						Description: "desc1",
					},
					Result: v1.CheckStatus_PASS,
				},
				{
					Definition: &v1.CheckDefinition{
						Name:        "check2",
						Description: "desc2",
					},
					Result: v1.CheckStatus_PASS,
				},
			},
		},
		{
			BenchmarkId: "benchmark1",
			ScanId:      "scan1",
			StartTime:   ptypes.TimestampNow(),
			EndTime:     ptypes.TimestampNow(),
			Host:        "host2",
			ClusterId:   cluster2,
			Results: []*v1.CheckResult{
				{
					Definition: &v1.CheckDefinition{
						Name:        "check1",
						Description: "desc1",
					},
					Result: v1.CheckStatus_WARN,
					Notes:  []string{"note1"},
				},
				{
					Definition: &v1.CheckDefinition{
						Name:        "check2",
						Description: "desc2",
					},
					Result: v1.CheckStatus_WARN,
					Notes:  []string{"note2"},
				},
			},
		},
	}

	// Test Add
	for _, b := range benchmarks {
		suite.NoError(suite.AddBenchmarkResult(b))
	}

	res, err := suite.ListBenchmarkScans(&v1.ListBenchmarkScansRequest{BenchmarkId: "benchmark1"})
	suite.NoError(err)
	suite.Len(res, 1)

	scan, exists, err := suite.GetBenchmarkScan(&v1.GetBenchmarkScanRequest{ScanId: "scan1"})
	suite.NoError(err)
	suite.True(exists)

	expectedScan := &v1.BenchmarkScan{
		Checks: []*v1.BenchmarkScan_Check{
			{
				Definition: &v1.CheckDefinition{
					Name:        "check1",
					Description: "desc1",
				},
				HostResults: []*v1.BenchmarkScan_Check_HostResult{
					{
						Host:   "host1",
						Result: v1.CheckStatus_PASS,
					},
					{
						Host:   "host2",
						Result: v1.CheckStatus_WARN,
						Notes:  []string{"note1"},
					},
				},
				AggregatedResults: map[string]int32{
					v1.CheckStatus_PASS.String(): 1,
					v1.CheckStatus_WARN.String(): 1,
				},
			},
			{
				Definition: &v1.CheckDefinition{
					Name:        "check2",
					Description: "desc2",
				},
				HostResults: []*v1.BenchmarkScan_Check_HostResult{
					{
						Host:   "host1",
						Result: v1.CheckStatus_PASS,
					},
					{
						Host:   "host2",
						Result: v1.CheckStatus_WARN,
						Notes:  []string{"note2"},
					},
				},
				AggregatedResults: map[string]int32{
					v1.CheckStatus_PASS.String(): 1,
					v1.CheckStatus_WARN.String(): 1,
				},
			},
		},
	}
	suite.Equal(expectedScan, scan)
}
