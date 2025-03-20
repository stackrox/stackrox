package matcher

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/matchers/registry"
	"github.com/quay/claircore/nodejs"
	"github.com/quay/claircore/pkg/ctxlock"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/scanner/config"
	"github.com/stackrox/rox/scanner/datastore/postgres"
	"github.com/stackrox/rox/scanner/enricher/fixedby"
	"github.com/stackrox/rox/scanner/enricher/nvd"
	"github.com/stackrox/rox/scanner/internal/httputil"
	"github.com/stackrox/rox/scanner/matcher/updater/distribution"
	"github.com/stackrox/rox/scanner/matcher/updater/vuln"
)

// matcherNames specifies the ClairCore matchers to use.
var matcherNames = []string{
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

func init() {
	// ClairCore does not register the Node.js factory by default.
	m := nodejs.Matcher{}
	mf := driver.MatcherStatic(&m)
	registry.Register(m.Name(), mf)
	matcherNames = append(matcherNames, m.Name())
}

// Matcher represents a vulnerability matcher.
//
//go:generate mockgen-wrapper
type Matcher interface {
	GetVulnerabilities(ctx context.Context, ir *claircore.IndexReport) (*claircore.VulnerabilityReport, error)
	GetLastVulnerabilityUpdate(ctx context.Context) (time.Time, error)
	GetKnownDistributions(ctx context.Context) []claircore.Distribution
	Ready(ctx context.Context) error
	Initialized(ctx context.Context) error
	Close(ctx context.Context) error
}

// matcherImpl implements Matcher on top of a local instance of libvuln.
type matcherImpl struct {
	libVuln       *libvuln.Libvuln
	metadataStore postgres.MatcherMetadataStore
	pool          *pgxpool.Pool

	vulnUpdater   *vuln.Updater
	distroUpdater *distribution.Updater
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

	store, err := postgres.InitPostgresMatcherStore(ctx, pool, true)
	if err != nil {
		return nil, fmt.Errorf("initializing postgres matcher store: %w", err)
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

	metadataStore, err := postgres.InitPostgresMatcherMetadataStore(ctx, pool, true)
	if err != nil {
		return nil, fmt.Errorf("initializing postgres matcher metadata store: %w", err)
	}

	// There should not be any network activity by the libvuln package.
	// A nil *http.Client is not allowed, so use one which denies all outbound traffic.
	ccClient := &http.Client{
		Transport: httputil.DenyTransport,
	}

	libVuln, err := libvuln.New(ctx, &libvuln.Options{
		Store:        store,
		Locker:       locker,
		MatcherNames: matcherNames,
		Enrichers: []driver.Enricher{
			&nvd.Enricher{},
			&fixedby.Enricher{},
		},
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

	// Using http.DefaultTransport instead of httputil.DefaultTransport, as the Matcher
	// should never have a need to reach out to a server with untrusted certificates.
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
	vulnUpdater, err := vuln.New(ctx, vuln.Opts{
		Store:         store,
		Locker:        locker,
		Pool:          pool,
		MetadataStore: metadataStore,
		Client:        client,
		URL:           cfg.VulnerabilitiesURL,
	})
	if err != nil {
		return nil, fmt.Errorf("creating vuln updater: %w", err)
	}

	distroUpdater, err := distribution.New(ctx, store, vulnUpdater.Initialized)
	if err != nil {
		return nil, fmt.Errorf("creating known-distribution updater: %w", err)
	}

	// Start the known-distributions updater.
	go func() {
		if err := distroUpdater.Start(); err != nil {
			zlog.Error(ctx).Err(err).Msg("known-distributions updater failed")
		}
	}()

	// Start the vulnerability updater.
	go func() {
		if err := vulnUpdater.Start(); err != nil {
			zlog.Error(ctx).Err(err).Msg("vulnerability updater failed")
		}
	}()

	success = true
	return &matcherImpl{
		libVuln:       libVuln,
		metadataStore: metadataStore,
		pool:          pool,

		vulnUpdater:   vulnUpdater,
		distroUpdater: distroUpdater,
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

func (m *matcherImpl) GetKnownDistributions(_ context.Context) []claircore.Distribution {
	return m.distroUpdater.Known()
}

// Close closes the matcher.
func (m *matcherImpl) Close(ctx context.Context) error {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/backend/matcher.Close")
	err := errors.Join(m.distroUpdater.Stop(), m.vulnUpdater.Stop(), m.libVuln.Close(ctx))
	m.pool.Close()
	return err
}

// Initialized returns nil if the matcher is fully initialized including the
// vulnerability store.  Otherwise, an error with an explanation is returned.
func (m *matcherImpl) Initialized(ctx context.Context) error {
	if !m.vulnUpdater.Initialized(ctx) {
		return errors.New("initial load for the vulnerability store is in progress")
	}
	return nil
}

// Ready returns nil if the matcher is ready to query for vulnerabilities. Notice
// the vulnerability store initial load might still be in progress. Otherwise, an
// error with an explanation is returned.
func (m *matcherImpl) Ready(ctx context.Context) error {
	if err := m.pool.Ping(ctx); err != nil {
		return fmt.Errorf("matcher vulnerability store cannot be reached: %w", err)
	}
	return nil
}
