package service

import (
	"sort"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/protoconv"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewBenchmarkResultsService returns the BenchmarkResultsService API.
func NewBenchmarkResultsService(storage db.Storage) *BenchmarkResultsService {
	return &BenchmarkResultsService{
		storage: storage,
	}
}

// BenchmarkResultsService is the struct that manages the benchmark API
type BenchmarkResultsService struct {
	storage db.BenchmarkResultsStorage
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *BenchmarkResultsService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterBenchmarkResultsServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *BenchmarkResultsService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterBenchmarkResultsServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// GetBenchmarkResults retrieves benchmark results based on the request filters
func (s *BenchmarkResultsService) GetBenchmarkResults(ctx context.Context, request *v1.GetBenchmarkResultsRequest) (*v1.GetBenchmarkResultsResponse, error) {
	benchmarks, err := s.storage.GetBenchmarkResults(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.GetBenchmarkResultsResponse{Benchmarks: benchmarks}, nil
}

// PostBenchmarkResult inserts a new benchmark result into the system
func (s *BenchmarkResultsService) PostBenchmarkResult(ctx context.Context, request *v1.BenchmarkResult) (*empty.Empty, error) {
	if err := s.storage.AddBenchmarkResult(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}

// groupPayloadsByScanID groups the benchmark results by scan ID and returns them in order (the benchmarkResults have already been filtered by the passes benchmark name)
func groupPayloadsByScanID(benchmarkResults []*v1.BenchmarkResult) *v1.GetBenchmarkResultsGroupedResponse {
	// Maps all benchmark results to a particular scan id so we can group them
	scanIDToResults := make(map[string][]*v1.BenchmarkResult)
	for _, result := range benchmarkResults {
		scanIDToResults[result.ScanId] = append(scanIDToResults[result.ScanId], result)
	}

	var response v1.GetBenchmarkResultsGroupedResponse
	for scanID, payloads := range scanIDToResults {
		// Get the number of checks run in the benchmark
		// TODO(cgorman) Could these possibly be different in the future?
		numberOfChecks := len(payloads[0].Results)

		resultsGrouped := v1.BenchmarkResultsGrouped{
			ScanId: scanID,
		}
		var scanTime *timestamp.Timestamp
		for i := 0; i < numberOfChecks; i++ {
			scopedCheckResult := &v1.BenchmarkResultsGrouped_ScopedCheckResult{
				AggregatedResults: make(map[string]int32),
			}
			for _, payload := range payloads {
				result := payload.Results[i]
				scopedCheckResult.Definition = result.Definition // This is set every loop though somewhat unnecessarily (they are all the same)
				scopedCheckResult.HostResults = append(scopedCheckResult.HostResults, &v1.BenchmarkResultsGrouped_ScopedCheckResult_HostResult{
					Host:   payload.Host,
					Result: result.Result,
					Notes:  result.Notes,
				})
				scopedCheckResult.AggregatedResults[result.Result.String()]++
				if protoconv.CompareProtoTimestamps(scanTime, payload.EndTime) == -1 {
					scanTime = payload.EndTime
				}
			}
			resultsGrouped.CheckResults = append(resultsGrouped.CheckResults, scopedCheckResult)
		}
		resultsGrouped.Time = scanTime
		response.Benchmarks = append(response.Benchmarks, &resultsGrouped)
	}

	// Sort latest to earliest
	sort.SliceStable(response.Benchmarks, func(i, j int) bool {
		return protoconv.CompareProtoTimestamps(response.Benchmarks[i].Time, response.Benchmarks[j].Time) > 0
	})
	return &response
}

// GetBenchmarkResultsGrouped retrieves benchmark results and groups them for the UI
func (s *BenchmarkResultsService) GetBenchmarkResultsGrouped(ctx context.Context, request *v1.GetBenchmarkResultsGroupedRequest) (*v1.GetBenchmarkResultsGroupedResponse, error) {
	benchmarkResults, err := s.storage.GetBenchmarkResults(&v1.GetBenchmarkResultsRequest{Benchmark: request.Benchmark})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return groupPayloadsByScanID(benchmarkResults), nil
}
