package matcher

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/quay/claircore"
	"github.com/quay/claircore/alpine"
	"github.com/quay/claircore/aws"
	"github.com/quay/claircore/debian"
	"github.com/quay/claircore/enricher/epss"
	"github.com/quay/claircore/gobin"
	"github.com/quay/claircore/java"
	"github.com/quay/claircore/libvuln"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/matchers/registry"
	"github.com/quay/claircore/nodejs"
	"github.com/quay/claircore/oracle"
	"github.com/quay/claircore/photon"
	"github.com/quay/claircore/pkg/ctxlock"
	"github.com/quay/claircore/python"
	"github.com/quay/claircore/rhel"
	"github.com/quay/claircore/rhel/rhcc"
	"github.com/quay/claircore/ruby"
	"github.com/quay/claircore/suse"
	"github.com/quay/claircore/ubuntu"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/scanner/config"
	"github.com/stackrox/rox/scanner/datastore/postgres"
	"github.com/stackrox/rox/scanner/enricher/csaf"
	"github.com/stackrox/rox/scanner/enricher/fixedby"
	"github.com/stackrox/rox/scanner/enricher/nvd"
	"github.com/stackrox/rox/scanner/internal/httputil"
	"github.com/stackrox/rox/scanner/matcher/updater/vuln"
	"github.com/stackrox/rox/scanner/sbom"
)

// matcherNames specifies the matchers to use for vulnerability matching.
func matcherNames() []string {
	// Note: Do NOT hardcode the names. It's very easy to mess up...
	ms := []string{
		(*alpine.Matcher)(nil).Name(),
		(*aws.Matcher)(nil).Name(),
		(*debian.Matcher)(nil).Name(),
		(*oracle.Matcher)(nil).Name(),
		(*photon.Matcher)(nil).Name(),
		rhcc.Matcher.Name(),
		(*rhel.Matcher)(nil).Name(),
		(*suse.Matcher)(nil).Name(),
		(*ubuntu.Matcher)(nil).Name(),
	}
	if features.ScannerV4LanguageSupport.Enabled() {
		// Claircore does not register the Node.js factory by default, so register it here.
		nodeJSMatcher := nodejs.Matcher{}
		registry.Register(nodeJSMatcher.Name(), driver.MatcherStatic(&nodeJSMatcher))
		ms = append(ms,
			(*gobin.Matcher)(nil).Name(),
			(*java.Matcher)(nil).Name(),
			nodeJSMatcher.Name(),
			(*python.Matcher)(nil).Name(),
			(*ruby.Matcher)(nil).Name(),
		)
	}
	return ms
}

// Matcher represents a vulnerability matcher.
//
//go:generate mockgen-wrapper
type Matcher interface {
	GetVulnerabilities(ctx context.Context, ir *claircore.IndexReport) (*claircore.VulnerabilityReport, error)
	GetLastVulnerabilityUpdate(ctx context.Context) (time.Time, error)
	GetKnownDistributions(ctx context.Context) []claircore.Distribution
	GetSBOM(ctx context.Context, ir *claircore.IndexReport, opts *sbom.Options) ([]byte, error)
	Ready(ctx context.Context) error
	Initialized(ctx context.Context) error
	Close(ctx context.Context) error
}

// matcherImpl implements Matcher on top of a local instance of libvuln.
type matcherImpl struct {
	libVuln       *libvuln.Libvuln
	metadataStore postgres.MatcherMetadataStore
	pool          *pgxpool.Pool

	vulnUpdater *vuln.Updater
	sbomer      *sbom.SBOMer

	readyWithVulns bool
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

	enrichers := []driver.Enricher{
		&fixedby.Enricher{},
		&nvd.Enricher{},
	}
	var (
		epssEnabled bool
		csafEnabled bool
	)
	if features.EPSSScore.Enabled() {
		epssEnabled = true
		enrichers = append(enrichers, &epss.Enricher{})
	}
	if features.ScannerV4RedHatCSAF.Enabled() && !features.ScannerV4RedHatCVEs.Enabled() {
		csafEnabled = true
		enrichers = append(enrichers, &csaf.Enricher{})
	}
	zlog.Info(ctx).Bool("enabled", epssEnabled).Msg("EPSS enrichment")
	zlog.Info(ctx).Bool("enabled", csafEnabled).Msg("CSAF enrichment")
	libVuln, err := libvuln.New(ctx, &libvuln.Options{
		Store:                    store,
		Locker:                   locker,
		MatcherNames:             matcherNames(),
		Enrichers:                enrichers,
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
		MetadataStore: metadataStore,
		Client:        client,
		URL:           cfg.VulnerabilitiesURL,
	})
	if err != nil {
		return nil, fmt.Errorf("creating vuln updater: %w", err)
	}

	// SBOM generation capabilities are only avail via the matcher.
	// SBOMs may optionally include vulnerabilities which aligns SBOM
	// generation to matcher capabilities and reduces the complexity
	// of routing requests differently based on if a user chooses to
	// include vulnerabilities vs. not.
	sbomer := sbom.NewSBOMer()

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

		vulnUpdater: vulnUpdater,
		sbomer:      sbomer,

		readyWithVulns: cfg.Readiness == config.ReadinessVulnerability,
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
	return m.vulnUpdater.KnownDistributions()
}

func (m *matcherImpl) GetSBOM(ctx context.Context, ir *claircore.IndexReport, opts *sbom.Options) ([]byte, error) {
	return m.sbomer.GetSBOM(ctx, ir, opts)
}

// Close closes the matcher.
func (m *matcherImpl) Close(ctx context.Context) error {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/backend/matcher.Close")
	err := errors.Join(m.vulnUpdater.Stop(), m.libVuln.Close(ctx))
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
	if m.readyWithVulns && !m.vulnUpdater.Initialized(ctx) {
		return errors.New("initial load for the vulnerability store is in progress")
	}
	return nil
}
