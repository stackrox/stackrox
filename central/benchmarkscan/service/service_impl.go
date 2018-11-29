package service

import (
	"fmt"
	"sort"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	benchmarkDataStore "github.com/stackrox/rox/central/benchmark/datastore"
	"github.com/stackrox/rox/central/benchmarkscan/store"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		or.SensorOrAuthorizer(user.With(permissions.View(resources.BenchmarkScan))): {
			"/v1.BenchmarkScanService/ListBenchmarkScans",
			"/v1.BenchmarkScanService/GetBenchmarkScan",
			"/v1.BenchmarkScanService/GetHostResults",
			"/v1.BenchmarkScanService/GetBenchmarkScansSummary",
		},
		idcheck.SensorsOnly(): {
			"/v1.BenchmarkScanService/PostBenchmarkScan",
		},
	})
)

// BenchmarkScansService is the struct that manages the benchmark API
type serviceImpl struct {
	benchmarkScanStorage store.Store
	benchmarkStorage     benchmarkDataStore.DataStore
	clusterStorage       clusterDataStore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterBenchmarkScanServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterBenchmarkScanServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// PostBenchmarkScan inserts a scan into the database
func (s *serviceImpl) PostBenchmarkScan(ctx context.Context, scan *v1.BenchmarkScanMetadata) (*v1.Empty, error) {
	return &v1.Empty{}, s.benchmarkScanStorage.AddScan(scan)
}

// ListBenchmarkScans lists all of the scans that match the request parameters
func (s *serviceImpl) ListBenchmarkScans(ctx context.Context, request *v1.ListBenchmarkScansRequest) (*v1.ListBenchmarkScansResponse, error) {
	metadata, err := s.benchmarkScanStorage.ListBenchmarkScans(request)
	if err != nil {
		return nil, err
	}
	return &v1.ListBenchmarkScansResponse{
		ScanMetadata: metadata,
	}, nil
}

// GetBenchmarkScan retrieves a specific benchmark scan
func (s *serviceImpl) GetBenchmarkScan(ctx context.Context, request *v1.GetBenchmarkScanRequest) (*v1.BenchmarkScan, error) {
	if request.GetScanId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Scan ID must be defined when retrieving a scan")
	}
	scan, exists, err := s.benchmarkScanStorage.GetBenchmarkScan(request)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Could not find scan id %s", request.GetScanId()))
	}
	return scan, nil
}

func (s *serviceImpl) convertScanDataToBenchmarkGroup(benchmarkName string, scan *v1.BenchmarkScan) (*v1.BenchmarkGroup, error) {
	var scanMap = map[v1.BenchmarkCheckStatus]int64{
		v1.BenchmarkCheckStatus_PASS: 0,
		v1.BenchmarkCheckStatus_NOTE: 0,
		v1.BenchmarkCheckStatus_INFO: 0,
		v1.BenchmarkCheckStatus_WARN: 0,
	}
	for _, c := range scan.Checks {
		results, exists, err := s.benchmarkScanStorage.GetHostResults(&v1.GetHostResultsRequest{
			ScanId:    scan.GetId(),
			CheckName: c.GetDefinition().GetName(),
		})
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		for _, result := range results.HostResults {
			scanMap[result.GetResult()]++
		}
	}
	counts := make([]*v1.StatusCount, 0, len(scanMap))
	for k, v := range scanMap {
		counts = append(counts, &v1.StatusCount{
			Status: k,
			Count:  v,
		})
	}
	sort.SliceStable(counts, func(i, j int) bool { return counts[i].Status < counts[j].Status })
	return &v1.BenchmarkGroup{
		Benchmark: benchmarkName,
		Counts:    counts,
	}, nil
}

func (s *serviceImpl) getMostRecentScanData(clusterID string, benchmark *v1.Benchmark) (*v1.BenchmarkGroup, error) {
	scansMetadata, err := s.benchmarkScanStorage.ListBenchmarkScans(&v1.ListBenchmarkScansRequest{
		ClusterIds:  []string{clusterID},
		BenchmarkId: benchmark.GetId(),
	})
	if err != nil {
		return nil, err
	}
	var scan *v1.BenchmarkScan
	for _, metadata := range scansMetadata {
		var exists bool
		scan, exists, err = s.benchmarkScanStorage.GetBenchmarkScan(&v1.GetBenchmarkScanRequest{
			ScanId: metadata.GetScanId(),
		})
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		break
	}
	if scan == nil {
		return nil, nil
	}
	return s.convertScanDataToBenchmarkGroup(benchmark.GetName(), scan)
}

func (s *serviceImpl) getBenchmarkScansSummaryResponse(clusters []*v1.Cluster, benchmarks []*v1.Benchmark) (*v1.GetBenchmarkScansSummaryResponse, error) {
	response := new(v1.GetBenchmarkScansSummaryResponse)
	for _, c := range clusters {
		clusterGroup := &v1.ClusterGroup{
			ClusterName: c.GetName(),
			ClusterId:   c.GetId(),
		}
		for _, b := range benchmarks {
			benchmarkGroup, err := s.getMostRecentScanData(c.GetId(), b)
			if err != nil {
				return nil, err
			}
			if benchmarkGroup == nil {
				continue
			}
			clusterGroup.Benchmarks = append(clusterGroup.Benchmarks, benchmarkGroup)
		}
		sort.SliceStable(clusterGroup.Benchmarks, func(i, j int) bool {
			return clusterGroup.Benchmarks[i].Benchmark < clusterGroup.Benchmarks[j].Benchmark
		})
		response.Clusters = append(response.Clusters, clusterGroup)
	}
	sort.SliceStable(response.Clusters, func(i, j int) bool {
		return response.Clusters[i].GetClusterName() < response.Clusters[j].GetClusterName()
	})
	return response, nil
}

// GetBenchmarkScansSummary returns a summarized version of the clusters and their benchmarks
func (s *serviceImpl) GetBenchmarkScansSummary(context.Context, *v1.Empty) (*v1.GetBenchmarkScansSummaryResponse, error) {
	clusters, err := s.clusterStorage.GetClusters()
	if err != nil {
		return nil, err
	}
	benchmarks, err := s.benchmarkStorage.GetBenchmarks(&v1.GetBenchmarksRequest{})
	if err != nil {
		return nil, err
	}
	return s.getBenchmarkScansSummaryResponse(clusters, benchmarks)
}

// GetHostResults returns the check results for the scanid and check name specified
func (s *serviceImpl) GetHostResults(ctx context.Context, request *v1.GetHostResultsRequest) (*v1.HostResults, error) {
	hostResults, exists, err := s.benchmarkScanStorage.GetHostResults(request)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Could not find scan id %s", request.GetScanId()))
	}
	return hostResults, nil
}
