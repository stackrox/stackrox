package index

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
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
	"github.com/quay/claircore/toolkit/types"
	"github.com/quay/claircore/toolkit/types/cpe"
	"github.com/stackrox/rox/compliance/node"
	"github.com/stackrox/rox/compliance/utils"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/scannerv4/mappers"
	"github.com/stackrox/rox/pkg/scannerv4/rhcos"
	"github.com/stackrox/rox/pkg/urlfmt"
	pkgutils "github.com/stackrox/rox/pkg/utils"
)

const (
	layerMediaType = "application/vnd.claircore.filesystem"

	// scannerDefinitionsRouteInSensor should be in sync with `scannerDefinitionsRoute` in sensor/sensor.go
	// Direct import is prohibited by import rules
	scannerDefinitionsRouteInSensor = "/scanner/definitions"
	sensorMappingsFile              = "repo2cpe"
)

var (
	log                        = logging.LoggerForModule()
	errNotRHCOS                = errors.New("not RHCOS")
	rhcosVersionCodenameRegexp = regexp.MustCompile(`^([0-9]+\.[0-9]+\.[0-9]{8}-[0-9]+) \(Plow\)$`)

	// rhcosPackageDBs are Claircore PackageDB names for RHCOS RPM databases.
	// RHEL 8 uses Berkeley DB; RHEL 9+ uses SQLite. ostree reports the same RPMs
	// under usr/share/rpm and usr/lib/sysimage/rpm-ostree-base-db.
	rhcosPackageDBs = []string{
		"sqlite:usr/share/rpm",
		"sqlite:usr/lib/sysimage/rpm",
		"bdb:usr/share/rpm",
		"bdb:usr/lib/sysimage/rpm-ostree-base-db",
	}

	// layerDigest is a dummy digest solely meant as a workaround to use Claircore.
	// Claircore indexing requires layers to have a digest, which is not stored,
	// so we use a fake one for all nodes.
	layerDigest   = fmt.Sprintf("sha256:%s", strings.Repeat("a", 64))
	ccLayerDigest = claircore.MustParseDigest(layerDigest)
)

// newDefaultClient configures an HTTP client that works with both internal
// mTLS endpoints (Sensor) and external publicly-trusted endpoints.
func newDefaultClient(mappingURL string) (*http.Client, error) {
	serverName, err := extractHostname(mappingURL)
	if err != nil {
		return nil, err
	}
	// SensorSubject identifies the expected StackRox service for cert verification
	// when the endpoint is Sensor (the default). ServerName is set to the actual
	// hostname to avoid defaulting to sensor.stackrox, which would fail hostname
	// verification for external endpoints.
	tlsConf, err := clientconn.TLSConfig(mtls.SensorSubject, clientconn.TLSConfigOptions{
		UseClientCert: clientconn.MustUseClientCert,
		ServerName:    serverName,
	})
	if err != nil {
		return nil, errors.Wrap(err, "creating TLS config")
	}
	// Clear gRPC ALPN protos to avoid HTTP/2 framing issues (see ROX-21861).
	tlsConf.NextProtos = nil
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConf,
			Proxy:           proxy.FromConfig(),
		},
		Timeout: 30 * time.Second,
	}, nil
}

func extractHostname(rawURL string) (string, error) {
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", errors.Wrapf(err, "parsing URL %q", rawURL)
	}
	return parsed.Hostname(), nil
}

// NodeIndexerConfig represents Scanner V4 node indexer configuration parameters.
type NodeIndexerConfig struct {
	// HostPath is the mount point of the read-only host filesystem on the node.
	HostPath string
	// OSReleasePath is the path where os-release can be found for RHCOS detection.
	// This may differ from HostPath when the RPM database is mounted separately.
	OSReleasePath string
	// Client is the HTTP client used to reach out to external data sources.
	// If unset, a default which uses client-side TLS certificates is used.
	Client *http.Client
	// Repo2CPEMappingURL can be used to fetch the repo mapping file.
	// Consulting the mapping file is preferred over the Container API.
	Repo2CPEMappingURL string
	// Repo2CPEMappingFile is a local path to the repo-to-CPE mapping file.
	// When set, leave Repo2CPEMappingURL empty: claircore still fetches a
	// non-empty URL on every call (its freshly-constructed rate limiter is
	// not treated as "already fetched"), so a non-empty URL would make a
	// network request on top of the seeded file.
	Repo2CPEMappingFile string
	// Timeout controls the timeout for any remote API calls.
	Timeout time.Duration
	// PackageDBFilter keeps RHCOS RPM databases and drops other package DBs
	// Claircore finds under the host index mount. Empty means no filtering.
	PackageDBFilter []string
}

