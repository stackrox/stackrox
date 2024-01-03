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
	updaterdefaults "github.com/quay/claircore/updater/defaults"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/scanner/config"
	"github.com/stackrox/rox/scanner/updater/rhel"
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
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/backend/matcher.NewMatcher")
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
	// TODO: Remove when Scanner V4 updater pipeline is available.
	if err := updaterdefaults.Error(); err != nil {
		return nil, fmt.Errorf("vulnerability updater init: %w", err)
	}
	// TODO: Update HTTP client.
	c := http.DefaultClient
	// TODO: Disable when Scanner V4 updater pipeline is available.
	disableUpdaters := false
	var oot []driver.Updater
	var updaterSets []string
	if !disableUpdaters {
		zlog.Info(ctx).Msg("creating out-of-tree RHEL updaters")
		oot, err = rhel.Updaters(ctx, c)
		if err != nil {
			return nil, fmt.Errorf("out-of-tree RHEL updaters: %w", err)
		}
		updaterSets = []string{
			"alpine",
			"aws",
			"debian",
			"oracle",
			"photon",
			"pyupio",
			"suse",
			"ubuntu",
		}
	}
	libVuln, err := libvuln.New(ctx, &libvuln.Options{
		Store:                    store,
		Locker:                   locker,
		DisableBackgroundUpdates: disableUpdaters,
		UpdateRetention:          libvuln.DefaultUpdateRetention,
		Client:                   c,
		Enrichers: []driver.Enricher{
			&cvss.Enricher{},
		},
		UpdaterSets: updaterSets,
		Updaters:    oot,
	})
	if err != nil {
		return nil, fmt.Errorf("creating libvuln: %w", err)
	}
	return &matcherImpl{
		libVuln: libVuln,
	}, nil
}

func (m *matcherImpl) GetVulnerabilities(ctx context.Context, ir *claircore.IndexReport) (*claircore.VulnerabilityReport, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/backend/matcher.GetVulnerabilities")
	return m.libVuln.Scan(ctx, ir)
}

// Close closes the matcher.
func (m *matcherImpl) Close(ctx context.Context) error {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/backend/matcher.Close")
	return m.libVuln.Close(ctx)
}
