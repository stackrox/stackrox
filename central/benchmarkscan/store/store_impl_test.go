package store

import (
	"os"
	"testing"

	bolt "github.com/etcd-io/bbolt"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestBenchmarkStore(t *testing.T) {
	suite.Run(t, new(BenchmarkScanStoreTestSuite))
}

type BenchmarkScanStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
}

func (suite *BenchmarkScanStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store = New(db)
}

func (suite *BenchmarkScanStoreTestSuite) TearDownSuite() {
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *BenchmarkScanStoreTestSuite) TestResults() {
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
		suite.NoError(suite.store.AddScan(m))
	}

	benchmarks := []*v1.BenchmarkResult{
		{
			BenchmarkId: "benchmark1",
			ScanId:      "scan1",
			StartTime:   ptypes.TimestampNow(),
			EndTime:     ptypes.TimestampNow(),
			Host:        "host1",
			ClusterId:   cluster1,
			Results: []*v1.BenchmarkCheckResult{
				{
					Definition: &v1.BenchmarkCheckDefinition{
						Name:        "check1",
						Description: "desc1",
					},
					Result: v1.BenchmarkCheckStatus_PASS,
				},
				{
					Definition: &v1.BenchmarkCheckDefinition{
						Name:        "check2",
						Description: "desc2",
					},
					Result: v1.BenchmarkCheckStatus_PASS,
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
			Results: []*v1.BenchmarkCheckResult{
				{
					Definition: &v1.BenchmarkCheckDefinition{
						Name:        "check1",
						Description: "desc1",
					},
					Result: v1.BenchmarkCheckStatus_WARN,
					Notes:  []string{"note1"},
				},
				{
					Definition: &v1.BenchmarkCheckDefinition{
						Name:        "check2",
						Description: "desc2",
					},
					Result: v1.BenchmarkCheckStatus_WARN,
					Notes:  []string{"note2"},
				},
			},
		},
	}

	// Test Add
	for _, b := range benchmarks {
		suite.NoError(suite.store.AddBenchmarkResult(b))
	}

	res, err := suite.store.ListBenchmarkScans(&v1.ListBenchmarkScansRequest{BenchmarkId: "benchmark1"})
	suite.NoError(err)
	suite.Len(res, 1)

	scan, exists, err := suite.store.GetBenchmarkScan(&v1.GetBenchmarkScanRequest{ScanId: "scan1"})
	suite.NoError(err)
	suite.True(exists)

	expectedScan := &v1.BenchmarkScan{
		Id: "scan1",
		Checks: []*v1.BenchmarkScan_Check{
			{
				Definition: &v1.BenchmarkCheckDefinition{
					Name:        "check1",
					Description: "desc1",
				},
				AggregatedResults: map[string]int32{
					v1.BenchmarkCheckStatus_PASS.String(): 1,
					v1.BenchmarkCheckStatus_WARN.String(): 1,
				},
			},
			{
				Definition: &v1.BenchmarkCheckDefinition{
					Name:        "check2",
					Description: "desc2",
				},
				AggregatedResults: map[string]int32{
					v1.BenchmarkCheckStatus_PASS.String(): 1,
					v1.BenchmarkCheckStatus_WARN.String(): 1,
				},
			},
		},
	}
	suite.Equal(expectedScan, scan)
}
