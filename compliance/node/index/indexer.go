package index

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/quay/claircore"
	ccindexer "github.com/quay/claircore/indexer"
	"github.com/quay/claircore/indexer/controller"
	"github.com/quay/claircore/rhel"
	"github.com/quay/claircore/rpm"
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
}

// DefaultNodeIndexerConfig provides the default configuration for a node indexer.
func DefaultNodeIndexerConfig() NodeIndexerConfig {
	return NodeIndexerConfig{
		HostPath: env.NodeIndexHostPath.Setting(),
		// The default, mTLS-capable client will be used.
		Client:             nil,
		Repo2CPEMappingURL: buildMappingURL(),
		Timeout:            10 * time.Second,
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
	layer, err := layer(ctx, layerDigest, l.cfg.HostPath)
	if err != nil {
		return nil, err
	}
	defer pkgutils.IgnoreError(layer.Close)

	repos, err := runRepositoryScanner(ctx, l.cfg, layer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to run repository scanner")
	}

	pkgs, err := runPackageScanner(ctx, layer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to run package scanner")
	}

	ccReport, err := runCoalescer(ctx, ccLayerDigest, repos, pkgs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to coalesce report")
	}
	log.Debugf("Finished coalescing report. Report contains %d repositories with %d packages", len(ccReport.Repositories), len(ccReport.Packages))

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
	desc := &claircore.LayerDescription{
		Digest:    digest,
		URI:       hostPath,
		MediaType: layerMediaType,
	}

	l := &claircore.Layer{}
	err := l.Init(ctx, desc, nil)
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

func runPackageScanner(ctx context.Context, layer *claircore.Layer) ([]*claircore.Package, error) {
	scanner := rpm.Scanner{}
	pkgs, err := scanner.Scan(ctx, layer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to invoke RPM scanner")
	}

	// Filter out packages in which we are not interested.
	// At this time, we are only interested in the RHCOS RPM database.
	filtered := pkgs[:0]
	for _, pkg := range pkgs {
		if pkg.PackageDB == rhcosPackageDB {
			filtered = append(filtered, pkg)
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
