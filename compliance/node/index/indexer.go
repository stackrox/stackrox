package index

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/quay/claircore"
	ccindexer "github.com/quay/claircore/indexer"
	"github.com/quay/claircore/indexer/controller"
	"github.com/quay/claircore/rhel"
	rpm2 "github.com/quay/claircore/rpm"
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
)

var (
	// The layer carries a hardcoded digest, as it is exclusively used for passing
	// ClairCore checks, not for Scanners matching
	layerDigest = fmt.Sprintf("sha256:%s", strings.Repeat("a", 64))
	log         = logging.LoggerForModule()
)

// ClairCore is using zlog and needs separate configuration, as it's logging on debug level by default.
func configureClairCoreLogging() {
	l := zerolog.New(os.Stderr)
	l = l.Level(zerolog.InfoLevel)
	if logging.GetGlobalLogLevel().CapitalString() == "DEBUG" {
		l = l.Level(zerolog.DebugLevel)
	}
	zlog.Set(&l)
}

type NodeIndexerConfig struct {
	DisableAPI         bool
	API                string
	Repo2CPEMappingURL string
	Timeout            time.Duration
}

func NewNodeIndexerConfigFromEnv() *NodeIndexerConfig {
	configureClairCoreLogging()
	return &NodeIndexerConfig{
		DisableAPI:         false,
		API:                env.NodeIndexContainerAPI.Setting(),
		Repo2CPEMappingURL: env.NodeIndexMappingURL.Setting(),
		Timeout:            10 * time.Second,
	}
}

type localNodeIndexer struct {
	config *NodeIndexerConfig
}

// NewNodeIndexer creates a new node indexer
func NewNodeIndexer(config *NodeIndexerConfig) node.NodeIndexer {
	return &localNodeIndexer{config: config}
}

// GetIntervals returns the scanning intervals configured through env.
func (l *localNodeIndexer) GetIntervals() *utils.NodeScanIntervals {
	i := utils.NewNodeScanIntervalFromEnv()
	return &i
}

// IndexNode indexes a live fs.FS at the container mountpoint given in the basePath.
func (l *localNodeIndexer) IndexNode(ctx context.Context) (r *v4.IndexReport, err error) {
	report := &claircore.IndexReport{
		Packages:      map[string]*claircore.Package{},
		Environments:  map[string][]*claircore.Environment{},
		Distributions: map[string]*claircore.Distribution{},
		Repositories:  map[string]*claircore.Repository{},
		Files:         map[string]claircore.File{},
	}

	layer, err := constructLayer(ctx, layerDigest, env.NodeIndexHostPath.Setting())
	if err != nil {
		return nil, err
	}
	defer func() {
		if tmpErr := layer.Close(); tmpErr != nil {
			err = tmpErr
		}
	}()

	reps, err := runRepositoryScanner(ctx, l.config, layer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to run repository scanner")
	}

	pcks, err := runPackageScanner(ctx, layer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to run package scanner")
	}

	ir, err := coalesceReport(ctx, layerDigest, reps, pcks)
	if err != nil {
		return nil, errors.Wrap(err, "failed to coalesce report")
	}
	log.Debugf("Finished coalescing report. Report contains %d repositories with %d packages", len(ir.Repositories), len(ir.Packages))

	report = controller.MergeSR(report, []*claircore.IndexReport{ir})
	report.Success = true
	report.State = controller.IndexFinished.String()

	r, err = mappers.ToProtoV4IndexReport(report)
	if err != nil {
		return nil, errors.Wrap(err, "converting clair report to v4 report")
	}

	return
}

func coalesceReport(ctx context.Context, layerDigest string, reps []*claircore.Repository, pcks []*claircore.Package) (*claircore.IndexReport, error) {
	ch, err := claircore.ParseDigest(layerDigest)
	if err != nil {
		return nil, errors.Wrap(err, "parsing Clair Core digest")
	}

	la := &ccindexer.LayerArtifacts{
		Hash: ch,
	}
	la.Repos = append(la.Repos, reps...)
	la.Pkgs = append(la.Pkgs, pcks...)
	artifacts := []*ccindexer.LayerArtifacts{la}
	coal := new(rhel.Coalescer)

	ir, err := coal.Coalesce(ctx, artifacts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to coalesce report")
	}
	return ir, nil
}

func runPackageScanner(ctx context.Context, layer *claircore.Layer) ([]*claircore.Package, error) {
	rpm := rpm2.Scanner{}
	pck, err := rpm.Scan(ctx, layer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to invoke RPM scanner")
	}
	pck = filterPackages(ctx, pck)
	for i, p := range pck {
		p.ID = fmt.Sprintf("%d", i)
	}
	return pck, nil
}

// As we're only interested in the effective running RPM DB,
// we filter out packages from other DBs like rpm-ostree
func filterPackages(_ context.Context, pck []*claircore.Package) []*claircore.Package {
	var filtered []*claircore.Package
	for _, pkg := range pck {
		if pkg.PackageDB == "sqlite:usr/share/rpm" {
			filtered = append(filtered, pkg)
		}
	}
	return filtered
}

func createClient() (*http.Client, error) {
	clientCert, err := mtls.LeafCertificateFromFile()
	if err != nil {
		return nil, errors.Wrap(err, "obtaining client certificate")
	}
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{clientCert},
	}
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConf,
			Proxy:           proxy.FromConfig(),
		},
		Timeout: 30 * time.Second,
	}, nil
}

func runRepositoryScanner(ctx context.Context, cfg *NodeIndexerConfig, l *claircore.Layer) ([]*claircore.Repository, error) {
	c, err := createClient()
	if err != nil {
		return nil, errors.Wrap(err, "creating repository scanner http client - check TLS config")
	}
	sc := rhel.RepositoryScanner{}
	config := rhel.RepositoryScannerConfig{
		DisableAPI:         cfg.DisableAPI,
		API:                cfg.API,
		Repo2CPEMappingURL: cfg.Repo2CPEMappingURL,
		Timeout:            cfg.Timeout,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&config); err != nil {
		return nil, errors.Wrap(err, "failed to encode configuration")
	}
	if err := sc.Configure(ctx, json.NewDecoder(&buf).Decode, c); err != nil {
		return nil, errors.Wrap(err, "failed to configure repository scanner")
	}

	reps, err := sc.Scan(ctx, l)
	if err != nil {
		return nil, errors.Wrap(err, "failed to scan repositories")
	}
	for i, r := range reps {
		r.ID = fmt.Sprintf("%d", i)
	}
	return reps, nil
}

func constructLayer(ctx context.Context, digest string, hostPath string) (*claircore.Layer, error) {
	log.Debugf("Realizing mount path: %s", hostPath)
	desc := &claircore.LayerDescription{
		Digest:    digest,
		URI:       hostPath,
		MediaType: "application/vnd.claircore.filesystem",
	}

	l := claircore.Layer{}
	err := l.Init(ctx, desc, nil)
	return &l, errors.Wrap(err, "failed to init layer")
}
