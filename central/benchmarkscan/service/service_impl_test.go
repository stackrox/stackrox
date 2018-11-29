package service

import (
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
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
func (*mockStorage) GetHostResults(request *v1.GetHostResultsRequest) (*v1.HostResults, bool, error) {
	switch request.GetCheckName() {
	case "check1":
		return &v1.HostResults{HostResults: []*v1.HostResults_HostResult{
			{
				Result: v1.BenchmarkCheckStatus_PASS,
			},
			{
				Result: v1.BenchmarkCheckStatus_INFO,
			},
		},
		}, true, nil
	case "check2":
		return &v1.HostResults{
			HostResults: []*v1.HostResults_HostResult{
				{
					Result: v1.BenchmarkCheckStatus_WARN,
				},
				{
					Result: v1.BenchmarkCheckStatus_WARN,
				},
			},
		}, true, nil
	}
	return nil, false, nil
}
func (*mockStorage) AddScan(*v1.BenchmarkScanMetadata) error      { return nil }
func (*mockStorage) AddBenchmarkResult(*v1.BenchmarkResult) error { return nil }

var scan = &v1.BenchmarkScan{
	Checks: []*v1.BenchmarkScan_Check{
		{
			Definition: &v1.BenchmarkCheckDefinition{
				Name: "check1",
			},
		},
		{
			Definition: &v1.BenchmarkCheckDefinition{
				Name: "check2",
			},
		},
	},
}

var expectedGroup = &v1.BenchmarkGroup{
	Benchmark: "benchmark",
	Counts: []*v1.StatusCount{
		{
			Status: v1.BenchmarkCheckStatus_INFO,
			Count:  1,
		},
		{
			Status: v1.BenchmarkCheckStatus_WARN,
			Count:  2,
		},
		{
			Status: v1.BenchmarkCheckStatus_NOTE,
			Count:  0,
		},
		{
			Status: v1.BenchmarkCheckStatus_PASS,
			Count:  1,
		},
	},
}

func TestConvertScanDataToBenchmarkGroup(t *testing.T) {
	storage := &mockStorage{}
	service := &serviceImpl{
		benchmarkScanStorage: storage,
	}
	actualGroup, err := service.convertScanDataToBenchmarkGroup("benchmark", scan)
	assert.NoError(t, err)
	assert.Equal(t, expectedGroup, actualGroup)
}

func TestGetMostRecentScanData(t *testing.T) {
	storage := &mockStorage{}
	service := &serviceImpl{
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
	service := &serviceImpl{
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
