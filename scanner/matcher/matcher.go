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
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/scanner/config"
	"github.com/stackrox/rox/scanner/datastore/postgres"
	"github.com/stackrox/rox/scanner/internal/httputil"
	"github.com/stackrox/rox/scanner/matcher/updater"
)

// matcherNames specifies the ClairCore matchers to use.
func matcherNames() []string {
	names := []string{
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
	if env.ScannerV4NodeJSSupport.BooleanSetting() {
		names = append(names, "nodejs")
	}
	return names
}

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

	var success bool

	pool, err := postgres.Connect(ctx, cfg.Database.ConnString, "libvuln")
	if err != nil {
		return nil, fmt.Errorf("connecting to postgres for matcher: %w", err)
	}
	defer func() {
		if !success {
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
	defer func() {
		if !success {
			_ = locker.Close(ctx)
		}
	}()

	// There should not be any network activity by the libvuln package.
	// A nil *http.Client is not allowed, so use one which denies all outbound traffic.
	ccClient := &http.Client{
		Transport: httputil.DenyTransport,
	}
	libVuln, err := libvuln.New(ctx, &libvuln.Options{
		Store:        store,
		Locker:       locker,
		MatcherNames: matcherNames(),
		// TODO(ROX-21264): Replace with our own enricher(s).
		Enrichers:                nil,
		UpdateRetention:          libvuln.DefaultUpdateRetention,
		DisableBackgroundUpdates: true,
		Client:                   ccClient,
	})
	if err != nil {
		return nil, fmt.Errorf("creating libvuln: %w", err)
	}
	defer func() {
		if !success {
			_ = libVuln.Close(ctx)
		}
	}()

	// Note: http.DefaultTransport has already been modified to handle configured proxies.
	// See scanner/cmd/scanner/main.go.
	defaultTransport := http.DefaultTransport
	// If this is a release build, Matcher should only reach out to Central,
	// so deny (and log) any other traffic.
	if buildinfo.ReleaseBuild {
		defaultTransport = httputil.DenyTransport
	}
	// Matcher should never reach out to Sensor, so ensure all Sensor-traffic is always denied.
	transport, err := httputil.TransportMux(defaultTransport, httputil.WithDenyStackRoxServices(!cfg.StackRoxServices), httputil.WithDenySensor(true))
	if err != nil {
		return nil, fmt.Errorf("creating HTTP transport: %w", err)
	}
	client := &http.Client{
		Transport: transport,
	}
	u, err := updater.New(ctx, updater.Opts{
		Store:         store,
		Locker:        locker,
		Pool:          pool,
		MetadataStore: metadataStore,
		Client:        client,
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

	success = true
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
