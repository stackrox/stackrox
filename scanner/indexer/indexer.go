package indexer

import (
	"context"
	"net/http"
	"os"

	"github.com/quay/claircore"
	"github.com/quay/claircore/datastore/postgres"
	"github.com/quay/claircore/libindex"
	"github.com/quay/claircore/pkg/ctxlock"
)

type Indexer struct {
	fetcher *localFetchArena
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
	// TODO: Replace os.TempDir().
	fetcher := newLocalFetchArena(os.TempDir())
	// TODO: Consider making layer scan concurrency configurable?
	opts := libindex.Options{
		Store:                store,
		Locker:               locker,
		FetchArena:           fetcher,
		ScanLockRetry:        libindex.DefaultScanLockRetry,
		LayerScanConcurrency: libindex.DefaultLayerScanConcurrency,
	}

	indexer, err := libindex.New(ctx, &opts, c)
	if err != nil {
		return nil, err
	}

	return &Indexer{
		fetcher: fetcher,
		indexer: indexer,
	}, nil
}

// Index indexes the given image and returns the index report.
func (i *Indexer) Index(ctx context.Context, image string, opts ...Option) (*claircore.IndexReport, error) {
	m, err := i.fetcher.Get(ctx, image, opts...)
	if err != nil {
		return nil, err
	}

	ir, err := i.indexer.Index(ctx, m)
	if err != nil {
		return nil, err
	}

	return ir, nil
}
