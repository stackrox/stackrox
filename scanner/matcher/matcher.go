package matcher

import (
	"context"
	"fmt"
	"net/http"

	"github.com/quay/claircore"
	"github.com/quay/claircore/datastore/postgres"
	"github.com/quay/claircore/enricher/cvss"
	"github.com/quay/claircore/libvuln"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/pkg/ctxlock"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/scanner/config"
)

// Matcher represents a vulnerability matcher.
//
//go:generate mockgen-wrapper
type Matcher interface {
	GetVulnerabilities(ctx context.Context, ir *claircore.IndexReport) (*claircore.VulnerabilityReport, error)
	Close(ctx context.Context) error
}

// matcherImpl implements Matcher on top of a local instance of libvuln.
type matcherImpl struct {
	libVuln *libvuln.Libvuln
}

// NewMatcher creates a new matcher.
func NewMatcher(ctx context.Context, cfg config.MatcherConfig) (Matcher, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/backend/matcher")
	pool, err := postgres.Connect(ctx, cfg.Database.ConnString, "libvuln")
	if err != nil {
		return nil, fmt.Errorf("connecting to postgres for matcher: %w", err)
	}
	store, err := postgres.InitPostgresMatcherStore(ctx, pool, true)
	if err != nil {
		return nil, fmt.Errorf("initializing postgres matcher store: %w", err)
	}
	locker, err := ctxlock.New(ctx, pool)
	if err != nil {
		return nil, fmt.Errorf("creating matcher postgres locker: %w", err)
	}

	// TODO: Update HTTP client.
	c := http.DefaultClient
	libVuln, err := libvuln.New(ctx, &libvuln.Options{
		Store:  store,
		Locker: locker,
		// TODO: Enable when Scanner V4 updater pipeline is available.
		// // Run in "air-gapped" mode.
		// DisableBackgroundUpdates: true,
		UpdateRetention: libvuln.DefaultUpdateRetention,
		Client:          c,
		Enrichers: []driver.Enricher{
			&cvss.Enricher{},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("creating libvuln: %w", err)
	}
	return &matcherImpl{
		libVuln: libVuln,
	}, nil
}

func (m *matcherImpl) GetVulnerabilities(ctx context.Context, ir *claircore.IndexReport) (*claircore.VulnerabilityReport, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/backend/matcher")
	return m.libVuln.Scan(ctx, ir)
}

// Close closes the matcher.
func (m *matcherImpl) Close(ctx context.Context) error {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/backend/matcher")
	return m.libVuln.Close(ctx)
}
