package indexer

import (
	"context"
	"net/http"

	"github.com/quay/claircore/datastore/postgres"
	"github.com/quay/claircore/libindex"
	"github.com/quay/claircore/pkg/ctxlock"
)

type Indexer struct {
	indexer *libindex.Libindex
}

func NewIndexer(ctx context.Context) (*Indexer, error) {
	// TODO: Update the conn string to something configurable.
	pool, err := postgres.Connect(ctx, "TODO", "libindex")
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
	// TODO: Update the FetchArena.
	// TODO: Consider making layer scan concurrency configurable?
	opts := libindex.Options{
		Store:                store,
		Locker:               locker,
		FetchArena:           nil,
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
