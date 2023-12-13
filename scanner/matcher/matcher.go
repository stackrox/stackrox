package matcher

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/quay/claircore"
	"github.com/quay/claircore/datastore/postgres"
	"github.com/quay/claircore/enricher/cvss"
	"github.com/quay/claircore/libvuln"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/pkg/ctxlock"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/scanner/config"
	metadatapostgres "github.com/stackrox/rox/scanner/matcher/metadata/postgres"
	"github.com/stackrox/rox/scanner/matcher/updater"
)

// Matcher represents a vulnerability matcher.
//
//go:generate mockgen-wrapper
type Matcher interface {
	GetVulnerabilities(ctx context.Context, ir *claircore.IndexReport) (*claircore.VulnerabilityReport, error)
	GetLastVulnerabilityUpdate(ctx context.Context) (time.Time, error)
	Close(ctx context.Context) error
}

// matcherImpl implements Matcher on top of a local instance of libvuln.
type matcherImpl struct {
	libVuln       *libvuln.Libvuln
	metadataStore *metadatapostgres.MetadataStore
}

// NewMatcher creates a new matcher.
func NewMatcher(ctx context.Context, cfg config.MatcherConfig) (Matcher, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/backend/matcher.NewMatcher")
	pool, err := postgres.Connect(ctx, cfg.Database.ConnString, "libvuln")
	if err != nil {
		return nil, fmt.Errorf("connecting to postgres for matcher: %w", err)
	}
	store, err := postgres.InitPostgresMatcherStore(ctx, pool, true)
	if err != nil {
		return nil, fmt.Errorf("initializing postgres matcher store: %w", err)
	}
	metadataStore, err := metadatapostgres.InitPostgresMetadataStore(ctx, pool, true)
	if err != nil {
		return nil, fmt.Errorf("initializing postgres matcher metadata store: %w", err)
	}
	locker, err := ctxlock.New(ctx, pool)
	if err != nil {
		return nil, fmt.Errorf("creating matcher postgres locker: %w", err)
	}

	// TODO: Update HTTP client.
	c := http.DefaultClient

	libVuln, err := libvuln.New(ctx, &libvuln.Options{
		Store:                    store,
		Locker:                   locker,
		DisableBackgroundUpdates: true,
		UpdateRetention:          libvuln.DefaultUpdateRetention,
		Client:                   c,
		Enrichers: []driver.Enricher{
			&cvss.Enricher{},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("creating libvuln: %w", err)
	}

	u, err := updater.New(updater.Opts{
		Store:         store,
		Locker:        locker,
		Pool:          pool,
		MetadataStore: metadataStore,
		Client:        c,
		// TODO: temporary URL
		URL: "https://storage.googleapis.com/scanner-v4-test/vulnerability-bundles/4.3.x-173-g6bbb2e07dc/output.json.zst",
	})
	if err != nil {
		return nil, fmt.Errorf("creating vuln updater: %w", err)
	}

	zlog.Info(ctx).Msg("starting initial update")
	if err := u.Update(ctx); err != nil {
		zlog.Error(ctx).Err(err).Msg("errors encountered during updater run")
	}
	zlog.Info(ctx).Msg("completed initial update")
	go func() {
		if err := u.Start(ctx); err != nil {
			zlog.Error(ctx).Err(err).Msg("vulnerability updater failed")
		}
	}()

	return &matcherImpl{
		libVuln:       libVuln,
		metadataStore: metadataStore,
	}, nil
}

func (m *matcherImpl) GetVulnerabilities(ctx context.Context, ir *claircore.IndexReport) (*claircore.VulnerabilityReport, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/backend/matcher.GetVulnerabilities")
	return m.libVuln.Scan(ctx, ir)
}

func (m *matcherImpl) GetLastVulnerabilityUpdate(ctx context.Context) (time.Time, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/backend/matcher.GetLastVulnerabilityUpdate")
	return m.metadataStore.GetLastVulnerabilityUpdate(ctx)
}

// Close closes the matcher.
func (m *matcherImpl) Close(ctx context.Context) error {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/backend/matcher.Close")
	return m.libVuln.Close(ctx)
}
