package indexer

import (
	"context"
	"log/slog"

	"github.com/quay/claircore"
	"github.com/quay/claircore/toolkit/log"
	"github.com/stackrox/rox/pkg/scannerv4/client"
	"github.com/stackrox/rox/pkg/scannerv4/mappers"
)

// RemoteIndexer represents the interface offered by remote indexers.
type RemoteIndexer interface {
	ReportGetter
	GetRepositoryToCPEMapping(context.Context, string) (*FetchResult, error)
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
	ctx = log.With(ctx, "hash_id", hashID)
	slog.InfoContext(ctx, "fetching index report from remote indexer")
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
func (r *remoteIndexer) GetRepositoryToCPEMapping(ctx context.Context, ifModifiedSince string) (*FetchResult, error) {
	slog.InfoContext(ctx, "fetching repo-to-CPE mapping from remote indexer")

	result, err := r.indexer.GetRepositoryToCPEMapping(ctx, ifModifiedSince)
	if err != nil {
		return nil, err
	}

	return &FetchResult{
		Modified:     result.Modified,
		LastModified: result.LastModified,
		Data:         result.Data,
	}, nil
}
