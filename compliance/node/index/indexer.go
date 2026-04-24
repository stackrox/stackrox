package index

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/quay/claircore"
	ccindexer "github.com/quay/claircore/indexer"
	"github.com/quay/claircore/indexer/controller"
	"github.com/quay/claircore/osrelease"
	"github.com/quay/claircore/pkg/rhctag"
	"github.com/quay/claircore/rhel"
	"github.com/quay/claircore/rhel/rhcc"
	"github.com/quay/claircore/toolkit/types/cpe"
	"github.com/quay/zlog"
	"github.com/rs/zerolog"
	"github.com/stackrox/rox/compliance/node"
	"github.com/stackrox/rox/compliance/utils"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/scannerv4/mappers"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/urlfmt"
	pkgutils "github.com/stackrox/rox/pkg/utils"
	"go.uber.org/zap/zapcore"
)

const (
	layerMediaType = "application/vnd.claircore.filesystem"

	rhcosPackageDB = "sqlite:usr/share/rpm"

	// scannerDefinitionsRouteInSensor should be in sync with `scannerDefinitionsRoute` in sensor/sensor.go
	// Direct import is prohibited by import rules
	scannerDefinitionsRouteInSensor = "/scanner/definitions"
	sensorMappingsFile              = "repo2cpe"
)

var (
	log          = logging.LoggerForModule()
	zerologLevel = map[zapcore.Level]zerolog.Level{
		zapcore.DebugLevel: zerolog.DebugLevel,
		zapcore.InfoLevel:  zerolog.InfoLevel,
		zapcore.WarnLevel:  zerolog.WarnLevel,
		zapcore.ErrorLevel: zerolog.ErrorLevel,
		zapcore.PanicLevel: zerolog.PanicLevel,
		zapcore.FatalLevel: zerolog.FatalLevel,
	}

	// layerDigest is a dummy digest solely meant as a workaround to use Claircore.
	// Claircore indexing requires layers to have a digest, which is not stored,
	// so we use a fake one for all nodes.
	layerDigest   = fmt.Sprintf("sha256:%s", strings.Repeat("a", 64))
	ccLayerDigest = claircore.MustParseDigest(layerDigest)

	clientOnce       sync.Once
	defaultClient    *http.Client
	defaultClientErr error
)

func init() {
	// Default to info level.
	logLevel := zerolog.InfoLevel
	if level, ok := zerologLevel[logging.GetGlobalLogLevel()]; ok {
		logLevel = level
	}
	l := zerolog.New(os.Stderr).
		Level(logLevel)
	zlog.Set(&l)
}

func getDefaultClient() (*http.Client, error) {
	clientOnce.Do(func() {
		clientCert, err := mtls.LeafCertificateFromFile()
		if err != nil {
			defaultClientErr = errors.Wrap(err, "obtaining defaultClient certificate")
			return
		}
		defaultClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					// TODO: Should this always be set to true...?
					InsecureSkipVerify: true,
					Certificates:       []tls.Certificate{clientCert},
				},
				Proxy: proxy.FromConfig(),
			},
			Timeout: 30 * time.Second,
		}
	})
	return defaultClient, defaultClientErr
}

// NodeIndexerConfig represents Scanner V4 node indexer configuration parameters.
type NodeIndexerConfig struct {
	// HostPath is the mount point of the read-only host filesystem on the node.
	HostPath string
	// Client is the HTTP client used to reach out to external data sources.
	// If unset, a default which uses client-side TLS certificates is used.
	Client *http.Client
	// Repo2CPEMappingURL can be used to fetch the repo mapping file.
	// Consulting the mapping file is preferred over the Container API.
	Repo2CPEMappingURL string
	// Timeout controls the timeout for any remote API calls.
	Timeout time.Duration
	// PackageDBFilter removes irrelevant packages. For node scanning, we are
	// currently only interested in the RHCOS RPM database.
	// Filters out all packages whose packageDB does not match the filter.
	// Empty string corresponds to no filtering.
	PackageDBFilter string
}

// DefaultNodeIndexerConfig provides the default configuration for a node indexer.
func DefaultNodeIndexerConfig() NodeIndexerConfig {
	return NodeIndexerConfig{
		HostPath: env.NodeIndexHostPath.Setting(),
		// The default, mTLS-capable client will be used.
		Client:             nil,
		Repo2CPEMappingURL: buildMappingURL(),
		Timeout:            10 * time.Second,
		PackageDBFilter:    rhcosPackageDB,
	}
}