// DefaultNodeIndexerConfig provides the default configuration for a node indexer.
func DefaultNodeIndexerConfig() NodeIndexerConfig {
	return NodeIndexerConfig{
		HostPath:      env.NodeIndexHostPath.Setting(),
		OSReleasePath: env.NodeIndexOSReleasePath.Setting(),
		// The default, mTLS-capable client will be used.
		Client:             nil,
		Repo2CPEMappingURL: buildMappingURL(),
		Timeout:            10 * time.Second,
		PackageDBFilter:    rhcosPackageDBs,
	}
}

func buildMappingURL() string {
	if len(env.NodeIndexMappingURL.Setting()) > 0 {
		return urlfmt.FormatURL(env.NodeIndexMappingURL.Setting(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	}
	u := env.SensorEndpointSetting() + scannerDefinitionsRouteInSensor + "?file=" + sensorMappingsFile
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
	cfg, err := l.mappingConfig(ctx)
	if err != nil {
		return nil, err
	}

	// claircore no longer returns an error if the host path does not exist.
	if _, err := os.Stat(cfg.HostPath); err != nil {
		return nil, errors.Wrapf(err, "host path %q does not exist", cfg.HostPath)
	}

	layer, err := layer(ctx, layerDigest, cfg.HostPath)
	if err != nil {
		return nil, err
	}
	defer pkgutils.IgnoreError(layer.Close)

	repos, err := runRepositoryScanner(ctx, cfg, layer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to run repository scanner")
	}

	pkgs, err := runPackageScanner(ctx, cfg.PackageDBFilter, layer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to run package scanner")
	}

	ccReport, err := runCoalescer(ctx, ccLayerDigest, repos, pkgs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to coalesce report")
	}
	log.Debugf("Finished coalescing report. Report contains %d repositories with %d packages", len(ccReport.Repositories), len(ccReport.Packages))

	rhcosRel, err := osRelease(ctx, cfg.OSReleasePath)
	if err != nil {
		log.Debugf("Not adding RHCOS package to index report: %v", err)
	} else {
		metadata, labelsPresent, err := readRHCOSImageMetadata(ctx, l.cfg.OSReleasePath)
		if err != nil {
			log.Warnf("Unable to read RHCOS labels.json, using os-release identity: %v", err)
		}
		if err != nil || !labelsPresent {
			metadata, err = rhcosIdentityFromOSRelease(rhcosRel.osRel)
		}
		if err != nil {
			log.Debugf("Not adding RHCOS package to index report: %v", err)
		} else {
			rhcosRel.imageName = metadata.name
			rhcosRel.repoCPE = *metadata.repoCPE
			arch := extractArch(ccReport, pkgs)
			addRHCOS(rhcosRel, arch, ccReport)
		}
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
		client, err = newDefaultClient(cfg.Repo2CPEMappingURL)
		if err != nil {
			return nil, errors.Wrap(err, "creating repository scanner http client")
		}
	}

	scanner := rhel.RepositoryScanner{}
	config := rhel.RepositoryScannerConfig{
		// Do not reach out to the Red Hat Container Catalog API.
		// We do *not* want to reach out to the internet for node scanning.
		DisableAPI:          true,
		Repo2CPEMappingURL:  cfg.Repo2CPEMappingURL,
		Repo2CPEMappingFile: cfg.Repo2CPEMappingFile,
		Timeout:             cfg.Timeout,
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

func runPackageScanner(ctx context.Context, packageDBFilter []string, layer *claircore.Layer) ([]*claircore.Package, error) {
	scanner := rhel.PackageScanner{}
	pkgs, err := scanner.Scan(ctx, layer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to invoke RHEL scanner")
	}

	filtered := filterPackages(pkgs, packageDBFilter)
	for i, p := range filtered {
		p.ID = strconv.Itoa(i)
	}

	return filtered, nil
}

// filterPackages keeps packages whose PackageDB is in packageDBFilter.
// ostree nodes expose the same RPMs in two PackageDBs, so duplicates
// collapse to one per name/version/arch/kind.
func filterPackages(pkgs []*claircore.Package, packageDBFilter []string) []*claircore.Package {
	if len(packageDBFilter) == 0 {
		return pkgs
	}
	type ident struct {
		name, version, arch string
		kind                types.PackageKind
	}
	out := make([]*claircore.Package, 0, len(pkgs))
	seen := make(map[ident]struct{}, len(pkgs))
	for _, pkg := range pkgs {
		if !slices.Contains(packageDBFilter, pkg.PackageDB) {
			continue
		}
		id := ident{pkg.Name, pkg.Version, pkg.Arch, pkg.Kind}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, pkg)
	}
	return out
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
	variant := osRel["VARIANT_ID"]
	if variant == "" {
		return fmt.Errorf("%w: VARIANT_ID not found in os-release", errNotRHCOS)
	}
	if variant != "coreos" {
		return fmt.Errorf("%w: VARIANT_ID=%q", errNotRHCOS, variant)
	}
	if osRel["IMAGE_VERSION"] == "" && osRel["VERSION"] == "" {
		return errors.New("IMAGE_VERSION and VERSION not found in os-release")
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
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("opening os-release at %q: %w", path, err)
		}
		break
	}
	if f == nil {
		return nil, fmt.Errorf("%w: os-release not found in %s", errNotRHCOS, hostPath)
	}
	defer func() {
		_ = f.Close()
	}()
	return osrelease.Parse(ctx, f)
}

type rhcosRelease struct {
	imageName   string
	version     string
	normVersion claircore.Version
	repoCPE     cpe.WFN
	osRel       map[string]string
}

// osRelease prefers IMAGE_VERSION for the RHCOS package version. When it is
// empty, it uses VERSION and removes the known Plow display suffix only from
// a complete numeric major.minor.YYYYMMDD-release value. Other formats remain
// unchanged. The normalized version is derived from the selected value, and
// the original os-release fields are returned unchanged.
func osRelease(ctx context.Context, osRelPath string) (rhcosRelease, error) {
	osRel, err := parseOSRelease(ctx, osRelPath)
	if err != nil {
		return rhcosRelease{}, fmt.Errorf("failed to parse os-release: %w", err)
	}
	if err := validateOSRelease(osRel); err != nil {
		return rhcosRelease{}, err
	}

	version := osRel["IMAGE_VERSION"]
	if version == "" {
		// Some CoreOS instances (e.g. 9.4) do not provide IMAGE_VERSION.
		version = rhcosVersionCodenameRegexp.ReplaceAllString(osRel["VERSION"], "${1}")
	}
	rel := rhcosRelease{
		version: version,
		osRel:   osRel,
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

// rhcosIdentityFromOSRelease derives the image name and repository CPE from
// VERSION_ID and OPENSHIFT_VERSION, returning an error for missing fields or an
// invalid CPE.
func rhcosIdentityFromOSRelease(osRel map[string]string) (rhcosImageMetadata, error) {
	if osRel["VERSION_ID"] == "" {
		return rhcosImageMetadata{}, errors.New("VERSION_ID not found in os-release")
	}
	if osRel["OPENSHIFT_VERSION"] == "" {
		return rhcosImageMetadata{}, errors.New("OPENSHIFT_VERSION not found in os-release")
	}
	rhelMajor, _, _ := strings.Cut(osRel["VERSION_ID"], ".")
	name := "openshift/ose-rhel-coreos-" + rhelMajor
	cpeStr := fmt.Sprintf("cpe:/a:redhat:openshift:%s::el%s", osRel["OPENSHIFT_VERSION"], rhelMajor)
	repoCPE, err := cpe.Unbind(cpeStr)
	if err != nil {
		return rhcosImageMetadata{}, fmt.Errorf("failed to parse RHCOS repository CPE: %w", err)
	}
	return rhcosImageMetadata{name: name, repoCPE: &repoCPE}, nil
}

type rhcosImageMetadata struct {
	name    string
	repoCPE *cpe.WFN
}

// readRHCOSImageMetadata reads the image name and CPE using ClairCore's
// embedded-label detector. hostPath must be the full host root (os-release
// mount), where /usr/share/buildinfo/labels.json is reachable. It returns
// present=false only when neither supported labels.json path exists. A present
// but incomplete or unreadable file returns an error. Package versions are not
// included in the returned metadata.
func readRHCOSImageMetadata(ctx context.Context, hostPath string) (rhcosImageMetadata, bool, error) {
	var metadata rhcosImageMetadata
	l, err := layer(ctx, layerDigest, hostPath)
	if err != nil {
		return metadata, false, err
	}
	defer pkgutils.IgnoreError(l.Close)
	sys, err := l.FS()
	if err != nil {
		return metadata, false, fmt.Errorf("opening RHCOS filesystem: %w", err)
	}
	labelsPresent := false
	for _, path := range []string{"root/buildinfo/labels.json", "usr/share/buildinfo/labels.json"} {
		if _, err := fs.Stat(sys, path); err == nil {
			labelsPresent = true
			break
		} else if !errors.Is(err, fs.ErrNotExist) {
			return metadata, false, fmt.Errorf("checking labels.json at %q: %w", path, err)
		}
	}
	if !labelsPresent {
		return metadata, false, nil
	}

	ecosystem := rhcc.NewEcosystem(ctx)
	scanners, err := ecosystem.PackageScanners(ctx)
	if err != nil {
		return metadata, true, err
	}
	var found bool
	for _, scanner := range scanners {
		if scanner.Name() != "rhel_package_container_detector" {
			continue
		}
		pkgs, err := scanner.Scan(ctx, l)
		if err != nil {
			return metadata, true, err
		}
		found = true
		for _, pkg := range pkgs {
			if pkg.Kind == types.BinaryPackage {
				metadata.name = pkg.Name
				break
			}
		}
		break
	}
	if !found {
		return metadata, true, errors.New("ClairCore RHEL container label detector not found")
	}
	repoScanners, err := ecosystem.RepositoryScanners(ctx)
	if err != nil {
		return metadata, true, err
	}
	for _, scanner := range repoScanners {
		if scanner.Name() != "rhel_repo_container_detector" {
			continue
		}
		repos, err := scanner.Scan(ctx, l)
		if err != nil {
			return metadata, true, err
		}
		if len(repos) > 0 {
			metadata.repoCPE = &repos[0].CPE
		}
		if metadata.name == "" {
			return metadata, true, errors.New("labels.json is missing a usable image name")
		}
		if metadata.repoCPE == nil {
			return metadata, true, errors.New("labels.json is missing a usable product CPE")
		}
		return metadata, true, nil
	}
	return metadata, true, errors.New("ClairCore RHEL container repository detector not found")
}

func addRHCOS(rel rhcosRelease, arch string, report *claircore.IndexReport) {
	// Keep the legacy name until Red Hat finishes migrating VEX records.
	addRHCOSPackage(rel, arch, "rhcos", rhcos.LegacyPackageID, rhcos.LegacySourceID, report)
	addRHCOSPackage(rel, arch, rel.imageName, rhcos.PackageID, rhcos.SourceID, report)
}

func addRHCOSPackage(rel rhcosRelease, arch, name, pkgID, srcID string, report *claircore.IndexReport) {
	const rhcosRepoID = "rhcos-repo"
	srcPkg := &claircore.Package{
		ID:                srcID,
		Name:              name,
		Version:           rel.version,
		Kind:              types.SourcePackage,
		NormalizedVersion: rel.normVersion,
		Arch:              arch,
	}
	binPkg := &claircore.Package{
		ID:                pkgID,
		Name:              name,
		Version:           rel.version,
		Kind:              types.BinaryPackage,
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

	log.Debugf("Added RHCOS package: name=%s, version=%s, cpe=%s", name, rel.version, rel.repoCPE.String())
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
