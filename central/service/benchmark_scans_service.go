package service

import (
	"fmt"
	"sort"

	"bitbucket.org/stack-rox/apollo/central/datastore"
	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/idcheck"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/or"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/service"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewBenchmarkScansService returns the BenchmarkScansService API.
func NewBenchmarkScansService(datastore *datastore.DataStore) *BenchmarkScansService {
	return &BenchmarkScansService{
		benchmarkScanStorage: datastore,
		benchmarkStorage:     datastore,
		clusterStorage:       datastore,
	}
}

// BenchmarkScansService is the struct that manages the benchmark API
type BenchmarkScansService struct {
	benchmarkStorage     db.BenchmarkStorage
	benchmarkScanStorage db.BenchmarkScansStorage
	clusterStorage       db.ClusterStorage
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *BenchmarkScansService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterBenchmarkScanServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *BenchmarkScansService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterBenchmarkScanServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *BenchmarkScansService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	pr := service.PerRPC{
		Default: or.SensorOrUser(),
		Authorizers: map[string]authz.Authorizer{
			"/v1.BenchmarkScansService/PostBenchmarkScan": idcheck.SensorsOnly(),
		},
	}
	return ctx, returnErrorCode(pr.Authorized(ctx, fullMethodName))
}

// PostBenchmarkScan inserts a scan into the database
func (s *BenchmarkScansService) PostBenchmarkScan(ctx context.Context, scan *v1.BenchmarkScanMetadata) (*empty.Empty, error) {
	return &empty.Empty{}, s.benchmarkScanStorage.AddScan(scan)
}

// ListBenchmarkScans lists all of the scans that match the request parameters
func (s *BenchmarkScansService) ListBenchmarkScans(ctx context.Context, request *v1.ListBenchmarkScansRequest) (*v1.ListBenchmarkScansResponse, error) {
	metadata, err := s.benchmarkScanStorage.ListBenchmarkScans(request)
	if err != nil {
		return nil, err
	}
	return &v1.ListBenchmarkScansResponse{
		ScanMetadata: metadata,
	}, nil
}

// GetBenchmarkScan retrieves a specific benchmark scan
func (s *BenchmarkScansService) GetBenchmarkScan(ctx context.Context, request *v1.GetBenchmarkScanRequest) (*v1.BenchmarkScan, error) {
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

func convertScanDataToBenchmarkGroup(benchmarkName string, scan *v1.BenchmarkScan) *v1.BenchmarkGroup {
	var scanMap = map[v1.CheckStatus]int64{
		v1.CheckStatus_PASS: 0,
		v1.CheckStatus_NOTE: 0,
		v1.CheckStatus_INFO: 0,
		v1.CheckStatus_WARN: 0,
	}
	for _, c := range scan.Checks {
		for _, result := range c.GetHostResults() {
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
	}
}

func (s *BenchmarkScansService) getMostRecentScanData(clusterID string, benchmark *v1.Benchmark) (*v1.BenchmarkGroup, error) {
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
	return convertScanDataToBenchmarkGroup(benchmark.GetName(), scan), nil
}

func (s *BenchmarkScansService) getBenchmarkScansSummaryResponse(clusters []*v1.Cluster, benchmarks []*v1.Benchmark) (*v1.GetBenchmarkScansSummaryResponse, error) {
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
func (s *BenchmarkScansService) GetBenchmarkScansSummary(context.Context, *empty.Empty) (*v1.GetBenchmarkScansSummaryResponse, error) {
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