func buildMappingURL() string {
	if len(env.NodeIndexMappingURL.Setting()) > 0 {
		return urlfmt.FormatURL(env.NodeIndexMappingURL.Setting(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	}
	u := env.AdvertisedEndpoint.Setting() + scannerDefinitionsRouteInSensor + "?file=" + sensorMappingsFile
	return urlfmt.FormatURL(u, urlfmt.HTTPS, urlfmt.NoTrailingSlash)
}

type localNodeIndexer struct {
	cfg NodeIndexerConfig
}

// NewNodeIndexer creates a new node indexer.
func NewNodeIndexer(cfg NodeIndexerConfig) node.NodeIndexer {
	return &localNodeIndexer{cfg: cfg}
}

// GetIntervals returns the scanning intervals configured through env.
func (l *localNodeIndexer) GetIntervals() *utils.NodeScanIntervals {
	i := utils.NewNodeScanIntervalFromEnv()
	return &i
}

// IndexNode indexes a node at the configured host path mount.
func (l *localNodeIndexer) IndexNode(ctx context.Context) (*v4.IndexReport, error) {
	// claircore no longer returns an error if the host path does not exist.
	if _, err := os.Stat(l.cfg.HostPath); err != nil {
		return nil, errors.Wrapf(err, "host path %q does not exist", l.cfg.HostPath)
	}

	layer, err := layer(ctx, layerDigest, l.cfg.HostPath)
	if err != nil {
		return nil, err
	}
	defer pkgutils.IgnoreError(layer.Close)

	repos, err := runRepositoryScanner(ctx, l.cfg, layer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to run repository scanner")
	}

	pkgs, err := runPackageScanner(ctx, l.cfg.PackageDBFilter, layer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to run package scanner")
	}

	ccReport, err := runCoalescer(ctx, ccLayerDigest, repos, pkgs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to coalesce report")
	}
	log.Debugf("Finished coalescing report. Report contains %d repositories with %d packages", len(ccReport.Repositories), len(ccReport.Packages))

	rhcosRel, err := osRelease(ctx, l.cfg.HostPath)
	if err != nil {
		log.Debugf("Not adding RHCOS package to index report: %v", err)
	} else {
		arch := extractArch(ccReport, pkgs)
		addRHCOS(rhcosRel, arch, ccReport)
	}

	ccReport.Success = true
	ccReport.State = controller.IndexFinished.String()

	report, err := mappers.ToProtoV4IndexReport(ccReport)
	if err != nil {
		return nil, errors.Wrap(err, "converting clair report to v4 report")
	}

	return report, nil
}

func layer(ctx context.Context, digest string, hostPath string) (*claircore.Layer, error) {
	log.Debugf("Realizing mount path: %s", hostPath)
	if hostPath == "" {
		return nil, errors.New("host path is empty")
	}

	absoluteHostPath, err := filepath.Abs(hostPath)
	if err != nil {
		return nil, errors.Wrapf(err, "resolving absolute host path %q", hostPath)
	}
	hostURI := (&url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(absoluteHostPath),
	}).String()

	desc := &claircore.LayerDescription{
		Digest:    digest,
		URI:       hostURI,
		MediaType: layerMediaType,
	}

	l := &claircore.Layer{}
	err = l.Init(ctx, desc, nil)
	return l, errors.Wrap(err, "failed to init layer")
}

func runRepositoryScanner(ctx context.Context, cfg NodeIndexerConfig, l *claircore.Layer) ([]*claircore.Repository, error) {
	client := cfg.Client
	if client == nil {
		var err error
		client, err = getDefaultClient()
		if err != nil {
			return nil, errors.Wrap(err, "creating repository scanner http client - check TLS config")
		}
	}

	scanner := rhel.RepositoryScanner{}
	config := rhel.RepositoryScannerConfig{
		// Do not reach out to the Red Hat Container Catalog API.
		// We do *not* want to reach out to the internet for node scanning.
		DisableAPI:         true,
		Repo2CPEMappingURL: cfg.Repo2CPEMappingURL,
		Timeout:            cfg.Timeout,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&config); err != nil {
		return nil, errors.Wrap(err, "failed to encode configuration")
	}
	if err := scanner.Configure(ctx, json.NewDecoder(&buf).Decode, client); err != nil {
		return nil, errors.Wrap(err, "failed to configure repository scanner")
	}

	repos, err := scanner.Scan(ctx, l)
	if err != nil {
		return nil, errors.Wrap(err, "failed to scan repositories")
	}
	for i, r := range repos {
		r.ID = strconv.Itoa(i)
	}
	return repos, nil
}

func runPackageScanner(ctx context.Context, packageDBFilter string, layer *claircore.Layer) ([]*claircore.Package, error) {
	scanner := rhel.PackageScanner{}
	pkgs, err := scanner.Scan(ctx, layer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to invoke RHEL scanner")
	}

	// Filter out packages in which we are not interested.
	filtered := pkgs
	if packageDBFilter != "" {
		filtered = pkgs[:0]
		for _, pkg := range pkgs {
			if pkg.PackageDB == packageDBFilter {
				filtered = append(filtered, pkg)
			}
		}
	}
	for i, p := range filtered {
		p.ID = strconv.Itoa(i)
	}

	return filtered, nil
}

func runCoalescer(ctx context.Context, layerDigest claircore.Digest, repos []*claircore.Repository, pkgs []*claircore.Package) (*claircore.IndexReport, error) {
	la := &ccindexer.LayerArtifacts{
		Hash:  layerDigest,
		Repos: repos,
		Pkgs:  pkgs,
	}
	artifacts := []*ccindexer.LayerArtifacts{la}

	coal := rhel.Coalescer{}
	ir, err := coal.Coalesce(ctx, artifacts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to coalesce report")
	}

	return ir, nil
}

func validateOSRelease(osRel map[string]string) error {
	if variant := osRel["VARIANT_ID"]; variant != "coreos" {
		return fmt.Errorf("not RHCOS: VARIANT_ID=%q", variant)
	}
	if osRel["VERSION"] == "" {
		return errors.New("VERSION not found in os-release")
	}
	if osRel["OPENSHIFT_VERSION"] == "" {
		return errors.New("OPENSHIFT_VERSION not found in os-release")
	}
	if osRel["VERSION_ID"] == "" {
		return errors.New("VERSION_ID not found in os-release")
	}
	return nil
}

func parseOSRelease(ctx context.Context, hostPath string) (map[string]string, error) {
	var f *os.File
	for _, relPath := range []string{osrelease.Path, osrelease.FallbackPath} {
		path := filepath.Join(hostPath, relPath)
		var err error
		f, err = os.Open(path)
		if err != nil {
			continue
		}
		break
	}
	if f == nil {
		return nil, fmt.Errorf("os-release not found in %s", hostPath)
	}
	defer func() {
		_ = f.Close()
	}()
	return osrelease.Parse(ctx, f)
}

type rhcosRelease struct {
	version     string
	normVersion claircore.Version
	repoCPE     cpe.WFN
}

// osRelease opens, parse and validate the os release file, returning RHCOS
// version and repository CPE.
func osRelease(ctx context.Context, osRelPath string) (rhcosRelease, error) {
	osRel, err := parseOSRelease(ctx, osRelPath)
	if err != nil {
		return rhcosRelease{}, fmt.Errorf("failed to parse os-release: %w", err)
	}
	if err := validateOSRelease(osRel); err != nil {
		return rhcosRelease{}, fmt.Errorf("invalid os-release: %w", err)
	}

	rel := rhcosRelease{
		version: osRel["VERSION"],
	}

	// Set up the repository CPE.
	rhelMajor := strings.Split(osRel["VERSION_ID"], ".")[0]
	cpeStr := fmt.Sprintf("cpe:/a:redhat:openshift:%s::el%s", osRel["OPENSHIFT_VERSION"], rhelMajor)
	rel.repoCPE, err = cpe.Unbind(cpeStr)
	if err != nil {
		return rhcosRelease{}, fmt.Errorf("failed to parse RHCOS repository CPE: %w", err)
	}

	// Set up the normalized version.
	rhctagVersion, err := rhctag.Parse(rel.version)
	if err != nil {
		log.Warnf("Failed to parse RHCOS version %q: %v", rel.version, err)
		return rhcosRelease{}, fmt.Errorf("failed to parse RHCOS version %q: %w", rel.version, err)
	}
	minorStart := rhctagVersion.MinorStart()
	rel.normVersion = minorStart.Version(true)

	return rel, nil
}

func addRHCOS(rel rhcosRelease, arch string, report *claircore.IndexReport) {
	const (
		rhcosPkgID  = "rhcos-pkg"
		rhcosSrcID  = "rhcos-src"
		rhcosRepoID = "rhcos-repo"
	)

	srcPkg := &claircore.Package{
		ID:                rhcosSrcID,
		Name:              "rhcos",
		Version:           rel.version,
		Kind:              claircore.SOURCE,
		NormalizedVersion: rel.normVersion,
		Arch:              arch,
	}
	binPkg := &claircore.Package{
		ID:                rhcosPkgID,
		Name:              "rhcos",
		Version:           rel.version,
		Kind:              claircore.BINARY,
		NormalizedVersion: rel.normVersion,
		Source:            srcPkg,
		PackageDB:         "",
		Arch:              arch,
		RepositoryHint:    "rhcc",
	}
	repo := &claircore.Repository{
		ID:   rhcosRepoID,
		Name: rel.repoCPE.String(),
		Key:  rhcc.RepositoryKey,
		CPE:  rel.repoCPE,
	}

	if report.Packages == nil {
		report.Packages = make(map[string]*claircore.Package)
	}
	if report.Repositories == nil {
		report.Repositories = make(map[string]*claircore.Repository)
	}
	if report.Environments == nil {
		report.Environments = make(map[string][]*claircore.Environment)
	}

	report.Packages[srcPkg.ID] = srcPkg
	report.Packages[binPkg.ID] = binPkg
	report.Repositories[repo.ID] = repo
	report.Environments[binPkg.ID] = []*claircore.Environment{
		{
			PackageDB:     "",
			IntroducedIn:  ccLayerDigest,
			RepositoryIDs: []string{repo.ID},
		},
	}

	log.Debugf("Added RHCOS package: version=%s, cpe=%s", rel.version, rel.repoCPE.String())
}

// extractArch attempts to determine the RHCOS architecture from the current list
// of packages, failing open if no good guess is found, returning an empty string.
func extractArch(report *claircore.IndexReport, pkgs []*claircore.Package) string {
	for _, d := range report.Distributions {
		if d.Arch != "" && d.Arch != "noarch" {
			return d.Arch
		}
	}
	for _, p := range pkgs {
		if p.Arch != "" && p.Arch != "noarch" {
			return p.Arch
		}
	}
	return ""
}
