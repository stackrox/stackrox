package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"google.golang.org/grpc"
)

var log = logging.LoggerForModule()

type serviceImpl struct {
	central.UnimplementedScannerServiceServer
	reprocessor reprocessor.Loop
}

// New creates a new ScannerService.
func New(reprocessor reprocessor.Loop) *serviceImpl {
	return &serviceImpl{reprocessor: reprocessor}
}

func (s *serviceImpl) NotifyScannerReady(ctx context.Context, _ *protocompat.Empty) (*protocompat.Empty, error) {
	log.Info("Scanner V4 ready, triggering image reprocessing")
	s.reprocessor.ShortCircuit()
	return protocompat.ProtoEmpty(), nil
}

func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	central.RegisterScannerServiceServer(server, s)
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, or.ScannerV4().Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) RegisterServiceHandler(_ context.Context, _ *runtime.ServeMux, _ *grpc.ClientConn) error {
	return nil
}
