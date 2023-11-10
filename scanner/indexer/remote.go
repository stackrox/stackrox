package indexer

import (
	"context"
	"fmt"

	"github.com/quay/claircore"
	"github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/scanner/mappers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// remoteIndexer is the Indexer implementation that connects to a remote indexer
// using gRPC.
type remoteIndexer struct {
	conn    *grpc.ClientConn
	indexer v4.IndexerClient
}

// NewRemoteIndexer connect to the gRPC address and creates a new remote indexer.
func NewRemoteIndexer(ctx context.Context, address string) (*remoteIndexer, error) {
	connOpt := clientconn.Options{
		TLS: clientconn.TLSConfigOptions{
			GRPCOnly:           true,
			InsecureSkipVerify: true,
		},
	}
	// TODO: [ROX-19050] Set the Scanner V4 TLS validation and the correct subject
	//       when certificates are ready.
	conn, err := clientconn.GRPCConnection(ctx, mtls.ScannerSubject, address, connOpt)
	if err != nil {
		return nil, err
	}
	return &remoteIndexer{
		conn:    conn,
		indexer: v4.NewIndexerClient(conn),
	}, nil
}

// Close closes the remote indexer.
func (r *remoteIndexer) Close(ctx context.Context) error {
	return r.conn.Close()
}

// GetIndexReport call the remote service to retrieve an IndexReport for the given hash ID.
func (r *remoteIndexer) GetIndexReport(ctx context.Context, hashID string) (*claircore.IndexReport, bool, error) {
	resp, err := r.indexer.GetIndexReport(ctx, &v4.GetIndexReportRequest{HashId: hashID})
	if err != nil {
		if e, ok := status.FromError(err); ok && e.Code() == codes.NotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	if !resp.GetSuccess() {
		return nil, true, fmt.Errorf("report failed: state %s: %s", resp.GetState(), resp.GetErr())
	}
	ir, err := mappers.ToClairCoreIndexReport(resp.GetContents())
	if err != nil {
		return nil, true, err
	}
	return ir, true, nil
}
