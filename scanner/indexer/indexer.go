package indexer

import (
	"context"
	"net/http"
	"os"

	"github.com/quay/claircore/datastore/postgres"
	"github.com/quay/claircore/libindex"
	"github.com/quay/claircore/pkg/ctxlock"
	"github.com/stackrox/rox/pkg/utils"
)

type Indexer struct {
	indexer *libindex.Libindex
}

func NewIndexer(ctx context.Context) (*Indexer, error) {
	// TODO: Update the conn string to something configurable.
	pool, err := postgres.Connect(ctx, "postgresql:///postgres?host=/var/run/postgresql", "libindex")
	if err != nil {
		return nil, err
	}
	store, err := postgres.InitPostgresIndexerStore(ctx, pool, true)
	if err != nil {
		return nil, err
	}
	locker, err := ctxlock.New(ctx, pool)
	if err != nil {
		return nil, err
	}

	// TODO: Update the HTTP client.
	c := http.DefaultClient
	// TODO: When adding Indexer.Close(), make sure to clean-up /tmp.
	faRoot, err := os.MkdirTemp("", "scanner-fetcharena-*")
	if err != nil {
		return nil, err
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
		return nil, err
	}

	return &Indexer{
		indexer: indexer,
	}, nil
}

func (i *Indexer) Close(ctx context.Context) error {
	return i.indexer.Close(ctx)
}
