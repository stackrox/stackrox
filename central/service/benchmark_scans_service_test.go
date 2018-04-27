package service

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockStorage struct{}

func (*mockStorage) ListBenchmarkScans(req *v1.ListBenchmarkScansRequest) ([]*v1.BenchmarkScanMetadata, error) {
	return []*v1.BenchmarkScanMetadata{
		{
			ScanId: "scan1",
		},
		{
			ScanId: "scan2",
		},
	}, nil
}
func (*mockStorage) GetBenchmarkScan(req *v1.GetBenchmarkScanRequest) (*v1.BenchmarkScan, bool, error) {
	switch req.GetScanId() {
	case "scan1":
		return scan, true, nil
	default:
		return nil, false, nil
	}
}
func (*mockStorage) AddScan(*v1.BenchmarkScanMetadata) error      { return nil }
func (*mockStorage) AddBenchmarkResult(*v1.BenchmarkResult) error { return nil }

var scan = &v1.BenchmarkScan{
	Checks: []*v1.BenchmarkScan_Check{
		{
			HostResults: []*v1.BenchmarkScan_Check_HostResult{
				{
					Result: v1.CheckStatus_PASS,
				},
				{
					Result: v1.CheckStatus_NOTE,
				},
			},
		},
		{
			HostResults: []*v1.BenchmarkScan_Check_HostResult{
				{
					Result: v1.CheckStatus_WARN,
				},
				{
					Result: v1.CheckStatus_NOTE,
				},
			},
		},
		{
			HostResults: []*v1.BenchmarkScan_Check_HostResult{
				{
					Result: v1.CheckStatus_WARN,
				},
				{
					Result: v1.CheckStatus_WARN,
				},
			},
		},
	},
}

var expectedGroup = &v1.BenchmarkGroup{
	Benchmark: "benchmark",
	Counts: []*v1.StatusCount{
		{
			Status: v1.CheckStatus_INFO,
			Count:  0,
		},
		{
			Status: v1.CheckStatus_WARN,
			Count:  3,
		},
		{
			Status: v1.CheckStatus_NOTE,
			Count:  2,
		},
		{
			Status: v1.CheckStatus_PASS,
			Count:  1,
		},
	},
}

func TestConvertScanDataToBenchmarkGroup(t *testing.T) {
	actualGroup := convertScanDataToBenchmarkGroup("benchmark", scan)
	assert.Equal(t, expectedGroup, actualGroup)
}

func TestGetMostRecentScanData(t *testing.T) {
	storage := &mockStorage{}
	service := &BenchmarkScansService{
		benchmarkScanStorage: storage,
	}
	benchmark := &v1.Benchmark{
		Name: "benchmark",
		Id:   "benchmarkID",
	}
	group, err := service.getMostRecentScanData("cluster", benchmark)
	require.NoError(t, err)
	assert.Equal(t, expectedGroup, group)
}

func TestGetBenchmarkScansSummary(t *testing.T) {
	benchmarks := []*v1.Benchmark{{Name: "benchmark"}}
	clusters := []*v1.Cluster{{Id: "clusterID", Name: "cluster"}}

	storage := &mockStorage{}
	service := &BenchmarkScansService{
		benchmarkScanStorage: storage,
	}
	resp, err := service.getBenchmarkScansSummaryResponse(clusters, benchmarks)
	require.NoError(t, err)

	expectedResp := &v1.GetBenchmarkScansSummaryResponse{
		Clusters: []*v1.ClusterGroup{
			{
				ClusterName: "cluster",
				ClusterId:   "clusterID",
				Benchmarks: []*v1.BenchmarkGroup{
					expectedGroup,
				},
			},
		},
	}
	assert.Equal(t, expectedResp, resp)
}
