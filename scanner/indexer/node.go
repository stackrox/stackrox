package indexer

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/quay/claircore"
	ccpostgres "github.com/quay/claircore/datastore/postgres"
	"github.com/quay/claircore/libindex"
	"github.com/quay/claircore/pkg/ctxlock"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/scanner/config"
	"github.com/stackrox/rox/scanner/datastore/postgres"
	"github.com/stackrox/rox/scanner/internal/httputil"
)

// NodeIndexer represents a node indexer.
//
// It is a specialized mode of [indexer.Indexer] that takes a path and scans a live filesystem
// instead of downloading and scanning layers of a container manifest.
type NodeIndexer interface {
	IndexNode(ctx context.Context, basePath string) (*claircore.IndexReport, error)
}

type localNodeIndexer struct {
	libIndex          *libindex.Libindex
	pool              *pgxpool.Pool
	root              string // do we need u?
	getIndexerTimeout time.Duration
}

func (l *localNodeIndexer) IndexNode(ctx context.Context, basePath string) (*claircore.IndexReport, error) {
	//TODO implement me
	zlog.Info(ctx).Msg("Would call index node now!")
	return nil, errors.New("Not implemented")
}

func NewNodeIndexer(ctx context.Context, cfg config.IndexerConfig) (NodeIndexer, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/backend/indexer.NewNodeIndexer")

	var success bool

	pool, err := postgres.Connect(ctx, cfg.Database.ConnString, "libindexNodeIndex")
	if err != nil {
		return nil, fmt.Errorf("connecting to postgres for indexer: %w", err)
	}
	defer func() {
		if !success {
			pool.Close()
		}
	}()

	store, err := ccpostgres.InitPostgresIndexerStore(ctx, pool, true)
	if err != nil {
		return nil, fmt.Errorf("initializing postgres indexer store: %w", err)
	}
	defer func() {
		if !success {
			_ = store.Close(ctx)
		}
	}()

	locker, err := ctxlock.New(ctx, pool)
	if err != nil {
		return nil, fmt.Errorf("creating indexer postgres locker: %w", err)
	}
	defer func() {
		if !success {
			_ = locker.Close(ctx)
		}
	}()

	root, err := os.MkdirTemp("", "scanner-fetcharena-*")
	if err != nil {
		return nil, fmt.Errorf("creating indexer root directory: %w", err)
	}
	defer func() {
		if !success {
			_ = os.RemoveAll(root)
		}
	}()

	// Note: http.DefaultTransport has already been modified to handle configured proxies.
	// See scanner/cmd/scanner/main.go.
	t, err := httputil.TransportMux(http.DefaultTransport, httputil.WithDenyStackRoxServices(!cfg.StackRoxServices))
	if err != nil {
		return nil, fmt.Errorf("creating HTTP transport: %w", err)
	}
	client := &http.Client{
		Transport: t,
	}

	indexer, err := newLibindex(ctx, cfg, client, root, store, locker)
	if err != nil {
		return nil, err
	}

	success = true
	return &localNodeIndexer{
		libIndex:          indexer,
		pool:              pool,
		root:              root,
		getIndexerTimeout: time.Duration(cfg.GetLayerTimeout),
	}, nil
}
