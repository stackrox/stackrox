package matcher

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/quay/claircore"
	ccpostgres "github.com/quay/claircore/datastore/postgres"
	"github.com/quay/claircore/libvuln"
	"github.com/quay/claircore/pkg/ctxlock"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/scanner/config"
	"github.com/stackrox/rox/scanner/datastore/postgres"
	"github.com/stackrox/rox/scanner/matcher/updater"
)

var (
	// matcherNames specifies the ClairCore matchers to use.
	// TODO(ROX-14093): add NodeJS once implemented.
	matcherNames = []string{
		"alpine-matcher",
		"aws-matcher",
		"debian-matcher",
		"gobin",
		"java-maven",
		"oracle",
		"photon",
		"python",
		"rhel-container-matcher",
		"rhel",
		"ruby-gem",
		"suse",
		"ubuntu-matcher",
	}
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
	metadataStore postgres.MatcherMetadataStore
	pool          *pgxpool.Pool

	updater *updater.Updater
}

// NewMatcher creates a new matcher.
func NewMatcher(ctx context.Context, cfg config.MatcherConfig) (Matcher, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/backend/matcher.NewMatcher")

	pool, err := ccpostgres.Connect(ctx, cfg.Database.ConnString, "libvuln")
	if err != nil {
		return nil, fmt.Errorf("connecting to postgres for matcher: %w", err)
	}
	defer func() {
		if err != nil {
			pool.Close()
		}
	}()

	store, err := ccpostgres.InitPostgresMatcherStore(ctx, pool, true)
	if err != nil {
		return nil, fmt.Errorf("initializing postgres matcher store: %w", err)
	}

	metadataStore, err := postgres.InitPostgresMatcherMetadataStore(ctx, pool, true)
	if err != nil {
		return nil, fmt.Errorf("initializing postgres matcher metadata store: %w", err)
	}

	locker, err := ctxlock.New(ctx, pool)
	if err != nil {
		return nil, fmt.Errorf("creating matcher postgres locker: %w", err)
	}
	defer utils.IgnoreError(func() error {
		if err != nil {
			return locker.Close(ctx)
		}
		return nil
	})

	// TODO(ROX-18888): Update HTTP client.
	c := http.DefaultClient

	libVuln, err := libvuln.New(ctx, &libvuln.Options{
		Store:        store,
		Locker:       locker,
		MatcherNames: matcherNames,
		// TODO(ROX-21264): Replace with our own enricher(s).
		Enrichers:                nil,
		UpdateRetention:          libvuln.DefaultUpdateRetention,
		DisableBackgroundUpdates: true,
		Client:                   c,
	})
	if err != nil {
		return nil, fmt.Errorf("creating libvuln: %w", err)
	}
	defer utils.IgnoreError(func() error {
		if err != nil {
			return libVuln.Close(ctx)
		}
		return nil
	})

	u, err := updater.New(ctx, updater.Opts{
		Store:         store,
		Locker:        locker,
		Pool:          pool,
		MetadataStore: metadataStore,
		Client:        c,
		// TODO(ROX-19005): replace with a URL related to the desired version.
		URL: "https://storage.googleapis.com/scanner-v4-test/vulnerability-bundles/dev/output.json.zst",
	})
	if err != nil {
		return nil, fmt.Errorf("creating vuln updater: %w", err)
	}

	go func() {
		if err := u.Start(); err != nil {
			zlog.Error(ctx).Err(err).Msg("vulnerability updater failed")
		}
	}()

	return &matcherImpl{
		libVuln:       libVuln,
		metadataStore: metadataStore,
		pool:          pool,

		updater: u,
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
	err := errors.Join(m.updater.Stop(), m.libVuln.Close(ctx))
	m.pool.Close()
	return err
}
