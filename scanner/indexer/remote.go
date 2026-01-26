package indexer

import (
	"context"

	"github.com/quay/claircore"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/scannerv4/client"
	"github.com/stackrox/rox/pkg/scannerv4/mappers"
	"github.com/stackrox/rox/pkg/scannerv4/repositorytocpe"
)

// RemoteIndexer represents the interface offered by remote indexers.
type RemoteIndexer interface {
	ReportGetter
	GetRepositoryToCPEMapping(ctx context.Context) (*repositorytocpe.MappingFile, error)
	Close(context.Context) error
}

// remoteIndexer is the Indexer implementation that connects to a remote indexer
// using gRPC.
type remoteIndexer struct {
	indexer client.Scanner
}

// NewRemoteIndexer connect to the gRPC address and creates a new remote indexer.
func NewRemoteIndexer(ctx context.Context, address string) (*remoteIndexer, error) {
	indexer, err := client.NewGRPCScanner(ctx, client.WithIndexerAddress(address))
	if err != nil {
		return nil, err
	}
	return &remoteIndexer{
		indexer: indexer,
	}, nil
}

// Close closes the remote indexer.
func (r *remoteIndexer) Close(_ context.Context) error {
	return r.indexer.Close()
}

// GetIndexReport calls the remote service to retrieve an IndexReport for the given hash ID.
func (r *remoteIndexer) GetIndexReport(ctx context.Context, hashID string, _ bool) (*claircore.IndexReport, bool, error) {
	ctx = zlog.ContextWithValues(ctx,
		"component", "scanner/backend/remoteIndexer.GetIndexReport",
		"hash_id", hashID,
	)
	zlog.Info(ctx).Msg("fetching index report from remote indexer")
	resp, exists, err := r.indexer.GetImageIndex(ctx, hashID)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}
	ir, err := mappers.ToClairCoreIndexReport(resp.GetContents())
	if err != nil {
		return nil, true, err
	}
	ir.State = resp.GetState()
	ir.Success = resp.GetSuccess()
	ir.Err = resp.GetErr()
	return ir, true, nil
}

// GetRepositoryToCPEMapping fetches the repository-to-CPE mapping from the remote indexer.
func (r *remoteIndexer) GetRepositoryToCPEMapping(ctx context.Context) (*repositorytocpe.MappingFile, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/backend/remoteIndexer.GetRepositoryToCPEMapping")
	zlog.Info(ctx).Msg("fetching repo-to-CPE mapping from remote indexer")
	return r.indexer.GetRepositoryToCPEMapping(ctx)
}
