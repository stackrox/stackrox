package service

import (
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockStorage struct{}

func (*mockStorage) ListBenchmarkScans(req *v1.ListBenchmarkScansRequest) ([]*storage.BenchmarkScanMetadata, error) {
	return []*storage.BenchmarkScanMetadata{
		{
			ScanId: "scan1",
		},
		{
			ScanId: "scan2",
		},
	}, nil
}
func (*mockStorage) GetBenchmarkScan(req *v1.GetBenchmarkScanRequest) (*storage.BenchmarkScan, bool, error) {
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
				Result: storage.BenchmarkCheckStatus_PASS,
			},
			{
				Result: storage.BenchmarkCheckStatus_INFO,
			},
		},
		}, true, nil
	case "check2":
		return &v1.HostResults{
			HostResults: []*v1.HostResults_HostResult{
				{
					Result: storage.BenchmarkCheckStatus_WARN,
				},
				{
					Result: storage.BenchmarkCheckStatus_WARN,
				},
			},
		}, true, nil
	}
	return nil, false, nil
}
func (*mockStorage) AddScan(*storage.BenchmarkScanMetadata) error      { return nil }
func (*mockStorage) AddBenchmarkResult(*storage.BenchmarkResult) error { return nil }

var scan = &storage.BenchmarkScan{
	Checks: []*storage.BenchmarkScan_Check{
		{
			Definition: &storage.BenchmarkCheckDefinition{
				Name: "check1",
			},
		},
		{
			Definition: &storage.BenchmarkCheckDefinition{
				Name: "check2",
			},
		},
	},
}

var expectedGroup = &v1.BenchmarkGroup{
	Benchmark: "benchmark",
	Counts: []*v1.StatusCount{
		{
			Status: storage.BenchmarkCheckStatus_INFO,
			Count:  1,
		},
		{
			Status: storage.BenchmarkCheckStatus_WARN,
			Count:  2,
		},
		{
			Status: storage.BenchmarkCheckStatus_NOTE,
			Count:  0,
		},
		{
			Status: storage.BenchmarkCheckStatus_PASS,
			Count:  1,
		},
	},
}

func TestConvertScanDataToBenchmarkGroup(t *testing.T) {
	service := &serviceImpl{
		benchmarkScanStorage: &mockStorage{},
	}
	actualGroup, err := service.convertScanDataToBenchmarkGroup("benchmark", scan)
	assert.NoError(t, err)
	assert.Equal(t, expectedGroup, actualGroup)
}

func TestGetMostRecentScanData(t *testing.T) {
	service := &serviceImpl{
		benchmarkScanStorage: &mockStorage{},
	}
	benchmark := &storage.Benchmark{
		Name: "benchmark",
		Id:   "benchmarkID",
	}
	group, err := service.getMostRecentScanData("cluster", benchmark)
	require.NoError(t, err)
	assert.Equal(t, expectedGroup, group)
}

func TestGetBenchmarkScansSummary(t *testing.T) {
	benchmarks := []*storage.Benchmark{{Name: "benchmark"}}
	clusters := []*storage.Cluster{{Id: "clusterID", Name: "cluster"}}

	service := &serviceImpl{
		benchmarkScanStorage: &mockStorage{},
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
