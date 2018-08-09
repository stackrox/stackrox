package service

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	benchmarkDataStore "github.com/stackrox/rox/central/benchmark/datastore"
	"github.com/stackrox/rox/central/benchmarkscan/store"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	RegisterServiceServer(grpcServer *grpc.Server)
	RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	PostBenchmarkScan(ctx context.Context, scan *v1.BenchmarkScanMetadata) (*empty.Empty, error)
	ListBenchmarkScans(ctx context.Context, request *v1.ListBenchmarkScansRequest) (*v1.ListBenchmarkScansResponse, error)
	GetBenchmarkScan(ctx context.Context, request *v1.GetBenchmarkScanRequest) (*v1.BenchmarkScan, error)
	GetBenchmarkScansSummary(context.Context, *empty.Empty) (*v1.GetBenchmarkScansSummaryResponse, error)
	GetHostResults(ctx context.Context, request *v1.GetHostResultsRequest) (*v1.HostResults, error)
}

// New returns a new Service instance using the given DataStore.
func New(benchmarkScanStorage store.Store, benchmarkStorage benchmarkDataStore.DataStore, clusterStorage clusterDataStore.DataStore) Service {
	return &serviceImpl{
		benchmarkScanStorage: benchmarkScanStorage,
		benchmarkStorage:     benchmarkStorage,
		clusterStorage:       clusterStorage,
	}
}
