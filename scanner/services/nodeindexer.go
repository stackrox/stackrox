package services

import (
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/scanner/indexer"
	"google.golang.org/grpc"
)

type nodeIndexerService struct {
	v4.UnimplementedNodeIndexerServer
	nodeIndexer indexer.NodeIndexer
}

func NewNodeIndexerService(indexer indexer.NodeIndexer) *nodeIndexerService {
	return &nodeIndexerService{nodeIndexer: indexer}
}

func (n nodeIndexerService) RegisterServiceServer(server *grpc.Server) {
	//TODO implement me
	panic("implement me")
}

func (n nodeIndexerService) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	// Currently we do not set up gRPC gateway for the matcher.
	return nil
}
