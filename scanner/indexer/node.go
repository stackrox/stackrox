package indexer

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"errors"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/quay/claircore"
	ccpostgres "github.com/quay/claircore/datastore/postgres"
	ccindexer "github.com/quay/claircore/indexer"
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
//
// TODO: Find out if we really need a DB for the node indexer. Likely we need a caching layer, but not a DB.
type NodeIndexer interface {
	IndexNode(ctx context.Context) (*claircore.IndexReport, error)
	Close(ctx context.Context) error
}

type localNodeIndexer struct {
	libIndex          *libindex.Libindex
	pool              *pgxpool.Pool
	root              string // do we need u?
	getIndexerTimeout time.Duration
}

func NewNodeIndexer(ctx context.Context, cfg config.NodeIndexerConfig) (NodeIndexer, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/backend/indexer.NewNodeIndexer")

	var success bool

	pool, err := postgres.Connect(ctx, cfg.Database.ConnString, "libindexNodeIndex")
	if err != nil {
		return nil, fmt.Errorf("connecting to postgres for nodeindexer: %w", err)
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

	indexer, err := newNodeLibindex(ctx, cfg, client, root, store, locker)
	if err != nil {
		return nil, err
	}

	success = true
	return &localNodeIndexer{
		libIndex:          indexer,
		pool:              pool,
		root:              root,
		getIndexerTimeout: time.Duration(1 * time.Second),
	}, nil
}

func newNodeLibindex(ctx context.Context, _ config.NodeIndexerConfig, client *http.Client, mountPath string, store ccindexer.Store, locker *ctxlock.Locker) (*libindex.Libindex, error) {
	opts := libindex.Options{
		Store:                store,
		Locker:               locker,
		ScanLockRetry:        libindex.DefaultScanLockRetry,
		FetchArena:           libindex.NewRemoteFetchArena(client, ""), // FIXME: Unused, but required
		LayerScanConcurrency: 1,
		NoLayerValidation:    true,
		ControllerFactory:    nil, // TODO: We could go for a custom factory here as well
		Ecosystems:           ecosystems(ctx),
		ScannerConfig: struct {
			Package, Dist, Repo, File map[string]func(interface{}) error
		}{},
		Resolvers: nil,
	}

	indexer, err := libindex.NewNodeScan(ctx, &opts, client)
	if err != nil {
		return nil, fmt.Errorf("creating libindex: %w", err)
	}

	return indexer, nil
}

// IndexNode indexes a live fs.FS at the container mountpoint given in the basePath.
func (l *localNodeIndexer) IndexNode(ctx context.Context) (*claircore.IndexReport, error) {
	return l.libIndex.IndexNode(ctx)
}

// Close closes the NodeIndexer.
func (l *localNodeIndexer) Close(ctx context.Context) error {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/backend/nodeindexer.Close")
	err := errors.Join(l.libIndex.Close(ctx)) //, os.RemoveAll(l.root)) // FIXME: Close fs.FS here!
	return err
}

// Ready check function.
func (l *localNodeIndexer) Ready(_ context.Context) error {
	return nil
}
