package services

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/quay/zlog"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/scanner/indexer"
	"google.golang.org/grpc"
)

var (
	_ v4.NodeIndexerServer = (*nodeIndexerService)(nil)
)

type nodeIndexerService struct {
	v4.UnimplementedNodeIndexerServer
	nodeIndexer indexer.NodeIndexer
}

func NewNodeIndexerService(indexer indexer.NodeIndexer) *nodeIndexerService {
	return &nodeIndexerService{nodeIndexer: indexer}
}

func (s *nodeIndexerService) CreateNodeIndexReport(ctx context.Context, req *v4.CreateNodeIndexReportRequest) (*v4.IndexReport, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/service/nodeIndexer.CreateNodeIndexReport")
	// TODO: Actually run the scan and create the report

	return &v4.IndexReport{
		State:    "",
		Success:  false,
		Err:      "Not implemented",
		Contents: nil,
	}, nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *nodeIndexerService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	// If this a dev build, allow anonymous traffic for testing purposes.
	if !buildinfo.ReleaseBuild {
		auth := allow.Anonymous()
		return ctx, auth.Authorized(ctx, fullMethodName)
	}

	// FIXME: Set up auth for prod builds
	return ctx, errors.New("Not implemented / unauthorized")
}

func (s *nodeIndexerService) RegisterServiceServer(server *grpc.Server) {
	v4.RegisterNodeIndexerServer(server, s)
}

func (s *nodeIndexerService) RegisterServiceHandler(_ context.Context, _ *runtime.ServeMux, _ *grpc.ClientConn) error {
	// Currently we do not set up gRPC gateway for the indexer.
	return nil
}
