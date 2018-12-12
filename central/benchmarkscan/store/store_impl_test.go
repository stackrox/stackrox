package store

import (
	"os"
	"testing"

	bolt "github.com/etcd-io/bbolt"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
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
	scanMetadata := []*storage.BenchmarkScanMetadata{
		{
			ScanId:      "scan1",
			BenchmarkId: "benchmark1",
			ClusterIds:  []string{cluster1, cluster2},
			Checks:      []string{"check1", "check2"},
			Reason:      storage.BenchmarkReason_SCHEDULED,
		},
		{
			ScanId:      "scan2",
			BenchmarkId: "benchmark2",
			ClusterIds:  []string{cluster1, cluster2},
			Checks:      []string{"check1", "check2"},
			Reason:      storage.BenchmarkReason_SCHEDULED,
		},
	}
	for _, m := range scanMetadata {
		suite.NoError(suite.store.AddScan(m))
	}

	benchmarks := []*storage.BenchmarkResult{
		{
			BenchmarkId: "benchmark1",
			ScanId:      "scan1",
			StartTime:   ptypes.TimestampNow(),
			EndTime:     ptypes.TimestampNow(),
			Host:        "host1",
			ClusterId:   cluster1,
			Results: []*storage.BenchmarkCheckResult{
				{
					Definition: &storage.BenchmarkCheckDefinition{
						Name:        "check1",
						Description: "desc1",
					},
					Result: storage.BenchmarkCheckStatus_PASS,
				},
				{
					Definition: &storage.BenchmarkCheckDefinition{
						Name:        "check2",
						Description: "desc2",
					},
					Result: storage.BenchmarkCheckStatus_PASS,
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
			Results: []*storage.BenchmarkCheckResult{
				{
					Definition: &storage.BenchmarkCheckDefinition{
						Name:        "check1",
						Description: "desc1",
					},
					Result: storage.BenchmarkCheckStatus_WARN,
					Notes:  []string{"note1"},
				},
				{
					Definition: &storage.BenchmarkCheckDefinition{
						Name:        "check2",
						Description: "desc2",
					},
					Result: storage.BenchmarkCheckStatus_WARN,
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

	expectedScan := &storage.BenchmarkScan{
		Id: "scan1",
		Checks: []*storage.BenchmarkScan_Check{
			{
				Definition: &storage.BenchmarkCheckDefinition{
					Name:        "check1",
					Description: "desc1",
				},
				AggregatedResults: map[string]int32{
					storage.BenchmarkCheckStatus_PASS.String(): 1,
					storage.BenchmarkCheckStatus_WARN.String(): 1,
				},
			},
			{
				Definition: &storage.BenchmarkCheckDefinition{
					Name:        "check2",
					Description: "desc2",
				},
				AggregatedResults: map[string]int32{
					storage.BenchmarkCheckStatus_PASS.String(): 1,
					storage.BenchmarkCheckStatus_WARN.String(): 1,
				},
			},
		},
	}
	suite.Equal(expectedScan, scan)
}
