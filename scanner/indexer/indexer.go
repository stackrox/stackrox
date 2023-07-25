package indexer

import (
	"context"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/quay/claircore/datastore/postgres"
	"github.com/quay/claircore/libindex"
	"github.com/quay/claircore/pkg/ctxlock"
	"github.com/stackrox/rox/pkg/utils"
)

// Indexer represents an image indexer.
type Indexer struct {
	indexer *libindex.Libindex
}

// NewIndexer creates a new indexer.
func NewIndexer(ctx context.Context) (*Indexer, error) {
	// TODO: Update the conn string to something configurable.
	pool, err := postgres.Connect(ctx, "postgresql:///postgres?host=/var/run/postgresql", "libindex")
	if err != nil {
		return nil, errors.Wrap(err, "connecting to postgres for indexer")
	}
	store, err := postgres.InitPostgresIndexerStore(ctx, pool, true)
	if err != nil {
		return nil, errors.Wrap(err, "initializing postgres indexer store")
	}
	locker, err := ctxlock.New(ctx, pool)
	if err != nil {
		return nil, errors.Wrap(err, "creating indexer postgres locker")
	}

	// TODO: Update the HTTP client.
	c := http.DefaultClient
	// TODO: When adding Indexer.Close(), make sure to clean-up /tmp.
	faRoot, err := os.MkdirTemp("", "scanner-fetcharena-*")
	if err != nil {
		return nil, errors.Wrap(err, "creating indexer root directory")
	}
	defer utils.IgnoreError(func() error {
		if err != nil {
			return os.RemoveAll(faRoot)
		}
		return nil
	})
	// TODO: Consider making layer scan concurrency configurable?
	opts := libindex.Options{
		Store:                store,
		Locker:               locker,
		FetchArena:           libindex.NewRemoteFetchArena(c, faRoot),
		ScanLockRetry:        libindex.DefaultScanLockRetry,
		LayerScanConcurrency: libindex.DefaultLayerScanConcurrency,
	}

	indexer, err := libindex.New(ctx, &opts, c)
	if err != nil {
		return nil, errors.Wrap(err, "creating libindex")
	}

	return &Indexer{
		indexer: indexer,
	}, nil
}

// Close closes the indexer.
func (i *Indexer) Close(ctx context.Context) error {
	return i.indexer.Close(ctx)
}
